import assert from 'node:assert/strict'
import { randomBytes, randomUUID } from 'node:crypto'
import { after, before, describe } from 'node:test'
import { closeAll, newSession, step } from '../lib/app.mjs'
import { account, accountHash, category, row, rows, tx, value, waitFor } from '../lib/db.mjs'
import { gmailForwardingEmail, ingestUrl, otherEmail, postEmail, postNotify, postSePay, sepayPayload, timoStatementEmail, shownIngestUrl, vcbReceiptEmail, waitRaw } from '../lib/ingest.mjs'
import { vnDay, vnInstant } from '../lib/time.mjs'

describe('Token thiết bị forward, thông báo Android, email VCB, sao kê Timo .xls', () => {
  let s
  let notifyToken
  let emailToken
  const today = vnDay()
  const VCB = '1012345678'
  const TIMO = '8007000123456'

  before(async () => {
    s = await newSession({ prefix: 'mail' })
  })
  after(closeAll)

  async function createToken(button, name) {
    await s.settings('Kết nối ngân hàng')
    if (name) await s.fill('input[placeholder="Tên, vd: Android Minh"]', name)
    await s.click(button)
    await s.waitSheet('Token mới')
    const [url, auth] = await s.page.evaluate(() => [...window.__e2e.layers().at(-1).querySelectorAll('input[readonly]')].map((i) => i.value))
    await s.closeSheet()
    assert.match(auth, /^Bearer [0-9a-f-]{36}\.[\w-]{20,}$/)
    return { url, token: auth.slice('Bearer '.length) }
  }

  async function unparsedPanel() {
    await s.settings('Thông báo không đọc được')
  }

  step('tạo token thông báo Android: secret chỉ hiện 1 lần, server lưu bản mã hoá', async () => {
    const { url, token } = await createToken('Token thông báo Android', 'Android E2E')
    notifyToken = token
    assert.equal(url, shownIngestUrl(s.owner, 'notify'))
    const [id, secret] = token.split(/\.(.*)/s)
    const t = await row`select * from thuchi_ingest_tokens where id = ${id}`
    assert.equal(t.owner_id, s.owner)
    assert.equal(t.kind, 'notify')
    assert.equal(t.name, 'Android E2E')
    assert.ok(!t.secret_enc.includes(secret))
    await s.expectText('Android E2E')
    await s.expectText('nhận lần cuối chưa')
  })

  step('thông báo VCB (chưa có parser) được nhận, lưu raw và hiện ở "Thông báo không đọc được"', async () => {
    const payload = { package: 'com.VCB', title: 'Thông báo VCB', text: `SD TK ${VCB} -120,000VND luc ${today.d}/${today.m} 10:00. SD 5,000,000VND. Ref ${randomBytes(3).toString('hex')}`, posted_at: String(Date.now()) }
    const res = await postNotify(s, payload, { token: notifyToken })
    assert.equal(res.status, 200, res.body)
    assert.deepEqual(JSON.parse(res.body).data, { accepted: true })
    const raw = await waitRaw(s.owner, payload.text)
    assert.equal(raw.source, 'notify')
    assert.equal(raw.source_key, 'com.VCB')
    assert.equal(raw.status, 'unparsed')
    assert.equal(raw.error, 'chưa có parser cho thông báo của Vietcombank')
    await s.sync()
    await s.tab('Tổng quan')
    await s.click({ text: '1 thông báo ngân hàng chưa đọc được', exact: false })
    await s.find({ text: 'Thông báo không đọc được', selector: 'main h2', within: 'page' })
    await s.expectText('com.VCB')
    await s.expectText('chưa có parser cho thông báo của Vietcombank')
    await s.click({ text: 'com.VCB', exact: false, selector: 'main section button', within: 'page' })
    await s.expectText('"package":"com.VCB"')
    await s.settings('Kết nối ngân hàng')
    await s.expectText('nhận lần cuối vừa xong')
  })

  step('HMAC X-ThuChi-* và payload kiểu SmsForwarder; sai chữ ký/hết hạn/token lạ bị 401', async () => {
    const payload = { from: 'com.mservice.momotransfer', title: 'MoMo', content: `Nhan 50.000d tu NGUYEN A ${randomBytes(3).toString('hex')}`, receive_time: '2026-11-04 10:00:00' }
    assert.equal((await postNotify(s, payload, { token: notifyToken, mode: 'hmac' })).status, 200)
    const raw = await waitRaw(s.owner, payload.content)
    assert.equal(raw.source_key, 'com.mservice.momotransfer')
    assert.equal(raw.error, 'chưa có parser cho thông báo của MoMo')
    const [id] = notifyToken.split('.')
    assert.equal((await postNotify(s, payload, { token: `${id}.sai-secret` })).status, 401)
    assert.equal((await postNotify(s, payload, { token: `${id}.sai-secret`, mode: 'hmac' })).status, 401)
    assert.equal((await postNotify(s, payload, { token: notifyToken, mode: 'hmac', timestamp: String(Math.floor(Date.now() / 1000) - 600) })).status, 401)
    assert.equal((await postNotify(s, payload, { token: `${randomUUID()}.x` })).status, 401)
    assert.equal(await value`select count(*)::int from thuchi_raw_events where owner_id = ${s.owner}`, 2)
  })

  step('"Đọc lại" đưa raw event về xử lý lại, "Bỏ qua" ẩn khỏi danh sách', async () => {
    const vcb = await row`select * from thuchi_raw_events where owner_id = ${s.owner} and source_key = 'com.VCB'`
    await s.sync()
    await unparsedPanel()
    await waitFor(async () => (await s.page.evaluate(() => document.querySelectorAll('main section .divide-y > div').length)) === 2, { message: '2 thông báo trong danh sách' })
    await s.click({ text: 'Đọc lại', exact: false, nth: 1, within: 'page', selector: 'main button' })
    await waitFor(async () => Number((await row`select change_seq from thuchi_raw_events where id = ${vcb.id}`).change_seq) > Number(vcb.change_seq) + 1, { message: 'raw event được xử lý lại' })
    const again = await waitRaw(s.owner, 'com.VCB')
    assert.equal(again.status, 'unparsed')
    assert.equal(again.attempts, 1)
    await s.click({ text: 'Bỏ qua', nth: 1, within: 'page', selector: 'main button' })
    await waitFor(async () => (await row`select status from thuchi_raw_events where id = ${vcb.id}`).status === 'dismissed', { message: 'dismissed' })
    await s.expectNoText('com.VCB', { within: 'main' })
    await s.click({ text: 'Bỏ qua', within: 'page', selector: 'main button' })
    await s.expectText('Không có thông báo nào lỗi')
    await s.tab('Tổng quan')
    await s.expectNoText('thông báo ngân hàng chưa đọc được')
  })

  step('thu hồi token: thiết bị đó không gửi được nữa', async () => {
    await s.settings('Kết nối ngân hàng')
    await s.click('Thu hồi')
    const [id] = notifyToken.split('.')
    await waitFor(() => value`select revoked_at is not null from thuchi_ingest_tokens where id = ${id}`, { message: 'revoked_at' })
    await s.expectText('Chưa có thiết bị')
    assert.equal((await postNotify(s, { package: 'com.VCB', text: 'sau thu hoi' }, { token: notifyToken })).status, 401)
  })

  step('email biên lai Vietcombank: đọc ô HTML, cộng phí người chuyển trả, tạo tài khoản VCB kết nối email', async () => {
    emailToken = (await createToken('Token email')).token
    const at = vnInstant(today, 1, 0, 0)
    const raw = vcbReceiptEmail({ messageId: 'vcb-5123456789@e2e.local', at, orderNo: '5123456789', debit: VCB, amount: 250_000, fee: 1_100, payee: 'NGUYEN VAN B', payeeAccount: '0987654321', payeeBank: 'Techcombank', content: 'Tra tien an trua' })
    const res = await postEmail(s, raw, { token: emailToken })
    assert.equal(res.status, 200, res.body)
    const ev = await waitRaw(s.owner, 'vcb-5123456789@e2e.local')
    assert.equal(ev.status, 'parsed', ev.error)
    assert.equal(ev.source_key, 'Vietcombank <VCBDigibank@info.vietcombank.com.vn>')
    const t = await tx.get(ev.transaction_id)
    assert.equal(t.direction, 'out')
    assert.equal(t.amount, 251_100)
    assert.equal(new Date(t.occurred_at).getTime(), at.getTime())
    assert.equal(t.counterparty_name, 'NGUYEN VAN B')
    assert.equal(t.counterparty_account, '0987654321')
    assert.equal(t.raw_content, 'Tra tien an trua - Techcombank')
    assert.deepEqual(t.sources, ['email'])
    assert.equal(await value`select source_ref from thuchi_tx_sources where transaction_id = ${t.id}`, `vcb:${VCB}:5123456789`)
    const acc = await account.get(t.account_id)
    assert.equal(acc.name, 'Vietcombank ·5678')
    assert.equal(acc.connector, 'email')
    assert.equal(acc.account_no_hash, accountHash(s.owner, VCB))

    assert.equal((await postEmail(s, raw, { token: emailToken })).status, 200)
    await waitFor(async () => (await rows`select status from thuchi_raw_events where owner_id = ${s.owner} and payload like '%vcb-5123456789@e2e.local%' order by received_at`).map((r) => r.status).join() === 'parsed,duplicate', { message: 'email gửi lại = duplicate' })

    await s.sync()
    await s.tab('Tổng quan')
    await s.expectSection('Tài khoản', ['Vietcombank ·5678', 'Email'])
    await s.tab('Giao dịch')
    await s.expectText('NGUYEN VAN B')
    await s.expectText('−251.100 ₫')
  })

  step('email Timo kèm sao kê .xls: mỗi dòng thành 1 giao dịch, số dư lấy theo dòng cuối', async () => {
    const rowsIn = [
      { at: vnInstant(today, 1, 15), ref: 'TF2611040001', debit: 250_000, balance: 4_750_000, desc: 'Thanh toan HIGHLANDS COFFEE' },
      { at: vnInstant(today, 1, 30), ref: 'TF2611040002', credit: 5_000_000, balance: 9_750_000, desc: 'Nhan tien tu cong ty' },
    ]
    const res = await postEmail(s, timoStatementEmail({ account: TIMO, rows: rowsIn }), { token: emailToken })
    assert.equal(res.status, 200, res.body)
    const ev = await waitFor(async () => {
      const r = await row`select * from thuchi_raw_events where owner_id = ${s.owner} and source_key like '%timo.vn%' order by received_at desc limit 1`
      return r && r.status !== 'pending' && r
    }, { message: 'raw event sao kê' })
    assert.equal(ev.status, 'parsed', ev.error)
    const txs = await rows`select t.* from thuchi_transactions t join thuchi_tx_sources s on s.transaction_id = t.id where t.owner_id = ${s.owner} and s.source_ref like ${`stmt:${TIMO}:%`} order by t.occurred_at`
    assert.equal(txs.length, 2)
    assert.equal(ev.transaction_id, txs[0].id)
    assert.deepEqual(txs.map((t) => [t.direction, t.amount, t.balance_after]), [['out', 250_000, 4_750_000], ['in', 5_000_000, 9_750_000]])
    assert.equal(txs[0].merchant, 'Highlands Coffee')
    assert.equal(txs[0].category_id, (await category.byKey(s.owner, 'an_uong')).id)
    assert.equal(txs[1].category_id, (await category.byKey(s.owner, 'duoc_chuyen', 'income')).id)
    const timo = await account.get(txs[0].account_id)
    assert.equal(timo.name, 'Timo ·3456')
    assert.equal(timo.bank_code, 'TIMO')
    assert.equal(timo.balance, 9_750_000)
    await s.sync()
    await s.tab('Tổng quan')
    await s.expectSection('Tài khoản', ['Timo ·3456', '9.750.000 ₫'])
    await s.tab('Giao dịch')
    await s.expectText('Highlands Coffee')
    await s.expectText('+5.000.000 ₫')
  })

  step('cùng giao dịch về từ email rồi SePay → gộp làm 1, nguồn ưu tiên SePay, đánh dấu đã đối chiếu', async () => {
    const secret = `e2e-sepay-${randomBytes(6).toString('hex')}`
    const put = await s.api('PUT', '/plugins/thu-chi/ingest/sepay', { auth_method: 'hmac', secret })
    assert.equal(put.status, 200, JSON.stringify(put.body))
    s.sepaySecret = secret
    const at = vnInstant(today, 2, 0, 0)
    await postEmail(s, vcbReceiptEmail({ messageId: 'vcb-5123456790@e2e.local', at, orderNo: '5123456790', debit: VCB, amount: 99_000, payee: 'QUAN CAFE A', payeeAccount: '0123123123', payeeBank: 'ACB', content: 'Tra tien cafe' }), { token: emailToken })
    const ev = await waitRaw(s.owner, 'vcb-5123456790@e2e.local')
    const emailTx = await tx.get(ev.transaction_id)
    const payload = sepayPayload({ account: VCB, gateway: 'Vietcombank', amount: 99_000, content: 'MBVCB.5123456790 Tra tien cafe', at: new Date(at.getTime() + 2 * 60_000), accumulated: 5_000_000 })
    assert.equal((await postSePay(s, payload)).status, 200)
    const sev = await waitRaw(s.owner, payload.referenceCode)
    assert.equal(sev.status, 'parsed')
    assert.equal(sev.transaction_id, emailTx.id, 'SePay gộp vào giao dịch email')
    const merged = await tx.get(emailTx.id)
    assert.deepEqual(merged.sources, ['email', 'sepay'])
    assert.equal(merged.reconciled, true)
    assert.equal(merged.raw_content, 'MBVCB.5123456790 Tra tien cafe', 'nguồn SePay ưu tiên hơn ghi đè nội dung')
    assert.equal(merged.balance_after, 5_000_000)
    assert.equal(await value`select count(*)::int from thuchi_transactions where owner_id = ${s.owner} and amount = 99000`, 1)
    assert.equal(await value`select count(*)::int from thuchi_tx_sources where transaction_id = ${emailTx.id}`, 2)
    await s.sync()
    await s.tab('Giao dịch')
    await s.openTx('QUAN CAFE A')
    await s.expectText('Nguồn: email, sepay · đã đối chiếu')
  })

  step('email xác nhận chuyển tiếp Gmail và email lạ được lưu để người dùng xử lý', async () => {
    const link = 'https://mail-settings.google.com/mail/vf-%5BANGjdJ_e2e%5D-abc'
    await postEmail(s, gmailForwardingEmail(link), { token: emailToken })
    await postEmail(s, otherEmail(), { token: emailToken })
    const gm = await waitFor(async () => {
      const r = await row`select * from thuchi_raw_events where owner_id = ${s.owner} and source_key like '%forwarding-noreply@google.com%'`
      return r && r.status !== 'pending' && r
    })
    assert.equal(gm.status, 'unparsed')
    assert.equal(gm.error, `Gmail cần xác nhận địa chỉ chuyển tiếp: ${link}`)
    const shop = await waitFor(async () => {
      const r = await row`select * from thuchi_raw_events where owner_id = ${s.owner} and source_key like '%news@shop.example%'`
      return r && r.status !== 'pending' && r
    })
    assert.equal(shop.error, 'chưa có parser cho email từ news@shop.example (Khuyến mãi tháng này)')
    assert.equal((await postEmail(s, otherEmail(), { token: emailToken, owner: randomUUID() })).status, 404, 'owner khác chưa bật tiện ích: host chặn trước khi tới plugin')
    await s.sync()
    await unparsedPanel()
    await s.expectText(link)
    await s.expectText('chưa có parser cho email từ news@shop.example')
  })
})
