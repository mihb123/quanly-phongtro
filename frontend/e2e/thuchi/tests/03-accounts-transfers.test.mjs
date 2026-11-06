import assert from 'node:assert/strict'
import { after, before, describe } from 'node:test'
import { closeAll, newSession, step, vnd } from '../lib/app.mjs'
import { account, accountHash, category, row, rows, tx, value, waitFor } from '../lib/db.mjs'
import { monthLabel, vnDay } from '../lib/time.mjs'

const TRASH = 'button[class*="text-destructive"]'

describe('Tài khoản, chuyển tiền kèm phí, điều chỉnh số dư', () => {
  let s
  let timo
  let momo
  const accountNo = '8007 0411 3678'
  const heroTitle = `Chi ${monthLabel(vnDay()).toLowerCase()}`
  const balanceIs = (id, want) => waitFor(async () => (await account.get(id)).balance === want, { message: `số dư ${id} = ${want}` })

  before(async () => {
    s = await newSession({ prefix: 'acct' })
  })
  after(closeAll)

  async function openAccount(name) {
    await s.settings('Tài khoản')
    await s.click({ text: name, exact: false, selector: 'main section button', within: 'page' })
    await s.waitSheet('Sửa tài khoản')
  }

  step('thêm tài khoản ngân hàng với số dư ban đầu, số TK chỉ lưu dạng băm + 4 số cuối', async () => {
    await s.settings('Tài khoản')
    await s.click('Thêm tài khoản')
    await s.waitSheet('Thêm tài khoản')
    await s.fill('#acc-name', 'Timo chính')
    await s.fill('#acc-bank', 'timo')
    await s.fill('#acc-no', accountNo)
    await s.fill('#acc-holder', 'NGUYEN VAN E2E')
    await s.fill('#acc-bal', '2tr')
    await s.click('Lưu')
    await s.toast('Đã lưu tài khoản')
    timo = await waitFor(() => account.byName(s.owner, 'Timo chính'), { message: 'tài khoản Timo lên server' })
    assert.equal(timo.kind, 'bank')
    assert.equal(timo.bank_code, 'TIMO')
    assert.equal(timo.connector, 'manual')
    assert.equal(timo.balance, 2_000_000)
    assert.equal(timo.holder_name, 'NGUYEN VAN E2E')
    assert.equal(timo.account_no_last4, '3678')
    assert.equal(timo.account_no_hash, accountHash(s.owner, accountNo))
    const raw = await value`select row_to_json(a)::text from thuchi_accounts a where id = ${timo.id}`
    assert.ok(!raw.includes('800704113678'), 'DB không chứa số tài khoản đầy đủ')
    const local = (await s.localRows('accounts')).find((a) => a.id === timo.id)
    assert.equal(local.account_no, undefined, 'IndexedDB không giữ số tài khoản')
    await s.expectText('Ngân hàng · ···3678 · nhập tay')
    await s.expectText('2.000.000 ₫')
  })

  step('thêm ví điện tử 500k', async () => {
    await s.click('Thêm tài khoản')
    await s.waitSheet('Thêm tài khoản')
    await s.fill('#acc-name', 'Ví MoMo')
    await s.choose({ label: 'Loại' }, 'Ví điện tử')
    await s.fill('#acc-bank', 'MOMO')
    await s.fill('#acc-bal', '500k')
    await s.click('Lưu')
    await s.toast('Đã lưu tài khoản')
    momo = await waitFor(() => account.byName(s.owner, 'Ví MoMo'))
    assert.equal(momo.kind, 'ewallet')
    assert.equal(momo.balance, 500_000)
    await s.tab('Tổng quan')
    await s.expectSection('Tài khoản', ['Timo chính', '2.000.000 ₫', 'Ví MoMo', '500.000 ₫', vnd(2_500_000)])
  })

  step('chuyển tiền giữa 2 tài khoản kèm phí: 2 giao dịch nội bộ ghép cặp + 1 khoản phí', async () => {
    await s.addTx({ kind: 'Chuyển tiền', amount: '1tr', title: 'Nạp ví', account: 'Timo chính', to: 'Ví MoMo', fee: '3.300' })
    const pair = await waitFor(async () => {
      const r = await rows`select * from thuchi_transactions where owner_id = ${s.owner} and kind = 'transfer' and deleted_at is null order by direction`
      return r.length === 2 && r
    }, { message: '2 giao dịch chuyển tiền' })
    const [inn, out] = pair
    assert.equal(out.direction, 'out')
    assert.equal(out.account_id, timo.id)
    assert.equal(inn.account_id, momo.id)
    for (const t of pair) {
      assert.equal(t.amount, 1_000_000)
      assert.equal(t.pair_status, 'confirmed')
      assert.equal(t.note, 'Nạp ví')
      assert.equal(t.category_id, null)
      assert.deepEqual(t.sources, ['manual'])
    }
    assert.equal(out.pair_id, inn.id)
    assert.equal(inn.pair_id, out.id)
    const fee = await row`select * from thuchi_transactions where owner_id = ${s.owner} and pair_id = ${out.id} and kind = 'normal' and deleted_at is null`
    assert.ok(fee, 'có giao dịch phí')
    assert.equal(fee.amount, 3300)
    assert.equal(fee.direction, 'out')
    assert.equal(fee.account_id, timo.id)
    assert.equal(fee.merchant, 'Phí chuyển khoản')
    assert.equal(fee.category_id, (await category.byKey(s.owner, 'phi_ngan_hang')).id)
    assert.equal(await value`select count(*)::int from thuchi_tx_sources where owner_id = ${s.owner} and source_ref = ${`fee:${out.id}`}`, 1)
    await balanceIs(timo.id, 2_000_000 - 1_000_000 - 3_300)
    await balanceIs(momo.id, 1_500_000)

    await s.expectSection(heroTitle, '3.300 ₫')
    await s.expectSection('Tài khoản', ['996.700 ₫', '1.500.000 ₫'])
    await s.tab('Giao dịch')
    await s.expectText('Chi 3.300 ₫ · Thu 0 ₫')
    await s.expectText('Nội bộ')
    await s.expectText('Phí chuyển khoản')
    await s.openTx('Nạp ví')
    await s.expectText('Timo chính → Ví MoMo')
    await s.expectText('Phí chuyển: 3.300 ₫')
  })

  step('sửa chuyển tiền: đổi số tiền và bỏ phí → cả cặp cập nhật, phí bị xoá mềm', async () => {
    const out = await row`select * from thuchi_transactions where owner_id = ${s.owner} and kind = 'transfer' and direction = 'out'`
    await s.click('Sửa')
    await s.waitSheet('Sửa giao dịch')
    await s.fill('#qa-amount', '1,2tr')
    await s.fill('#qa-fee', '')
    await s.click({ text: 'Lưu', selector: 'button[type="submit"]' })
    await s.toast('Đã cập nhật giao dịch')
    await waitFor(async () => {
      const r = await rows`select amount from thuchi_transactions where owner_id = ${s.owner} and kind = 'transfer' and deleted_at is null`
      return r.length === 2 && r.every((t) => t.amount === 1_200_000)
    }, { message: 'cả cặp = 1.200.000' })
    await waitFor(() => value`select deleted_at is not null from thuchi_transactions where pair_id = ${out.id} and kind = 'normal'`, { message: 'phí bị tombstone' })
    await balanceIs(timo.id, 800_000)
    await balanceIs(momo.id, 1_700_000)
    await s.expectText('Chi 0 ₫ · Thu 0 ₫')
    await s.expectNoText('Phí chuyển khoản')
  })

  step('xoá một phía của chuyển tiền xoá cả cặp và hoàn số dư', async () => {
    await s.openTx('Nạp ví')
    await s.click('Xoá')
    await s.toast('Đã xoá giao dịch')
    await waitFor(async () => (await value`select count(*)::int from thuchi_transactions where owner_id = ${s.owner} and kind = 'transfer' and deleted_at is not null`) === 2, { message: 'cả cặp tombstone' })
    await balanceIs(timo.id, 2_000_000)
    await balanceIs(momo.id, 500_000)
    await s.expectText('Không có giao dịch')
  })

  step('đổi số dư hiện tại tạo giao dịch "Điều chỉnh số dư" cho phần chênh lệch, không tính vào thu chi', async () => {
    await openAccount('Ví MoMo')
    await s.expectText('Đổi số dư sẽ tạo giao dịch “Điều chỉnh số dư”')
    await s.fill('#acc-bal', '650k')
    await s.click('Lưu')
    await s.toast('Đã lưu tài khoản')
    const adj = await waitFor(() => row`select * from thuchi_transactions where owner_id = ${s.owner} and kind = 'adjustment' and deleted_at is null`, { message: 'giao dịch điều chỉnh' })
    assert.equal(adj.direction, 'in')
    assert.equal(adj.amount, 150_000)
    assert.equal(adj.account_id, momo.id)
    await balanceIs(momo.id, 650_000)
    await s.tab('Giao dịch')
    await s.expectText('Điều chỉnh số dư')
    await s.expectText('+150.000 ₫')
    await s.expectText('Chi 0 ₫ · Thu 0 ₫')
  })

  step('tắt "Tính vào thống kê": chi trên tài khoản đó không vào tổng, số dư không cộng vào tổng tài sản', async () => {
    await openAccount('Ví MoMo')
    await s.toggle('Tính vào thống kê')
    await s.click('Lưu')
    await s.toast('Đã lưu tài khoản')
    await waitFor(async () => (await account.get(momo.id)).include_in_stats === false, { message: 'include_in_stats = false' })
    await s.addTx({ amount: '70k', title: 'Trà sữa', account: 'Ví MoMo' })
    const t = await waitFor(() => tx.byMerchant(s.owner, 'Trà sữa'))
    assert.equal(t.account_id, momo.id)
    await balanceIs(momo.id, 580_000)
    await s.tab('Tổng quan')
    await s.expectSection('Tài khoản', ['Ví MoMo', 'không tính thống kê', '580.000 ₫', vnd(2_000_000)])
    await s.expectSection(heroTitle, '0 ₫')
  })

  step('xoá tài khoản tay = tombstone, biến mất khỏi danh sách', async () => {
    await openAccount('Ví MoMo')
    await s.click({ selector: TRASH })
    await waitFor(async () => (await account.get(momo.id)).deleted_at, { message: 'tài khoản tombstone' })
    await s.waitNoSheet()
    await s.expectNoText('Ví MoMo', { within: 'main' })
    await s.tab('Tổng quan')
    await s.expectSection('Tài khoản', ['Timo chính', 'Tiền mặt'])
    assert.ok(!(await s.section('Tài khoản')).includes('Ví MoMo'))
  })
})
