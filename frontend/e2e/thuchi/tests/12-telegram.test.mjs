import assert from 'node:assert/strict'
import { after, before, describe } from 'node:test'
import { closeAll, newSession, step } from '../lib/app.mjs'
import { TELEGRAM_FAKE_URL } from '../lib/config.mjs'
import { row, rows, value, waitFor } from '../lib/db.mjs'
import { FAKE_BOT_USERNAME, fakeTelegramClient } from '../lib/telegram-fake.mjs'

describe('Bot Telegram của tiện ích (Bot API giả): kết nối, ghi giao dịch, sửa, tôn trọng bật/tắt', () => {
  let s
  const tg = fakeTelegramClient(TELEGRAM_FAKE_URL)
  const user = { id: 700000000 + Math.floor(Math.random() * 1e6), name: 'Minh', username: 'minh_e2e' }

  before(async () => {
    s = await newSession({ prefix: 'tele' })
  })
  after(closeAll)

  const lastReply = async (after = 0) => {
    const msgs = await tg.sent(user.id)
    return msgs.filter((m) => m.message_id > after).at(-1)
  }
  const waitReply = (match, after = 0) =>
    waitFor(
      async () => {
        const msgs = (await tg.sent(user.id)).filter((m) => m.message_id > after || m.edited)
        return msgs.find((m) => match.test(m.text))
      },
      { message: `bot trả lời khớp ${match}` },
    )
  const txByMerchant = (merchant) => row`select * from thuchi_transactions where owner_id = ${s.owner} and merchant = ${merchant} and deleted_at is null`

  step('Cài đặt → Telegram hiện bot của server, tạo mã kết nối 1 lần', async () => {
    await s.settings('Telegram')
    await s.expectText(`@${FAKE_BOT_USERNAME}`)
    await s.click('Kết nối Telegram')
    await s.expectText('Mở Telegram, bấm Start')
    const link = await row`select code, code_expires_at from thuchi_telegram_links where owner_id = ${s.owner}`
    assert.match(link.code, /^[A-Z0-9]{8}$/)
    await waitFor(async () => (await s.page.evaluate(() => [...document.querySelectorAll('input[readonly]')].map((i) => i.value))).includes(`/link ${link.code}`), { message: 'ô Lệnh kết nối' })
  })

  step('/start <mã> trong chat riêng → kết nối tài khoản Telegram với chủ trọ, app tự cập nhật', async () => {
    const code = await value`select code from thuchi_telegram_links where owner_id = ${s.owner}`
    await tg.message({ from: user, text: `/start ${code}` })
    await waitReply(/Đã kết nối với Thu Chi/)
    const link = await row`select telegram_user_id, telegram_username, code from thuchi_telegram_links where owner_id = ${s.owner}`
    assert.equal(Number(link.telegram_user_id), user.id)
    assert.equal(link.telegram_username, user.username)
    assert.equal(link.code, null, 'mã đã dùng bị xoá')
    await s.expectText('@minh_e2e', { timeout: 20000 })
  })

  step('nhắn "cafe sáng 15k" → giao dịch tay vào Tiền mặt, tự đoán danh mục, đồng bộ về app', async () => {
    const before = (await lastReply())?.message_id ?? 0
    await tg.message({ from: user, text: 'cafe sáng 15k' })
    const reply = await waitReply(/cafe sáng/i, before)
    assert.match(reply.text, /15\.000/)
    const t = await waitFor(() => txByMerchant('Cafe sáng'))
    assert.equal(Number(t.amount), 15_000)
    assert.equal(t.direction, 'out')
    assert.deepEqual(t.sources, ['manual'])
    const cash = await row`select id, balance from thuchi_accounts where owner_id = ${s.owner} and kind = 'cash'`
    assert.equal(t.account_id, cash.id)
    assert.equal(Number(cash.balance), -15_000)
    assert.ok(t.category_id, 'đã đoán danh mục')
    await s.tab('Giao dịch')
    await s.sync()
    await s.expectText('Cafe sáng')
  })

  step('trả lời tin của bot bằng "20k" → sửa số tiền', async () => {
    const reply = await waitReply(/cafe sáng/i)
    await tg.message({ from: user, text: '20k', replyTo: reply.message_id })
    await waitFor(async () => Number((await txByMerchant('Cafe sáng'))?.amount) === 20_000, { message: 'số tiền sửa thành 20k' })
  })

  step('tắt tiện ích → bot báo đang tắt, không ghi; bật lại → ghi bình thường', async () => {
    const off = await s.api('PUT', '/plugins/thu-chi/enabled', { enabled: false })
    assert.equal(off.status, 200, JSON.stringify(off.body))
    let before = (await lastReply())?.message_id ?? 0
    await tg.message({ from: user, text: 'bánh mì 20k' })
    await waitReply(/đang tắt/, before)
    assert.equal(await txByMerchant('Bánh mì'), undefined)
    before = (await lastReply())?.message_id ?? 0
    await tg.message({ from: user, text: '/tong' })
    await waitReply(/đang tắt/, before)

    const on = await s.api('PUT', '/plugins/thu-chi/enabled', { enabled: true })
    assert.equal(on.status, 200)
    before = (await lastReply())?.message_id ?? 0
    await tg.message({ from: user, text: 'bánh mì 20k' })
    await waitReply(/bánh mì/i, before)
    const t = await waitFor(() => txByMerchant('Bánh mì'))
    assert.equal(Number(t.amount), 20_000)
  })

  step('người Telegram chưa kết nối được nhắc kết nối, không tạo dữ liệu cho ai', async () => {
    const stranger = { id: user.id + 1, name: 'Lạ', username: 'stranger_e2e' }
    await tg.message({ from: stranger, text: 'trà sữa 30k' })
    await waitFor(async () => (await tg.sent(stranger.id)).some((m) => /chưa biết bạn dùng tài khoản Thu Chi nào/.test(m.text)), { message: 'nhắc kết nối' })
    assert.deepEqual(await rows`select id from thuchi_transactions where merchant = 'trà sữa'`, [])
  })
})
