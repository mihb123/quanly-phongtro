import assert from 'node:assert/strict'
import { randomUUID } from 'node:crypto'
import { after, before, describe } from 'node:test'
import { closeAll, newSession, step } from '../lib/app.mjs'
import { TELEGRAM_FAKE_URL } from '../lib/config.mjs'
import { account, execSql, row, rows, value, waitFor } from '../lib/db.mjs'
import { FAKE_ALLOWED_GROUP, FAKE_BOT_USERNAME, button, buttons, fakeTelegramClient } from '../lib/telegram-fake.mjs'

const vnd = (n) => `${String(n).replace(/\B(?=(\d{3})+(?!\d))/g, '.')}đ`
const randomId = () => 710000000 + Math.floor(Math.random() * 1e8)
const vnDate = (d) => new Intl.DateTimeFormat('en-GB', { timeZone: 'Asia/Ho_Chi_Minh', day: '2-digit', month: '2-digit' }).format(d)

describe('Bot Telegram (Bot API giả): lệnh, nhiều khoản, nút bấm, trả lời, sửa tin, nhóm, tài khoản ghi vào, ngắt kết nối', () => {
  let s
  let s2
  const tg = fakeTelegramClient(TELEGRAM_FAKE_URL)
  const user = { id: randomId(), name: 'Lan', username: 'lan_e2e' }
  const user2 = { id: user.id + 1, name: 'Hùng', username: 'hung_e2e' }
  const stranger = { id: user.id + 2, name: 'Khách', username: 'khach_e2e' }
  const group = { id: FAKE_ALLOWED_GROUP, type: 'supergroup', title: 'Nhóm thu chi E2E' }
  const otherGroup = { id: FAKE_ALLOWED_GROUP - 1, type: 'supergroup', title: 'Nhóm không được phép' }

  before(async () => {
    s = await newSession({ prefix: 'tgflow' })
  })
  after(closeAll)

  const chatOf = (opts) => opts.chat?.id ?? (opts.from ?? user).id
  const say = (text, opts = {}) => tg.message({ from: opts.from ?? user, chat: opts.chat, text, replyTo: opts.replyTo, messageId: opts.messageId })
  const ask = async (text, opts = {}) => {
    const id = await say(text, opts)
    return { id, reply: await tg.replyTo(chatOf(opts), id) }
  }
  const drain = (opts = {}) => ask('/huongdan', opts)
  const repliesTo = async (chatId, messageId) => (await tg.sent(chatId)).filter((m) => m.reply_to === messageId)
  const waitLatest = (chatId, botMessageId, pred, message) => waitFor(async () => {
    const m = await tg.latest(chatId, botMessageId)
    return m && pred(m) ? m : null
  }, { message: `tin ${botMessageId} của bot: ${message}` })
  const press = async (opts, msg, re) => {
    const b = button(msg, re)
    assert.ok(b, `không có nút ${re} trong ${JSON.stringify(buttons(msg).map((x) => x.text))}`)
    const id = await tg.press({ from: opts.from ?? user, chat: opts.chat, messageId: msg.message_id, data: b.callback_data })
    return tg.answer(id)
  }
  const recordTxs = async (chatId, messageId) => {
    await waitFor(() => row`select 1 as ok from thuchi_telegram_messages where chat_id = ${chatId} and message_id = ${messageId}`, { message: `bản ghi tin ${messageId}` })
    return rows`select t.*, c.key as cat_key, c.name as cat_name from thuchi_telegram_messages m
      cross join lateral jsonb_array_elements_text(m.transaction_ids) with ordinality x(id, ord)
      join thuchi_transactions t on t.id = x.id::uuid
      left join thuchi_categories c on c.id = t.category_id
      where m.chat_id = ${chatId} and m.message_id = ${messageId} order by x.ord`
  }
  const cashBalance = async (owner = s.owner) => Number((await account.byName(owner, 'Tiền mặt')).balance)
  const pendingCode = (owner) => waitFor(() => value`select code from thuchi_telegram_links where owner_id = ${owner} and telegram_user_id is null and code is not null`, { message: 'mã kết nối' })

  step('chưa kết nối: /start chào + hướng dẫn, /link mã sai báo lỗi, chat riêng luôn được nhắc, nhóm chỉ nhắc khi có số tiền (1 lần)', async () => {
    let r = await ask('/start')
    assert.match(r.reply.text, /Chào bạn!/)
    assert.match(r.reply.text, /Ghi thu chi bằng tin nhắn/)
    r = await ask('/link SAIMA123')
    assert.match(r.reply.text, /Mã kết nối không đúng hoặc đã hết hạn/)
    r = await ask('xin chào')
    assert.match(r.reply.text, /chưa biết bạn dùng tài khoản Thu Chi nào/)
    r = await ask('/lenhla')
    assert.match(r.reply.text, /Ghi thu chi bằng tin nhắn/, 'lệnh lạ trong chat riêng → hướng dẫn')

    const hello = await say('xin chào cả nhà', { from: stranger, chat: group })
    r = await ask('bánh mì 20k', { from: stranger, chat: group })
    assert.match(r.reply.text, /Khách ơi, mình chưa biết bạn/)
    const again = await say('bánh mì 25k', { from: stranger, chat: group })
    await drain({ from: stranger, chat: group })
    assert.deepEqual(await repliesTo(group.id, hello), [], 'tin không có số tiền trong nhóm → im lặng')
    assert.deepEqual(await repliesTo(group.id, again), [], 'nhắc kết nối trong nhóm tối đa 1 lần / 6 giờ')
    assert.deepEqual(await rows`select id from thuchi_transactions where merchant = 'Bánh mì' and owner_id = ${s.owner}`, [])
  })

  step('mã hết hạn bị từ chối; /link <mã> trong nhóm được phép thì kết nối, mã chỉ dùng 1 lần, app tự cập nhật', async () => {
    await s.settings('Telegram')
    await s.click('Kết nối Telegram')
    await s.expectText('Mở Telegram, bấm Start')
    const expired = await pendingCode(s.owner)
    await execSql`update thuchi_telegram_links set code_expires_at = now() - interval '1 minute' where code = ${expired}`
    let r = await ask(`/start ${expired}`)
    assert.match(r.reply.text, /Mã kết nối không đúng hoặc đã hết hạn/)

    const created = await s.api('POST', '/plugins/thu-chi/telegram/link')
    assert.equal(created.status, 201, JSON.stringify(created.body))
    const code = await pendingCode(s.owner)
    assert.notEqual(code, expired)
    assert.equal(await value`select count(*)::int from thuchi_telegram_links where code = ${expired}`, 0, 'tạo mã mới xoá mã cũ chưa dùng')

    r = await ask(`/link ${code.toLowerCase()}`, { chat: group })
    assert.match(r.reply.text, /Đã kết nối với Thu Chi/)
    const link = await row`select telegram_user_id, telegram_username, telegram_name, code from thuchi_telegram_links where owner_id = ${s.owner} and telegram_user_id is not null`
    assert.equal(Number(link.telegram_user_id), user.id)
    assert.equal(link.telegram_name, 'Lan')
    assert.equal(link.code, null)

    r = await ask(`/link ${code}`, { from: user2 })
    assert.match(r.reply.text, /Mã kết nối không đúng hoặc đã hết hạn/, 'mã đã dùng')
    await s.expectText('@lan_e2e', { timeout: 20000 })
  })

  step('nhiều khoản 1 tin (xuống dòng + dấu phẩy), dấu + là khoản thu, tự phân loại, số dư Tiền mặt', async () => {
    const before = await cashBalance()
    const r = await ask('ăn trưa 45k, gửi xe 5k\n+2tr lương')
    assert.match(r.reply.text, new RegExp(`Đã ghi 3 khoản · chi ${vnd(50_000)} · thu ${vnd(2_000_000)}`))
    const txs = await recordTxs(user.id, r.id)
    assert.deepEqual(txs.map((t) => [t.merchant, Number(t.amount), t.direction, t.cat_key]), [
      ['Ăn trưa', 45_000, 'out', 'an_uong'],
      ['Gửi xe', 5_000, 'out', 'di_lai'],
      ['Lương', 2_000_000, 'in', 'luong'],
    ])
    assert.ok(txs.every((t) => t.needs_review === false && JSON.stringify(t.sources) === '["manual"]'))
    assert.equal(await cashBalance(), before + 2_000_000 - 50_000)
    assert.equal(buttons(r.reply).filter((b) => b.callback_data.startsWith('p:')).length, 3, 'mỗi khoản 1 nút ✏️')
    assert.equal(buttons(r.reply).filter((b) => b.callback_data.startsWith('d:')).length, 3, 'mỗi khoản 1 nút 🗑')
    await s.tab('Giao dịch')
    await s.sync()
    await s.expectText('Gửi xe')
  })

  step('"hôm qua grab 32k" → giao dịch ngày hôm qua, tin trả lời ghi ngày', async () => {
    const yesterday = new Date(Date.now() - 86_400_000)
    const r = await ask('hôm qua grab 32k')
    const [t] = await recordTxs(user.id, r.id)
    assert.equal(Number(t.amount), 32_000)
    assert.equal(await value`select (occurred_at at time zone 'Asia/Ho_Chi_Minh')::date = (now() at time zone 'Asia/Ho_Chi_Minh')::date - 1 from thuchi_transactions where id = ${t.id}`, true)
    assert.match(r.reply.text, new RegExp(`· ${vnDate(yesterday)}`))
  })

  step('mô tả lạ → ❓ + bảng chọn danh mục; bấm chọn → học luật, lần sau tự phân loại', async () => {
    const r = await ask('zuvok 50k')
    let [t] = await recordTxs(user.id, r.id)
    assert.equal(t.category_id, null)
    assert.equal(t.needs_review, true)
    assert.match(r.reply.text, /❓ <b>Zuvok<\/b>/)
    assert.match(r.reply.text, /Chọn danh mục, mình sẽ nhớ cho lần sau/)
    assert.ok(!button(r.reply, /Phí ngân hàng/), 'không gợi ý Phí ngân hàng')
    assert.ok(button(r.reply, /Là khoản thu/))

    const ans = await press({}, r.reply, /Mua sắm$/)
    assert.equal(ans.text, 'Đã đổi sang Mua sắm')
    ;[t] = await recordTxs(user.id, r.id)
    assert.equal(t.cat_key, 'mua_sam')
    assert.equal(t.needs_review, false)
    const rule = await row`select r.match, r.origin, c.key from thuchi_rules r join thuchi_categories c on c.id = r.category_id where r.owner_id = ${s.owner} and lower(r.pattern) = 'zuvok' and r.deleted_at is null`
    assert.deepEqual(rule, { match: 'merchant', origin: 'learned', key: 'mua_sam' })
    const edited = await waitLatest(user.id, r.reply.message_id, (m) => m.edited && /Đã nhớ/.test(m.text), 'đã đổi danh mục')
    assert.match(edited.text, /✅ <b>Zuvok<\/b>/)
    assert.match(edited.text, /Mua sắm · Tiền mặt/)
    assert.ok(button(edited, /Đổi danh mục/) && button(edited, /Xoá/))

    const again = await ask('zuvok 70k')
    const [t2] = await recordTxs(user.id, again.id)
    assert.equal(t2.cat_key, 'mua_sam', 'áp luật đã học')
    assert.match(again.reply.text, /✅ <b>Zuvok<\/b>/)
    assert.ok(Number(await value`select hits from thuchi_rules where owner_id = ${s.owner} and lower(pattern) = 'zuvok'`) >= 1)
  })

  step('nút ✏️ → bảng danh mục chi (✓ danh mục hiện tại), ⇄ sang danh mục thu đổi chiều giao dịch, « Xong quay lại', async () => {
    const before = await cashBalance()
    const r = await ask('phở 40k')
    const botId = r.reply.message_id
    await press({}, r.reply, /Đổi danh mục/)
    let m = await waitLatest(user.id, botId, (x) => !!button(x, /Là khoản thu/), 'bảng danh mục chi')
    assert.ok(button(m, /^✓ .*Ăn uống$/), 'đánh dấu danh mục hiện tại')
    assert.ok(!button(m, /Lương/))

    await press({}, m, /Là khoản thu/)
    m = await waitLatest(user.id, botId, (x) => !!button(x, /Là khoản chi/), 'bảng danh mục thu')
    assert.ok(button(m, /Lương/) && button(m, /Được chuyển/))
    const ans = await press({}, m, /Được chuyển$/)
    assert.equal(ans.text, 'Đã đổi sang Được chuyển')
    const [t] = await recordTxs(user.id, r.id)
    assert.equal(t.direction, 'in')
    assert.equal(t.cat_key, 'duoc_chuyen')
    m = await waitLatest(user.id, botId, (x) => /\+40\.000đ/.test(x.text), 'hiện khoản thu')
    assert.equal(await cashBalance(), before + 40_000)

    await press({}, m, /Đổi danh mục/)
    m = await waitLatest(user.id, botId, (x) => !!button(x, /Xong/), 'mở lại bảng chọn')
    await press({}, m, /Xong/)
    m = await waitLatest(user.id, botId, (x) => !!button(x, /Đổi danh mục/) && !button(x, /Xong/), 'đóng bảng chọn')
    assert.equal((await recordTxs(user.id, r.id))[0].cat_key, 'duoc_chuyen', '« Xong không đổi gì')
  })

  step('nút 🗑 xoá → gạch + ↩️ Hoàn tác; hoàn tác khôi phục; số dư đi theo', async () => {
    const before = await cashBalance()
    const r = await ask('bún bò 55k')
    const botId = r.reply.message_id
    assert.equal(await cashBalance(), before - 55_000)
    let ans = await press({}, r.reply, /Xoá/)
    assert.equal(ans.text, 'Đã xoá')
    let m = await waitLatest(user.id, botId, (x) => /đã xoá/.test(x.text), 'gạch giao dịch')
    assert.ok(button(m, /Hoàn tác/))
    assert.notEqual((await recordTxs(user.id, r.id))[0].deleted_at, null)
    assert.equal(await cashBalance(), before)

    ans = await press({}, m, /Hoàn tác/)
    assert.equal(ans.text, 'Đã khôi phục')
    m = await waitLatest(user.id, botId, (x) => !!button(x, /Đổi danh mục/), 'khôi phục')
    assert.equal((await recordTxs(user.id, r.id))[0].deleted_at, null)
    assert.equal(await cashBalance(), before - 55_000)
  })

  step('người khác bấm nút: Telegram chưa kết nối bị chặn, chủ trọ khác nhận "không phải giao dịch của bạn"', async () => {
    s2 = await newSession({ prefix: 'tgflow2' })
    const created = await s2.api('POST', '/plugins/thu-chi/telegram/link')
    assert.equal(created.status, 201, JSON.stringify(created.body))
    const r2 = await ask(`/start ${await pendingCode(s2.owner)}`, { from: user2 })
    assert.match(r2.reply.text, /Đã kết nối với Thu Chi/)

    const r = await ask('trà sữa 30k', { chat: group })
    const [t] = await recordTxs(group.id, r.id)
    assert.equal(t.owner_id, s.owner)
    let ans = await press({ from: stranger, chat: group }, r.reply, /Xoá/)
    assert.match(ans.text, /Bạn chưa kết nối Thu Chi/)
    assert.equal(ans.show_alert, true)
    ans = await press({ from: user2, chat: group }, r.reply, /Xoá/)
    assert.equal(ans.text, 'Đây không phải giao dịch của bạn.')
    assert.equal(ans.show_alert, true)
    assert.equal((await recordTxs(group.id, r.id))[0].deleted_at, null)
    assert.deepEqual(await rows`select id from thuchi_transactions where owner_id = ${s2.owner} and merchant = 'Trà sữa'`, [])
  })

  step('trả lời tin bot: tên danh mục → đổi + học + 👌; câu lạ → bot hỏi lại; "xoá" → xoá', async () => {
    const r = await ask('kem que 15k')
    const botId = r.reply.message_id
    assert.equal((await recordTxs(user.id, r.id))[0].category_id, null)

    const pick = await say('giải trí', { replyTo: botId })
    await waitFor(async () => (await recordTxs(user.id, r.id))[0].cat_key === 'giai_tri', { message: 'đổi sang Giải trí' })
    await waitFor(() => tg.reaction(user.id, pick), { message: 'thả 👌' })
    assert.equal((await tg.reaction(user.id, pick)).reaction[0].emoji, '👌')
    await waitLatest(user.id, botId, (x) => /Giải trí/.test(x.text) && /Đã nhớ/.test(x.text), 'hiện danh mục mới')
    assert.ok(await row`select 1 as ok from thuchi_rules where owner_id = ${s.owner} and lower(pattern) = 'kem que' and origin = 'learned'`)

    const huh = await ask('abc', { replyTo: botId })
    assert.match(huh.reply.text, /Mình chưa hiểu/)

    await say('xoá', { replyTo: botId })
    await waitFor(async () => (await recordTxs(user.id, r.id))[0].deleted_at !== null, { message: 'xoá bằng trả lời' })
    await waitLatest(user.id, botId, (x) => /đã xoá/.test(x.text), 'gạch giao dịch')
  })

  step('sửa tin gốc: đổi số tiền, bớt khoản thì xoá khoản thừa, thêm khoản thì ghi thêm; update trùng không ghi đôi', async () => {
    const r = await ask('taxi 30k, nước 10k')
    const botId = r.reply.message_id
    await say('taxi 30k, nước 10k', { messageId: r.id })
    await drain()
    assert.equal(await value`select count(*)::int from thuchi_transactions where owner_id = ${s.owner} and merchant = 'Taxi'`, 1, 'nhận lại cùng message_id → bỏ qua')
    assert.equal((await repliesTo(user.id, r.id)).length, 1)

    await tg.edit({ from: user, messageId: r.id, text: 'taxi 35k' })
    await waitFor(async () => {
      const [taxi, nuoc] = await recordTxs(user.id, r.id)
      return Number(taxi.amount) === 35_000 && nuoc.deleted_at !== null
    }, { message: 'sửa còn 1 khoản' })
    await waitLatest(user.id, botId, (x) => new RegExp(`Đã ghi 1 khoản · chi ${vnd(35_000)}`).test(x.text) && /<s>Nước/.test(x.text), 'cập nhật tin trả lời')

    await tg.edit({ from: user, messageId: r.id, text: 'taxi 35k, nước 12k, bánh 8k' })
    const txs = await waitFor(async () => {
      const list = await recordTxs(user.id, r.id)
      return list.length === 3 && list[1].deleted_at === null ? list : null
    }, { message: 'sửa thành 3 khoản' })
    assert.deepEqual(txs.map((t) => [t.merchant, Number(t.amount)]), [['Taxi', 35_000], ['Nước', 12_000], ['Bánh', 8_000]])
    await waitLatest(user.id, botId, (x) => new RegExp(`Đã ghi 3 khoản · chi ${vnd(55_000)}`).test(x.text), 'cập nhật 3 khoản')

    const hi = await ask('chào cả nhà')
    assert.match(hi.reply.text, /Mình chưa thấy số tiền/)
    await tg.edit({ from: user, messageId: hi.id, text: 'trà chanh 12k' })
    const [t] = await recordTxs(user.id, hi.id)
    assert.equal(t.merchant, 'Trà chanh')
    await waitFor(async () => (await repliesTo(user.id, hi.id)).some((m) => /Trà chanh/.test(m.text)), { message: 'trả lời giao dịch mới từ tin sửa' })
  })

  step('/tong: chi/thu tháng này, hôm nay, top danh mục khớp DB; /tong@bot_khac trong nhóm bị bỏ qua', async () => {
    const totals = await row`select
        coalesce(sum(t.amount) filter (where t.direction = 'out'), 0)::bigint as spend,
        coalesce(sum(t.amount) filter (where t.direction = 'in'), 0)::bigint as income,
        coalesce(sum(t.amount) filter (where t.direction = 'out' and t.occurred_at >= date_trunc('day', now() at time zone 'Asia/Ho_Chi_Minh') at time zone 'Asia/Ho_Chi_Minh'), 0)::bigint as today,
        coalesce(sum(t.amount) filter (where t.direction = 'out' and c.key = 'mua_sam'), 0)::bigint as mua_sam
      from thuchi_transactions t
      left join thuchi_accounts a on a.id = t.account_id
      left join thuchi_categories c on c.id = t.category_id
      where t.owner_id = ${s.owner} and t.deleted_at is null and not t.excluded and t.kind = 'normal' and coalesce(a.include_in_stats, true)
        and t.occurred_at >= date_trunc('month', now() at time zone 'Asia/Ho_Chi_Minh') at time zone 'Asia/Ho_Chi_Minh'`
    const r = await ask('/tong')
    assert.match(r.reply.text, /📊 <b>Tháng \d{2}\/\d{4}<\/b>/)
    assert.ok(r.reply.text.includes(`Chi: <b>${vnd(Number(totals.spend))}</b> · Thu: <b>${vnd(Number(totals.income))}</b>`), r.reply.text)
    assert.ok(r.reply.text.includes(`Hôm nay chi: ${vnd(Number(totals.today))}`), r.reply.text)
    assert.ok(r.reply.text.includes(`Mua sắm ${vnd(Number(totals.mua_sam))}`), r.reply.text)

    const other = await say('/tong@bot_khac', { chat: group })
    const mine = await ask(`/tong@${FAKE_BOT_USERNAME}`, { chat: group })
    assert.match(mine.reply.text, /📊/)
    assert.deepEqual(await repliesTo(group.id, other), [])
  })

  step('nhóm: được phép thì ghi cả tin @mention; nhóm ngoài THUCHI_TELEGRAM_CHAT_ID bị bỏ qua hoàn toàn', async () => {
    const r = await ask(`@${FAKE_BOT_USERNAME} cơm tấm 35k`, { chat: group })
    const [t] = await recordTxs(group.id, r.id)
    assert.equal(t.merchant, 'Cơm tấm')
    assert.equal(Number(t.amount), 35_000)

    await say('cơm gà 40k', { chat: otherGroup })
    await say('/tong', { chat: otherGroup })
    await drain()
    assert.deepEqual(await tg.sent(otherGroup.id), [])
    assert.deepEqual(await rows`select id from thuchi_transactions where owner_id = ${s.owner} and merchant = 'Cơm gà'`, [])
  })

  step('Cài đặt → Telegram → "Ghi vào" tài khoản khác: tin mới ghi vào tài khoản đó; về mặc định thì lại Tiền mặt', async () => {
    const walletId = randomUUID()
    const res = await s.push([{ entity: 'accounts', op: 'upsert', id: walletId, changes: { name: 'Ví E2E', kind: 'ewallet', include_in_stats: true } }])
    assert.equal(res.status, 200, JSON.stringify(res.body))
    assert.ok(res.body.data.results[0].ok, JSON.stringify(res.body))
    await s.sync()
    await s.settings('Telegram')
    await s.choose({ label: 'Ghi vào' }, 'Ví E2E')
    await waitFor(async () => (await value`select default_account_id from thuchi_telegram_links where owner_id = ${s.owner} and telegram_user_id = ${user.id}`) === walletId, { message: 'lưu tài khoản ghi vào' })

    const cash = await cashBalance()
    const r = await ask('xăng 50k')
    assert.match(r.reply.text, /Ví E2E/)
    const [t] = await recordTxs(user.id, r.id)
    assert.equal(t.account_id, walletId)
    assert.equal(t.cat_key, 'xang_xe')
    assert.equal(Number((await account.get(walletId)).balance), -50_000)
    assert.equal(await cashBalance(), cash)

    await s.choose({ label: 'Ghi vào' }, 'Tiền mặt (mặc định)')
    await waitFor(async () => (await row`select default_account_id from thuchi_telegram_links where owner_id = ${s.owner} and telegram_user_id = ${user.id}`)?.default_account_id === null, { message: 'về mặc định' })
    const back = await ask('xăng 60k')
    assert.equal((await recordTxs(user.id, back.id))[0].account_id, (await account.byName(s.owner, 'Tiền mặt')).id)
  })

  step('Ngắt kết nối trong app → bot quên tài khoản, nhắn tiếp được nhắc kết nối, không ghi', async () => {
    await s.settings('Telegram')
    await s.expectText('@lan_e2e')
    await s.click('Ngắt')
    await waitFor(async () => (await value`select count(*)::int from thuchi_telegram_links where telegram_user_id = ${user.id}`) === 0, { message: 'xoá liên kết' })
    await s.expectNoText('@lan_e2e')
    await s.expectText('Kết nối Telegram')

    const r = await ask('trà đá 5k')
    assert.match(r.reply.text, /chưa biết bạn dùng tài khoản Thu Chi nào/)
    assert.deepEqual(await rows`select id from thuchi_transactions where owner_id = ${s.owner} and merchant = 'Trà đá'`, [])
    const link2 = await row`select telegram_user_id from thuchi_telegram_links where owner_id = ${s2.owner} and telegram_user_id is not null`
    assert.equal(Number(link2.telegram_user_id), user2.id, 'không ảnh hưởng liên kết của chủ trọ khác')
  })
})
