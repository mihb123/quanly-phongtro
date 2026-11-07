import { execFileSync } from 'node:child_process'

const DSN = process.env.POSTGRES_DSN
const TOKEN = process.env.THUCHI_TELEGRAM_BOT_TOKEN
const PORT = process.env.APP_PORT_DEV || process.env.APP_PORT || '9180'
const GROUP = process.env.E2E_LIVE_GROUP || ''
const EMAIL = process.env.E2E_LIVE_EMAIL || ''
const filters = process.argv.slice(2)
const N = Array.from({ length: 5 }, () => 'bcdfghjklmnpqrstvwxz'[Math.floor(Math.random() * 20)]).join('')

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
const lit = (v) => (v === null || v === undefined ? 'NULL' : `'${String(v).replace(/'/g, "''")}'`)
const exec = (sql) => execFileSync('psql', [DSN, '-X', '-At', '-v', 'ON_ERROR_STOP=1', '-c', sql]).toString().trim()
const q = (sql) => JSON.parse(exec(`select coalesce(json_agg(t), '[]') from (${sql}) t`))
const tele = (args) => JSON.parse(execFileSync('tele', args, { maxBuffer: 16 << 20 }).toString())
const vnd = (n) => `${String(n).replace(/\B(?=(\d{3})+(?!\d))/g, '.')}đ`
const oneLine = (s) => s.replace(/\n+/g, ' / ')

async function until(fn, label, timeout = 30000) {
  const end = Date.now() + timeout
  for (;;) {
    const v = await fn()
    if (v) return v
    if (Date.now() > end) throw new Error(`hết ${timeout / 1000}s chờ ${label}`)
    await sleep(1500)
  }
}
const eq = (a, b, what) => {
  if (JSON.stringify(a) !== JSON.stringify(b)) throw new Error(`${what}: ${JSON.stringify(a)} ≠ ${JSON.stringify(b)}`)
}

function chat(target) {
  const mine = new Set()
  const messages = (limit = 15) => tele(['get', '-u', target, '-l', String(limit), '--max-age', '0', '-r']).messages
  return {
    send(text, replyTo) {
      const out = tele(['send', '-u', target, '-m', text, ...(replyTo ? ['-r', String(replyTo)] : [])])
      if (out.action !== 'sent') throw new Error(`tele send: ${JSON.stringify(out)}`)
      mine.add(out.message.id)
      return out.message.id
    },
    reply: (after, re, timeout) => until(() => messages().find((m) => m.id > after && !mine.has(m.id) && re.test(m.text)), `bot trả lời khớp ${re}`, timeout),
    message: (id, re) => until(() => messages(100).find((m) => m.id === id && re.test(m.text)), `tin ${id} của bot khớp ${re}`),
    botMessagesAfter: (after) => messages().filter((m) => m.id > after && !mine.has(m.id)),
  }
}

const results = []
async function step(name, fn) {
  if (filters.length && !filters.some((f) => name.toLowerCase().includes(f.toLowerCase()))) return
  try {
    const note = await fn()
    results.push(['PASS', name, note ?? ''])
  } catch (e) {
    results.push(['FAIL', name, e.message])
  }
  const [s, n, d] = results.at(-1)
  console.log(`${s === 'PASS' ? '\x1b[32m✔' : '\x1b[31m✘'} ${n}\x1b[0m${d ? `\n    ${d}` : ''}`)
}

async function preflight() {
  if (!DSN || !TOKEN) throw new Error('cần POSTGRES_DSN và THUCHI_TELEGRAM_BOT_TOKEN (chạy qua `make e2e-thuchi-telegram`, Makefile nạp .env)')
  const health = await fetch(`http://localhost:${PORT}/health`).catch(() => null)
  if (!health?.ok) throw new Error(`server dev không chạy ở :${PORT} (make dev)`)
  const me = tele(['status']).account
  const bot = (await (await fetch(`https://api.telegram.org/bot${TOKEN}/getMe`)).json()).result
  const hook = (await (await fetch(`https://api.telegram.org/bot${TOKEN}/getWebhookInfo`)).json()).result
  let owner = q(`select owner_id from thuchi_telegram_links where telegram_user_id = ${Number(me.id)}`)[0]?.owner_id
  if (!owner) {
    if (!EMAIL) throw new Error(`tài khoản tele @${me.username} chưa kết nối bot: đặt E2E_LIVE_EMAIL=<email chủ trọ dev> để script tự kết nối`)
    owner = q(`select id from users where email = ${lit(EMAIL)}`)[0]?.id
    if (!owner) throw new Error(`không có user ${EMAIL} trong DB dev`)
  }
  return { me, bot: bot.username, readsGroup: bot.can_read_all_group_messages, mode: hook.url ? `webhook ${hook.url}` : 'long polling', owner }
}

