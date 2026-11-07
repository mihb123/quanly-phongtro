import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { after, before, describe } from 'node:test'
import { closeAll, newSession, step, vnd } from '../lib/app.mjs'
import { account, accountHash, category, monthTotals, row, rows, tx, value, waitFor } from '../lib/db.mjs'
import { postSePay, shownIngestUrl, sepayPayload, waitRaw } from '../lib/ingest.mjs'
import { monthLabel, vnDay } from '../lib/time.mjs'

describe('SePay webhook → raw event → giao dịch → đồng bộ về máy', () => {
  let s
  let grab
  let tpb
  const TPB = '0001112223334'
  const MB = '0909123456'
  const today = vnDay()
  const heroTitle = `Chi ${monthLabel(today).toLowerCase()}`
  const minutes = (n) => new Date(Date.now() + n * 60_000)

  before(async () => {
    s = await newSession({ prefix: 'sepay' })
    s.sepaySecret = `e2e-sepay-${randomBytes(6).toString('hex')}`
  })
  after(closeAll)

  async function ingest(fields) {
    const payload = sepayPayload(fields)
    const res = await postSePay(s, payload)
    assert.equal(res.status, 200, res.body)
    assert.equal(res.body, '{"success":true}')
    const raw = await waitRaw(s.owner, payload.referenceCode)
    return { payload, raw, t: raw.transaction_id ? await tx.get(raw.transaction_id) : null }
  }

  async function expectHero() {
    const totals = await monthTotals(s.owner, today)
    await s.tab('Tổng quan')
    await s.expectSection(heroTitle, vnd(totals.expense))
    return totals
  }

  step('cấu hình SePay (HMAC) trong Cài đặt → Kết nối ngân hàng, secret lưu mã hoá', async () => {
    await s.settings('Kết nối ngân hàng')
    await s.expectText('Chưa cấu hình')
    const url = await s.page.evaluate(() => [...document.querySelectorAll('input[readonly]')].map((i) => i.value).find((v) => v.includes('/ingest/sepay')))
    assert.equal(url, shownIngestUrl(s.owner, 'sepay'))
    await s.fill('input[placeholder="Secret Key từ SePay"]', 'short')
    assert.equal(await s.page.evaluate(() => window.__e2e.find({ text: 'Lưu SePay' })?.disabled), true, 'secret < 8 ký tự thì không lưu')
    await s.fill('input[placeholder="Secret Key từ SePay"]', s.sepaySecret)
    await s.click('Lưu SePay')
    await s.toast('Đã lưu cấu hình SePay')
    await s.expectText('Đã cấu hình (hmac) · nhận lần cuối chưa')
    const tok = await row`select * from thuchi_ingest_tokens where owner_id = ${s.owner} and kind = 'sepay' and revoked_at is null`
    assert.deepEqual(tok.config, { auth_method: 'hmac' })
    assert.ok(!tok.secret_enc.includes(s.sepaySecret), 'secret không lưu dạng rõ')
  })

  step('webhook sai chữ ký hoặc thiếu header bị từ chối 401, không ghi raw event', async () => {
    const payload = sepayPayload({ account: TPB, amount: 1000, content: 'thu sai chu ky' })
    assert.equal((await postSePay(s, payload, { secret: 'sai-secret-123' })).status, 401)
    assert.equal((await postSePay(s, payload, { signature: '' })).status, 401)
    assert.equal((await postSePay(s, payload, { timestamp: String(Math.floor(Date.now() / 1000) - 3600) })).status, 401, 'timestamp quá 5 phút')
    assert.equal(await value`select count(*)::int from thuchi_raw_events where owner_id = ${s.owner}`, 0)
  })

  step('tiền ra Grab: tự tạo tài khoản TPBank, cập nhật số dư, nhận diện merchant + danh mục', async () => {
    const r = await ingest({ account: TPB, type: 'out', amount: 65_000, content: 'GRAB*A-8XK2 thanh toan dat xe', accumulated: 1_935_000 })
    grab = r.t
    assert.equal(r.raw.status, 'parsed')
    assert.equal(r.raw.source, 'sepay')
    assert.equal(r.raw.source_key, 'TPBank 3334')
    assert.equal(grab.amount, 65_000)
    assert.equal(grab.direction, 'out')
    assert.equal(grab.kind, 'normal')
    assert.equal(grab.merchant, 'Grab')
    assert.equal(grab.category_id, (await category.byKey(s.owner, 'di_lai')).id)
    assert.deepEqual(grab.sources, ['sepay'])
    assert.equal(grab.balance_after, 1_935_000)
    const src = await row`select * from thuchi_tx_sources where transaction_id = ${grab.id}`
    assert.equal(src.source_ref, `${TPB}:${r.payload.referenceCode}`)
    assert.equal(src.raw_event_id, r.raw.id)
    tpb = await account.get(grab.account_id)
    assert.equal(tpb.name, 'TPBank ·3334')
    assert.equal(tpb.bank_code, 'TPB')
    assert.equal(tpb.connector, 'sepay')
    assert.equal(tpb.balance, 1_935_000)
    assert.equal(tpb.account_no_hash, accountHash(s.owner, TPB))

    await s.sync()
    await s.tab('Tổng quan')
    await s.expectSection('Tài khoản', ['TPBank ·3334', 'SePay', '1.935.000 ₫'])
    await s.expectSection(heroTitle, '65.000 ₫')
    await s.tab('Giao dịch')
    await s.expectText('Grab')
    await s.expectText('· Đi lại')
    await s.openTx('Grab')
    await s.expectText('GRAB*A-8XK2 thanh toan dat xe')
    await s.expectText('Nguồn: sepay')
    assert.equal(await s.exists({ text: 'Sửa' }), false, 'giao dịch ngân hàng không sửa số tiền được')
    assert.equal(await s.exists({ text: 'Xoá' }), false)
    await s.closeSheet()
    await s.settings('Kết nối ngân hàng')
    await s.expectText('Đã cấu hình (hmac) · nhận lần cuối vừa xong')
  })

  step('SePay gửi lại cùng giao dịch → raw event "duplicate", không tạo giao dịch mới', async () => {
    const payload = JSON.parse(JSON.stringify(sepayPayload({ account: TPB, amount: 65_000, content: 'GRAB*A-8XK2 thanh toan dat xe', accumulated: 1_935_000 })))
    payload.referenceCode = (await row`select source_ref from thuchi_tx_sources where transaction_id = ${grab.id}`).source_ref.split(':')[1]
    assert.equal((await postSePay(s, payload)).status, 200)
    const raw = await waitFor(async () => {
      const r = await rows`select * from thuchi_raw_events where owner_id = ${s.owner} and payload like ${`%${payload.referenceCode}%`} order by received_at`
      return r.length === 2 && r[1].status !== 'pending' && r[1]
    }, { message: 'raw event thứ 2 xử lý xong' })
    assert.equal(raw.status, 'duplicate')
    assert.equal(raw.transaction_id, grab.id)
    assert.equal(await value`select count(*)::int from thuchi_transactions where owner_id = ${s.owner}`, 1)
  })

  step('tiền vào "HOAN TIEN GRAB" là hoàn tiền: trừ vào chi của danh mục gốc', async () => {
    const { t } = await ingest({ account: TPB, type: 'in', amount: 20_000, content: 'HOAN TIEN GRAB A-8XK2', accumulated: 1_955_000 })
    assert.equal(t.kind, 'refund')
    assert.equal(t.merchant, 'Grab')
    assert.equal(t.refund_of_category_id, grab.category_id)
    await s.sync()
    const totals = await expectHero()
    assert.equal(totals.expense, 45_000)
    assert.equal(totals.income, 0)
    await s.tab('Giao dịch')
    await s.expectText('Hoàn tiền')
    await s.expectText('Chi 45.000 ₫ · Thu 0 ₫')
  })

  step('"THANH TOAN THE TIN DUNG" là trả thẻ: không tính chi', async () => {
    const { t } = await ingest({ account: TPB, type: 'out', amount: 1_000_000, content: 'THANH TOAN THE TIN DUNG 411111', accumulated: 955_000 })
    assert.equal(t.kind, 'card_payment')
    await s.sync()
    await s.tab('Giao dịch')
    await s.expectText('Trả thẻ')
    await s.expectText('Chi 45.000 ₫ · Thu 0 ₫')
    await s.tab('Tổng quan')
    await s.expectSection('Tài khoản', ['TPBank ·3334', '955.000 ₫'])
  })

  step('client không sửa được trường ngân hàng (số tiền, chiều, thời gian, nội dung), chỉ trường người dùng', async () => {
    const res = await s.push([{ entity: 'transactions', op: 'patch', id: grab.id, changes: { amount: 1, direction: 'in', occurred_at: '2020-01-01T00:00:00Z', raw_content: 'sua bay', note: 'ghi chú hợp lệ' } }])
    assert.equal(res.status, 200)
    assert.equal(res.body.data.results[0].ok, true)
    const t = await tx.get(grab.id)
    assert.equal(t.amount, 65_000)
    assert.equal(t.direction, 'out')
    assert.equal(new Date(t.occurred_at).getTime(), new Date(grab.occurred_at).getTime())
    assert.equal(t.raw_content, grab.raw_content)
    assert.equal(t.note, 'ghi chú hợp lệ')
  })

  step('merchant lạ vào hộp "Cần xem"; phân loại 1 giao dịch + tự học áp cho giao dịch tương tự', async () => {
    const a = await ingest({ account: TPB, amount: 150_000, content: 'TT TIEM HOA MAI 778899' })
    const b = await ingest({ account: TPB, amount: 90_000, content: 'TT TIEM HOA MAI 112233' })
    for (const { t } of [a, b]) {
      assert.equal(t.merchant, 'Tiem Hoa Mai')
      assert.equal(t.category_id, null)
      assert.equal(t.needs_review, true)
    }
    await s.sync()
    await s.tab('Tổng quan')
    await s.expectSection('Cần xem (2)', ['Tiem Hoa Mai', '−150.000 ₫', '−90.000 ₫'])
    await s.expectText('Giao dịch 2', { within: 'nav' })
    await s.tab('Giao dịch')
    await s.click({ text: 'Cần xem', exact: false, selector: 'main button', within: 'page' })
    await s.expectText('2 giao dịch')
    await s.openTx('150.000')
    await s.click({ text: 'Mua sắm', selector: 'button' })
    await s.toast('Đã phân loại và ghi nhớ cho giao dịch tương tự')
    const muaSam = await category.byKey(s.owner, 'mua_sam')
    await waitFor(async () => (await rows`select category_id, needs_review from thuchi_transactions where owner_id = ${s.owner} and merchant = 'Tiem Hoa Mai'`).every((t) => t.category_id === muaSam.id && !t.needs_review), { message: 'cả 2 giao dịch Tiem Hoa Mai thành Mua sắm' })
    const rule = await row`select * from thuchi_rules where owner_id = ${s.owner} and origin = 'learned'`
    assert.equal(rule.pattern, 'Tiem Hoa Mai')
    assert.equal(rule.match, 'merchant')
    await s.sync()
    await s.expectText('Không còn giao dịch nào cần xem 🎉')
    await s.expectNoText('Giao dịch 2', { within: 'nav' })

    const c = await ingest({ account: TPB, amount: 60_000, content: 'TT TIEM HOA MAI 445566' })
    assert.equal(c.t.category_id, muaSam.id, 'giao dịch mới tự phân loại theo luật đã học')
    assert.equal(c.t.needs_review, false)
    assert.equal(await value`select hits::int from thuchi_rules where id = ${rule.id}`, 1)
  })

  step('chuyển khoản nội bộ có số TK của mình trong nội dung → tự ghép "auto", tách phí 3.300 ₫', async () => {
    await s.settings('Tài khoản')
    await s.click('Thêm tài khoản')
    await s.waitSheet('Thêm tài khoản')
    await s.fill('#acc-name', 'MB của tôi')
    await s.fill('#acc-bank', 'MB')
    await s.fill('#acc-no', MB)
    await s.click('Lưu')
    await s.toast('Đã lưu tài khoản')
    const mb = await waitFor(() => account.byName(s.owner, 'MB của tôi'))
    const out = await ingest({ account: TPB, amount: 1_003_300, content: `CK den ${MB} tiet kiem`, at: minutes(1) })
    assert.equal(out.t.kind, 'transfer')
    const inn = await ingest({ account: MB, gateway: 'MBBank', type: 'in', amount: 1_000_000, content: 'NHAN TIEN TU TPBANK', at: minutes(2), accumulated: 1_000_000 })
    assert.equal(inn.t.account_id, mb.id, 'nhận ra tài khoản đã khai báo theo hash số TK')
    const [o, i] = await Promise.all([tx.get(out.t.id), tx.get(inn.t.id)])
    assert.equal(o.kind, 'transfer')
    assert.equal(i.kind, 'transfer')
    assert.equal(o.pair_id, i.id)
    assert.equal(i.pair_id, o.id)
    assert.equal(o.pair_status, 'auto')
    assert.equal(i.category_id, null)
    const fee = await row`select * from thuchi_transactions where owner_id = ${s.owner} and sources = '["derived"]'::jsonb`
    assert.equal(fee.amount, 3_300)
    assert.equal(fee.account_id, tpb.id)
    assert.equal(fee.category_id, (await category.byKey(s.owner, 'phi_ngan_hang')).id)
    const mbNow = await account.get(mb.id)
    assert.equal(mbNow.connector, 'sepay', 'tài khoản tay chuyển sang kết nối SePay')
    assert.equal(mbNow.balance, 1_000_000)
    await s.sync()
    await expectHero()
    await s.tab('Giao dịch')
    await s.expectText('Phí chuyển khoản')
    await s.expectText('Nội bộ')
  })

  step('chỉ khớp số tiền → "Nghi nội bộ" trong Cần xem; xác nhận thì thành chuyển nội bộ', async () => {
    const out = await ingest({ account: TPB, amount: 2_000_000, content: 'CHUYEN TIEN', at: minutes(3) })
    const inn = await ingest({ account: MB, gateway: 'MBBank', type: 'in', amount: 2_000_000, content: 'NHAN TIEN', at: minutes(4), accumulated: 3_000_000 })
    await waitFor(async () => (await tx.get(out.t.id)).pair_status === 'suggested', { message: 'pair suggested' })
    const i = await tx.get(inn.t.id)
    assert.equal(i.pair_status, 'suggested')
    assert.equal(i.pair_id, out.t.id)
    assert.equal(i.needs_review, true)
    await s.sync()
    await s.tab('Tổng quan')
    await s.expectSection('Cần xem (2)', 'Nghi nội bộ')
    await s.tab('Giao dịch')
    await s.click({ text: 'Cần xem', exact: false, selector: 'main button', within: 'page' })
    await s.openTx('+2.000.000')
    await s.expectText('Có thể là chuyển nội bộ với 2.000.000 ₫ ở TPBank ·3334')
    await s.click('Đúng, chuyển nội bộ')
    await waitFor(async () => (await rows`select kind, pair_status from thuchi_transactions where id in (${out.t.id}, ${inn.t.id})`).every((t) => t.kind === 'transfer' && t.pair_status === 'confirmed'), { message: 'cả cặp confirmed' })
    await s.sync()
    await s.expectText('Không còn giao dịch nào cần xem 🎉')
    await expectHero()
  })

  step('"Không phải" bỏ ghép; đánh dấu/bỏ đánh dấu chuyển nội bộ bằng tay', async () => {
    const out = await ingest({ account: TPB, amount: 3_000_000, content: 'CHUYEN TIEN', at: minutes(5) })
    const inn = await ingest({ account: MB, gateway: 'MBBank', type: 'in', amount: 3_000_000, content: 'NHAN TIEN', at: minutes(6), accumulated: 6_000_000 })
    await waitFor(async () => (await tx.get(inn.t.id)).pair_status === 'suggested', { message: 'pair suggested' })
    await s.sync()
    await s.tab('Giao dịch')
    await s.click({ text: 'Cần xem', exact: false, selector: 'main button', within: 'page' })
    await s.openTx('+3.000.000')
    await s.click('Không phải')
    await waitFor(async () => (await rows`select kind, pair_status from thuchi_transactions where id in (${out.t.id}, ${inn.t.id})`).every((t) => t.kind === 'normal' && t.pair_status === 'rejected'), { message: 'cả cặp rejected' })
    await s.sync()
    const before = await expectHero()

    await s.tab('Giao dịch')
    await s.click({ text: 'Trong tháng', selector: 'main button', within: 'page' })
    await s.openTx('−3.000.000')
    await s.click('Đánh dấu chuyển nội bộ')
    await waitFor(async () => (await tx.get(out.t.id)).kind === 'transfer', { message: 'mark_transfer' })
    const marked = await expectHero()
    assert.equal(marked.expense, before.expense - 3_000_000)
    await s.tab('Giao dịch')
    await s.openTx('−3.000.000')
    await s.click('Không phải chuyển nội bộ')
    await waitFor(async () => (await tx.get(out.t.id)).kind === 'normal', { message: 'unmark_transfer' })
    assert.equal((await expectHero()).expense, before.expense)
  })
})
