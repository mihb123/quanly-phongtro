import assert from 'node:assert/strict'
import { after, before, describe } from 'node:test'
import { closeAll, newSession, step } from '../lib/app.mjs'
import { account, tx, waitFor } from '../lib/db.mjs'
import { monthLabel, vnDay } from '../lib/time.mjs'

describe('Vay nợ: cho vay, thu nợ, đi vay, trả nợ theo từng người', () => {
  let s
  let cash
  const heroTitle = `Chi ${monthLabel(vnDay()).toLowerCase()}`

  before(async () => {
    s = await newSession({ prefix: 'debt' })
    cash = await account.byName(s.owner, 'Tiền mặt')
  })
  after(closeAll)

  async function debtView() {
    await s.tab('Thống kê')
    await s.click({ text: 'Vay nợ', selector: 'main button', within: 'page' })
    await s.expectText('Theo người')
  }

  step('cho vay: bắt buộc nhập người, lưu kind lend chiều ra, không có danh mục', async () => {
    await s.tab('Giao dịch')
    await s.click({ label: 'Thêm giao dịch', selector: 'button', within: 'page' })
    await s.waitSheet('Thêm giao dịch')
    await s.click('Vay nợ')
    await s.expectNoText('Danh mục')
    await s.fill('#qa-amount', '2tr')
    await s.fill('#qa-title', 'Mượn sửa xe')
    assert.equal(await s.page.$eval('button[type="submit"]', (b) => b.disabled), true, 'chưa nhập người thì không lưu được')
    await s.fill('#qa-party', 'Anh Nam')
    await s.click({ text: 'Lưu', selector: 'button[type="submit"]' })
    await s.toast('Đã lưu giao dịch')
    const t = await waitFor(() => tx.byMerchant(s.owner, 'Mượn sửa xe'))
    assert.equal(t.kind, 'lend')
    assert.equal(t.direction, 'out')
    assert.equal(t.party, 'Anh Nam')
    assert.equal(t.category_id, null)
    await waitFor(async () => (await account.get(cash.id)).balance === -2_000_000, { message: 'số dư −2tr' })
  })

  step('thu nợ chọn lại người qua gợi ý tên đã dùng', async () => {
    await s.click({ label: 'Thêm giao dịch', selector: 'button', within: 'page' })
    await s.waitSheet('Thêm giao dịch')
    await s.click('Vay nợ')
    await s.click('Thu nợ')
    await s.fill('#qa-party', 'anh')
    await s.click({ text: 'Anh Nam', selector: 'button' })
    assert.equal(await s.inputValue('#qa-party'), 'Anh Nam')
    await s.fill('#qa-amount', '500k')
    await s.fill('#qa-title', 'Anh Nam trả một phần')
    await s.click({ text: 'Lưu', selector: 'button[type="submit"]' })
    await s.toast('Đã lưu giao dịch')
    const t = await waitFor(() => tx.byMerchant(s.owner, 'Anh Nam trả một phần'))
    assert.equal(t.kind, 'collect')
    assert.equal(t.direction, 'in')
    assert.equal(t.party, 'Anh Nam')
  })

  step('đi vay rồi trả hết', async () => {
    await s.addTx({ kind: 'Vay nợ', debtKind: 'Đi vay', party: 'Chị Lan', amount: '1tr', title: 'Vay chị Lan' })
    await s.addTx({ kind: 'Vay nợ', debtKind: 'Trả nợ', party: 'Chị Lan', amount: '1tr', title: 'Trả chị Lan' })
    const borrow = await waitFor(() => tx.byMerchant(s.owner, 'Vay chị Lan'))
    const repay = await waitFor(() => tx.byMerchant(s.owner, 'Trả chị Lan'))
    assert.deepEqual([borrow.kind, borrow.direction], ['borrow', 'in'])
    assert.deepEqual([repay.kind, repay.direction], ['repay', 'out'])
    await waitFor(async () => (await account.get(cash.id)).balance === -1_500_000, { message: 'số dư −1,5tr' })
  })

  step('vay nợ không tính vào thu/chi nhưng vẫn đổi số dư', async () => {
    await s.tab('Tổng quan')
    await s.expectSection(heroTitle, ['0 ₫', 'Thu 0'])
    await s.expectSection('Tài khoản', ['Tiền mặt', '−1.500.000 ₫'])
    await s.tab('Giao dịch')
    await s.expectText('4 giao dịch')
    await s.expectText('Chi 0 ₫ · Thu 0 ₫')
    await s.expectText('Cho vay Anh Nam')
    await s.expectText('Trả nợ Chị Lan')
  })

  step('màn Vay nợ: cần thu/cần trả theo từng người, bấm xem lịch sử', async () => {
    await debtView()
    await s.expectText('Cần thu 1.500.000 ₫')
    await s.expectText('Cần trả 0 ₫')
    await s.expectText('Anh Nam cho vay 2tr · đã thu 500k Thu 1.500.000 ₫')
    await s.expectText('Chị Lan đi vay 1tr · đã trả 1tr Đã xong')
    await s.click({ text: 'Anh Nam', exact: false, selector: 'button[aria-expanded]', within: 'page' })
    await s.expectText('Mượn sửa xe')
    await s.expectText('Anh Nam trả một phần')
  })

  step('đổi giao dịch chi thường thành cho vay ngay trong chi tiết giao dịch, mở lại để nhập người', async () => {
    await s.addTx({ amount: '300k', title: 'Ứng tiền cho Bình' })
    await s.openTx('Ứng tiền cho Bình')
    await s.expectText('Vay nợ')
    await s.click({ text: 'Cho vay', selector: 'button' })
    await waitFor(async () => (await tx.byMerchant(s.owner, 'Ứng tiền cho Bình')).kind === 'lend', { message: 'kind = lend' })
    await s.closeSheet()
    await s.openTx('Ứng tiền cho Bình')
    await s.fill('#tx-party', 'Bình')
    await s.blur()
    await waitFor(async () => (await tx.byMerchant(s.owner, 'Ứng tiền cho Bình')).party === 'Bình', { message: 'party = Bình' })
    await s.closeSheet()
    await debtView()
    await s.expectText('Cần thu 1.800.000 ₫')
    await s.expectText('Bình cho vay 300k Thu 300.000 ₫')
  })

  step(
    'đổi sang Cho vay rồi nhập người ngay trong cùng một lần mở chi tiết',
    async () => {
      await s.addTx({ amount: '200k', title: 'Ứng tiền cho Cường' })
      await s.openTx('Ứng tiền cho Cường')
      await s.click({ text: 'Cho vay', selector: 'button' })
      await waitFor(async () => (await tx.byMerchant(s.owner, 'Ứng tiền cho Cường')).kind === 'lend', { message: 'kind = lend' })
      await s.page.waitForSelector('#tx-party', { visible: true })
      assert.equal(await s.page.evaluate(() => document.querySelectorAll('#tx-party').length), 1, 'chỉ có một khối Vay nợ')
      await s.fill('#tx-party', 'Cường')
      await s.blur()
      await waitFor(async () => (await tx.byMerchant(s.owner, 'Ứng tiền cho Cường')).party === 'Cường', { message: 'party = Cường', timeout: 5000 })
    },
    { todo: 'Lỗi UI: TxSheets.tsx DebtField và NoteField là anh em cùng key={live.id}; khi đổi sang vay nợ (khối danh mục ẩn đi) React render trùng khối "Vay nợ", ô người không lưu' },
  )

  step('sửa số tiền khoản cho vay cập nhật số cần thu', async () => {
    await s.tab('Giao dịch')
    await s.openTx('Mượn sửa xe')
    await s.click('Sửa')
    await s.waitSheet('Sửa giao dịch')
    assert.equal(await s.inputValue('#qa-party'), 'Anh Nam')
    await s.fill('#qa-amount', '2,5tr')
    await s.click({ text: 'Lưu', selector: 'button[type="submit"]' })
    await s.toast('Đã cập nhật giao dịch')
    await waitFor(async () => (await tx.byMerchant(s.owner, 'Mượn sửa xe')).amount === 2_500_000, { message: 'amount = 2,5tr' })
    await debtView()
    await s.expectText('Cần thu 2.500.000 ₫')
  })
})