const ctx = await preflight().catch((e) => {
  console.error(`\x1b[31m${e.message}\x1b[0m`)
  process.exit(1)
})
const { owner } = ctx
const dm = chat(`@${ctx.bot}`)
const grp = GROUP ? chat(GROUP) : null
const setEnabled = (on) => exec(`insert into plugin_settings (manager_id, plugin_id, enabled) values (${lit(owner)}, 'thu-chi', ${on}) on conflict (manager_id, plugin_id) do update set enabled = excluded.enabled, updated_at = now()`)
const newCode = () => {
  exec(`delete from thuchi_telegram_links where owner_id = ${lit(owner)} and telegram_user_id is null`)
  const code = Array.from({ length: 8 }, () => 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789'[Math.floor(Math.random() * 32)]).join('')
  exec(`insert into thuchi_telegram_links (owner_id, code, code_expires_at) values (${lit(owner)}, ${lit(code)}, now() + interval '15 minutes')`)
  return code
}
const linkRow = () => q(`select id, default_account_id from thuchi_telegram_links where telegram_user_id = ${Number(ctx.me.id)} and owner_id = ${lit(owner)}`)[0]
const tx = (merchant) =>
  q(`select t.amount, t.direction, t.deleted_at, t.needs_review, c.key cat, a.name acct, to_char(t.occurred_at at time zone 'Asia/Ho_Chi_Minh', 'YYYY-MM-DD') d, t.sources
     from thuchi_transactions t left join thuchi_categories c on c.id = t.category_id left join thuchi_accounts a on a.id = t.account_id
     where t.owner_id = ${lit(owner)} and t.merchant = ${lit(merchant)} order by t.created_at desc`)
const waitTx = (merchant, pred = () => true, label = merchant) => until(() => tx(merchant).find(pred), `giao dịch ${label}`)
const vnDay = (offset = 0) => q(`select to_char((now() at time zone 'Asia/Ho_Chi_Minh')::date + ${offset}, 'YYYY-MM-DD') d`)[0].d

console.log(`Bot @${ctx.bot} (${ctx.mode}, đọc tin nhóm: ${ctx.readsGroup}) · tele @${ctx.me.username} · hậu tố ${N}${GROUP ? ` · nhóm ${GROUP}` : ' · bỏ qua nhóm (đặt E2E_LIVE_GROUP)'}\n`)

setEnabled(true)
await step('kết nối: /start <mã> (nếu chưa kết nối), /start khi đã kết nối → hướng dẫn', async () => {
  if (!linkRow()) {
    const id = dm.send(`/start ${newCode()}`)
    await dm.reply(id, /Đã kết nối với Thu Chi/)
  }
  const id = dm.send('/start')
  const r = await dm.reply(id, /Ghi thu chi bằng tin nhắn/)
  if (/Chào bạn/.test(r.text)) throw new Error('đã kết nối mà vẫn chào như người lạ')
})

let entry
const tra = `Trà đá ${N}`
await step(`"${tra.toLowerCase()} 15k" → Tiền mặt, Ăn uống, bot trả lời`, async () => {
  const id = dm.send(`${tra.toLowerCase()} 15k`)
  entry = await dm.reply(id, new RegExp(tra))
  const t = await waitTx(tra)
  eq([t.amount, t.direction, t.cat, t.acct, t.sources], [15000, 'out', 'an_uong', 'Tiền mặt', ['manual']], 'giao dịch')
  return oneLine(entry.text)
})
await step('trả lời "20k" → sửa số tiền, tin bot sửa theo', async () => {
  dm.send('20k', entry.id)
  await waitTx(tra, (t) => t.amount === 20000, '20k')
  return oneLine((await dm.message(entry.id, /20\.000đ/)).text)
})
await step('nhiều khoản 1 tin (dấu phẩy + xuống dòng, "+" là khoản thu)', async () => {
  const id = dm.send(`ăn trưa ${N} 45k, gửi xe ${N} 5k\n+2tr lương ${N}`)
  const r = await dm.reply(id, /Đã ghi 3 khoản/)
  if (!r.text.includes(`chi ${vnd(50000)} · thu ${vnd(2000000)}`)) throw new Error(r.text)
  const got = [`Ăn trưa ${N}`, `Gửi xe ${N}`, `Lương ${N}`].map((m) => tx(m)[0]).map((t) => [t?.amount, t?.direction, t?.cat])
  eq(got, [[45000, 'out', 'an_uong'], [5000, 'out', 'di_lai'], [2000000, 'in', 'luong']], 'các khoản')
  return r.text.split('\n')[0]
})
await step('ngày trong tin: "hôm qua …" và "20/10 …"', async () => {
  let id = dm.send(`hôm qua grab ${N} 32k`)
  await dm.reply(id, new RegExp(`Grab ${N}`))
  eq((await waitTx(`Grab ${N}`)).d, vnDay(-1), 'hôm qua')
  id = dm.send(`20/10 tiền điện ${N} 450k`)
  const r = await dm.reply(id, new RegExp(`Tiền điện ${N}`))
  const t = await waitTx(`Tiền điện ${N}`)
  eq([t.d.slice(5), t.cat, t.amount], ['10-20', 'hoa_don', 450000], '20/10')
  if (!r.text.includes('· 20/10')) throw new Error(r.text)
})
await step('cách viết số tiền: "30" cuối dòng = 30.000, "1tr2", "+300k thanh lý" là khoản thu', async () => {
  const cases = [
    [`ăn sáng ${N} 30`, `Ăn sáng ${N}`, 30000, 'out', 'an_uong'],
    [`tiền nhà ${N} 1tr2`, `Tiền nhà ${N}`, 1200000, 'out', 'nha_cua'],
    [`+300k thanh lý ${N}`, `Thanh lý ${N}`, 300000, 'in', 'khac_thu'],
  ]
  for (const [text, merchant, amount, dir, cat] of cases) {
    const id = dm.send(text)
    await dm.reply(id, new RegExp(merchant))
    const t = await waitTx(merchant)
    eq([t.amount, t.direction, t.cat], [amount, dir, cat], text)
  }
})
let unknown
const unknownName = N.charAt(0).toUpperCase() + N.slice(1)
await step(`mô tả lạ "${N} 50k" → ❓ chưa phân loại + bảng chọn`, async () => {
  const id = dm.send(`${N} 50k`)
  unknown = await dm.reply(id, new RegExp(unknownName))
  const t = await waitTx(unknownName)
  eq([t.amount, t.cat, t.needs_review], [50000, null, true], 'giao dịch')
  if (!/Chọn danh mục/.test(unknown.text)) throw new Error(unknown.text)
})
await step('trả lời "mua sắm" → đổi danh mục + học luật; lần sau tự phân loại', async () => {
  dm.send('mua sắm', unknown.id)
  await waitTx(unknownName, (t) => t.cat === 'mua_sam', 'Mua sắm')
  eq(q(`select r.origin, c.key from thuchi_rules r join thuchi_categories c on c.id = r.category_id where r.owner_id = ${lit(owner)} and lower(r.pattern) = ${lit(N)}`), [{ origin: 'learned', key: 'mua_sam' }], 'luật')
  await dm.message(unknown.id, /Đã nhớ/)
  const id = dm.send(`${N} 70k`)
  await dm.reply(id, new RegExp(unknownName))
  await waitTx(unknownName, (t) => t.amount === 70000 && t.cat === 'mua_sam', 'áp luật đã học')
})
await step('trả lời câu lạ → bot hỏi lại; trả lời "xoá" → xoá giao dịch', async () => {
  const id = dm.send('abc', entry.id)
  await dm.reply(id, /Mình chưa hiểu/)
  dm.send('xoá', entry.id)
  await waitTx(tra, (t) => t.deleted_at, 'bị xoá')
  await dm.message(entry.id, /đã xoá/)
})
await step('tin không có số tiền → nhắc cách nhắn; /huongdan', async () => {
  let id = dm.send(`chào cả nhà ${N}`)
  await dm.reply(id, /chưa thấy số tiền/)
  id = dm.send('/huongdan')
  await dm.reply(id, /Ghi thu chi bằng tin nhắn/)
})
const totals = () =>
  q(`select coalesce(sum(t.amount) filter (where t.direction = 'out'), 0)::bigint spend, coalesce(sum(t.amount) filter (where t.direction = 'in'), 0)::bigint income
     from thuchi_transactions t left join thuchi_accounts a on a.id = t.account_id
     where t.owner_id = ${lit(owner)} and t.deleted_at is null and not t.excluded and t.kind = 'normal' and coalesce(a.include_in_stats, true)
       and t.occurred_at >= date_trunc('month', now() at time zone 'Asia/Ho_Chi_Minh') at time zone 'Asia/Ho_Chi_Minh'`)[0]
await step('/tong → tổng chi/thu tháng khớp DB', async () => {
  const s = totals()
  const id = dm.send('/tong')
  const r = await dm.reply(id, /Tháng \d{2}\/\d{4}/)
  if (!r.text.includes(`Chi: ${vnd(s.spend)} · Thu: ${vnd(s.income)}`)) throw new Error(`cần Chi ${vnd(s.spend)} · Thu ${vnd(s.income)}: ${oneLine(r.text)}`)
  return oneLine(r.text)
})
await step('tắt tiện ích → bot báo đang tắt, không ghi; bật lại ghi bình thường', async () => {
  setEnabled(false)
  try {
    let id = dm.send(`bánh mì ${N} 20k`)
    await dm.reply(id, /đang tắt/)
    id = dm.send('/tong')
    await dm.reply(id, /đang tắt/)
    eq(tx(`Bánh mì ${N}`).length, 0, 'không ghi khi tắt')
  } finally {
    setEnabled(true)
  }
  const id = dm.send(`bánh mì ${N} 25k`)
  await dm.reply(id, new RegExp(`Bánh mì ${N}`))
  await waitTx(`Bánh mì ${N}`, (t) => t.amount === 25000)
})

if (grp) {
  let inGroup = false
  await step('nhóm: bot có trong nhóm và trả lời lệnh /huongdan@bot', async () => {
    const id = grp.send(`/huongdan@${ctx.bot}`)
    await grp.reply(id, /Ghi thu chi bằng tin nhắn/, 20000).catch(() => {
      throw new Error(`bot không trả lời trong nhóm ${GROUP}: thêm @${ctx.bot} vào nhóm (privacy mode tắt thì phải xoá rồi thêm lại bot)`)
    })
    inGroup = true
  })
  if (inGroup) {
    let groupEntry
    await step('nhóm: tin thường "cơm tấm … 35k" được ghi (privacy mode tắt), bot trả lời trong nhóm', async () => {
      if (!ctx.readsGroup) throw new Error('bot đang bật privacy mode: không đọc được tin thường trong nhóm')
      const id = grp.send(`cơm tấm ${N} 35k`)
      await grp.reply(id, new RegExp(`Cơm tấm ${N}`))
      eq((await waitTx(`Cơm tấm ${N}`)).amount, 35000, 'số tiền')
    })
    await step('nhóm: "@bot bún chả … 40k" bỏ @mention rồi ghi; trả lời "45k" sửa số tiền', async () => {
      const id = grp.send(`@${ctx.bot} bún chả ${N} 40k`)
      groupEntry = await grp.reply(id, new RegExp(`Bún chả ${N}`))
      eq((await waitTx(`Bún chả ${N}`)).amount, 40000, 'số tiền')
      grp.send('45k', groupEntry.id)
      await waitTx(`Bún chả ${N}`, (t) => t.amount === 45000, '45k')
      await grp.message(groupEntry.id, /45\.000đ/)
    })
    await step('nhóm: tin không có số tiền và /tong@bot_khac bị im lặng; /tong@bot trả tổng tháng', async () => {
      const first = grp.send(`alo ${N}`)
      grp.send('/tong@bot_khac_e2e')
      const id = grp.send(`/tong@${ctx.bot}`)
      const s = totals()
      const r = await grp.reply(id, /Tháng \d{2}\/\d{4}/)
      if (!r.text.includes(`Chi: ${vnd(s.spend)} · Thu: ${vnd(s.income)}`)) throw new Error(oneLine(r.text))
      const extra = grp.botMessagesAfter(first).filter((m) => m.id !== r.id)
      eq(extra.map((m) => m.text), [], 'bot không được trả lời 2 tin trên')
    })
  }
}

await step(`ngắt kết nối → bot nhắc kết nối, không ghi; /link <mã>${grp ? ' trong nhóm' : ''} → kết nối lại`, async () => {
  const before = linkRow()
  exec(`delete from thuchi_telegram_links where id = ${lit(before.id)}`)
  let id = dm.send(`bánh ${N} 20k`)
  await dm.reply(id, /chưa biết bạn dùng tài khoản Thu Chi nào/)
  eq(tx(`Bánh ${N}`).length, 0, 'không ghi khi chưa kết nối')
  const code = newCode()
  const target = grp ?? dm
  id = target.send(`/link ${code.toLowerCase()}`)
  await target.reply(id, /Đã kết nối với Thu Chi/)
  const after = await until(linkRow, 'liên kết mới')
  if (before.default_account_id) exec(`update thuchi_telegram_links set default_account_id = ${lit(before.default_account_id)} where id = ${lit(after.id)}`)
})

const failed = results.filter((r) => r[0] === 'FAIL').length
console.log(`\n${results.length - failed}/${results.length} PASS · dữ liệu test có hậu tố "${N}" trong sổ thu chi của chủ trọ ${owner}`)
process.exit(failed ? 1 : 0)
