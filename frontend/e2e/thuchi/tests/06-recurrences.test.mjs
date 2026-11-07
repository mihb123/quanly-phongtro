import assert from 'node:assert/strict'
import { after, before, describe } from 'node:test'
import { closeAll, newSession, step, vnd } from '../lib/app.mjs'
import { account, category, row, rows, value, waitFor } from '../lib/db.mjs'
import { vnDate, vnDay, vnInstant, vnMonth } from '../lib/time.mjs'

describe('Ghi chép định kỳ: ghi bù, tạm dừng, kết thúc, xoá', () => {
  let s
  let cash
  let rent
  const today = vnDay()
  const day = Math.min(today.d, 28)
  const rentStart = vnMonth(-2, day)
  const rentCount = 2 + (today.d > 28 || Date.now() >= vnInstant(today, 8, 0).getTime() ? 1 : 0)
  const weeklyStart = vnDay(-42)
  const weeklyEnd = vnDay(-7)

  before(async () => {
    s = await newSession({ prefix: 'rec' })
    cash = await account.byName(s.owner, 'Tiền mặt')
  })
  after(closeAll)

  const generated = (recId) => rows`select * from thuchi_transactions where recurrence_id = ${recId} order by occurred_at`

  async function openRecurrence(name) {
    await s.settings('Định kỳ')
    await s.click({ text: name, exact: false, selector: 'main section button', within: 'page' })
    await s.waitSheet('Sửa định kỳ')
  }

  step('thêm định kỳ hằng tháng bắt đầu 2 tháng trước: báo trước và ghi bù các kỳ đã đến hạn', async () => {
    await s.settings('Định kỳ')
    await s.expectText('Chưa có ghi chép định kỳ')
    await s.click('Thêm định kỳ')
    await s.waitSheet('Thêm định kỳ')
    await s.fill('#rec-amount', '3tr')
    await s.fill('#rec-merchant', 'Tiền nhà')
    await s.click({ text: 'Nhà cửa', selector: 'button' })
    await s.setValue('#rec-start', vnDate(rentStart))
    await s.expectText(`Ngày bắt đầu đã qua: sẽ ghi ngay ${rentCount} giao dịch đã đến hạn`)
    await s.click('Lưu')
    await s.toast('Đã lưu định kỳ')

    rent = await waitFor(() => row`select * from thuchi_recurrences where owner_id = ${s.owner} and merchant = 'Tiền nhà'`, { message: 'định kỳ lên server' })
    assert.equal(rent.amount, 3_000_000)
    assert.equal(rent.direction, 'out')
    assert.equal(rent.freq, 'monthly')
    assert.equal(rent.freq_interval, 1)
    assert.equal(rent.active, true)
    assert.equal(rent.account_id, cash.id)
    assert.equal(new Date(rent.start_at).getTime(), vnInstant(rentStart, 8, 0).getTime())
    const txs = await waitFor(async () => {
      const r = await generated(rent.id)
      return r.length === rentCount && r
    }, { message: `${rentCount} giao dịch sinh từ định kỳ` })
    const nhaCua = await category.byKey(s.owner, 'nha_cua')
    txs.forEach((t, i) => {
      assert.equal(t.merchant, 'Tiền nhà')
      assert.equal(t.amount, 3_000_000)
      assert.equal(t.category_id, nhaCua.id)
      assert.deepEqual(t.sources, ['manual'])
      assert.equal(new Date(t.occurred_at).getTime(), vnInstant(vnMonth(-2 + i, day), 8, 0).getTime())
    })
    rent = await row`select * from thuchi_recurrences where id = ${rent.id}`
    assert.ok(new Date(rent.next_at) > new Date(), 'next_at ở tương lai')
    await waitFor(async () => (await account.get(cash.id)).balance === -3_000_000 * rentCount, { message: 'số dư trừ đủ các kỳ' })
  })

  step('danh sách định kỳ và giao dịch sinh ra hiện badge "Định kỳ"', async () => {
    await s.sync()
    await s.expectText('Tiền nhà')
    await s.expectText(`Hằng tháng · tiếp theo ${new Date(rent.next_at).toLocaleDateString('vi-VN', { timeZone: 'Asia/Ho_Chi_Minh' })}`)
    await s.expectText(vnd(-3_000_000))
    await s.tab('Giao dịch')
    await s.click('Mọi lúc')
    await s.expectText(`${rentCount} giao dịch`)
    await s.expectText('Định kỳ')
    await s.openTx('Tiền nhà')
    await s.expectText('Tự ghi từ định kỳ hằng tháng “Tiền nhà”')
    await s.closeSheet()
    await s.click('Mọi lúc')
  })

  step('tạm dừng định kỳ', async () => {
    await openRecurrence('Tiền nhà')
    await s.toggle('Đang bật')
    await s.click('Lưu')
    await s.toast('Đã lưu định kỳ')
    await waitFor(async () => (await row`select active from thuchi_recurrences where id = ${rent.id}`).active === false, { message: 'active = false' })
    await s.expectText('Hằng tháng · tạm dừng')
  })

  step('định kỳ 2 tuần/lần có ngày kết thúc: chỉ ghi các kỳ trước ngày kết thúc rồi tự tắt', async () => {
    await s.click('Thêm định kỳ')
    await s.waitSheet('Thêm định kỳ')
    await s.click({ text: 'Thu', selector: 'button' })
    await s.fill('#rec-amount', '1tr')
    await s.fill('#rec-merchant', 'Lương tuần')
    await s.click({ text: 'Lương', selector: 'button' })
    await s.choose({ label: 'Lặp lại' }, 'Hằng tuần')
    await s.expectText('Mỗi bao nhiêu tuần')
    await s.fill('#rec-interval', '2')
    await s.setValue('#rec-start', vnDate(weeklyStart))
    await s.setValue('#rec-end', vnDate(weeklyEnd))
    await s.expectText('sẽ ghi ngay 3 giao dịch đã đến hạn')
    await s.click('Lưu')
    await s.toast('Đã lưu định kỳ')
    const weekly = await waitFor(() => row`select * from thuchi_recurrences where owner_id = ${s.owner} and merchant = 'Lương tuần'`)
    assert.equal(weekly.freq, 'weekly')
    assert.equal(weekly.freq_interval, 2)
    assert.equal(weekly.direction, 'in')
    const txs = await waitFor(async () => {
      const r = await generated(weekly.id)
      return r.length === 3 && r
    }, { message: '3 giao dịch tuần' })
    txs.forEach((t, i) => assert.equal(new Date(t.occurred_at).getTime(), vnInstant(vnDay(-42 + 14 * i), 8, 0).getTime()))
    await waitFor(async () => (await row`select active from thuchi_recurrences where id = ${weekly.id}`).active === false, { message: 'quá ngày kết thúc thì tự tắt' })
    await s.sync()
    await s.expectText('Mỗi 2 tuần · đã kết thúc')
  })

  step('gia hạn (bỏ ngày kết thúc) thì tự bật lại', async () => {
    await openRecurrence('Lương tuần')
    await s.expectText('Định kỳ đã kết thúc. Gia hạn ngày kết thúc (hoặc để trống) để tự bật lại.')
    await s.setValue('#rec-end', '')
    await s.click('Lưu')
    await s.toast('Đã lưu định kỳ')
    const r = await waitFor(async () => {
      const x = await row`select * from thuchi_recurrences where owner_id = ${s.owner} and merchant = 'Lương tuần'`
      return x.active && x
    }, { message: 'bật lại' })
    assert.equal(r.end_at, null)
    await s.sync()
    await s.expectText('Mỗi 2 tuần · tiếp theo')
  })

  step('xoá định kỳ không xoá giao dịch đã ghi', async () => {
    await openRecurrence('Tiền nhà')
    await s.click({ label: 'Xoá định kỳ', selector: 'button' })
    await s.toast('Đã xoá định kỳ, các giao dịch đã ghi vẫn giữ nguyên')
    await waitFor(async () => (await row`select deleted_at from thuchi_recurrences where id = ${rent.id}`).deleted_at, { message: 'định kỳ tombstone' })
    assert.equal(await value`select count(*)::int from thuchi_transactions where recurrence_id = ${rent.id} and deleted_at is null`, rentCount)
    await s.expectNoText('Tiền nhà', { within: 'main section' })
    await s.tab('Giao dịch')
    await s.click('Mọi lúc')
    await s.openTx('Tiền nhà')
    await s.expectText('Tự ghi từ định kỳ (đã xoá)')
    await s.closeSheet()
    await s.click('Mọi lúc')
  })
})
