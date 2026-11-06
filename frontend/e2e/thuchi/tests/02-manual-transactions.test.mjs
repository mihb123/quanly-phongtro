import assert from 'node:assert/strict'
import { readdirSync, readFileSync } from 'node:fs'
import path from 'node:path'
import { after, before, describe } from 'node:test'
import { closeAll, GEO, newSession, step, vnd } from '../lib/app.mjs'
import { account, category, row, rows, tx, value, waitFor } from '../lib/db.mjs'
import { monthLabel, vnDay, vnInput, vnInstant, vnMonth } from '../lib/time.mjs'

describe('Giao dịch nhập tay: thêm, sửa, phân loại, lọc, xoá, xuất CSV', () => {
  let s
  let cash
  const today = vnDay()
  const selectedCategories = () => s.page.evaluate(() => [...(window.__e2e.layers().at(-1)?.querySelectorAll('button.border-primary') ?? [])].map((b) => window.__e2e.norm(b.innerText)))

  before(async () => {
    s = await newSession({ prefix: 'manual' })
    cash = await account.byName(s.owner, 'Tiền mặt')
  })
  after(closeAll)

  step('thêm khoản chi: "45k" được hiểu là 45.000 ₫, nội dung "Phở Thìn" tự gợi ý Ăn uống', async () => {
    await s.click({ label: 'Thêm giao dịch', selector: 'button', within: 'page' })
    await s.waitSheet('Thêm giao dịch')
    await s.fill('#qa-amount', '45k')
    await s.expectText('45.000 ₫')
    await s.fill('#qa-title', 'Phở Thìn')
    await waitFor(async () => (await selectedCategories()).includes('Ăn uống'), { message: 'gợi ý danh mục Ăn uống' })
    await s.click({ text: 'Lưu', selector: 'button[type="submit"]' })
    await s.toast('Đã lưu giao dịch')

    const t = await waitFor(() => tx.byMerchant(s.owner, 'Phở Thìn'), { message: 'giao dịch Phở Thìn lên server' })
    const anUong = await category.byKey(s.owner, 'an_uong')
    assert.equal(t.amount, 45000)
    assert.equal(t.direction, 'out')
    assert.equal(t.kind, 'normal')
    assert.deepEqual(t.sources, ['manual'])
    assert.equal(t.category_id, anUong.id)
    assert.equal(t.account_id, cash.id)
    assert.ok(Number(t.change_seq) > 0)
    await waitFor(async () => (await account.get(cash.id)).balance === -45000, { message: 'số dư Tiền mặt = −45.000' })
  })

  step('Tổng quan hiện tổng chi, số dư tài khoản tay và danh mục chi nhiều nhất', async () => {
    await s.expectSection(`Chi ${monthLabel(today).toLowerCase()}`, ['45.000 ₫', 'Thu 0'])
    await s.expectSection('Tài khoản', ['Tiền mặt', '−45.000 ₫'])
    await s.expectSection('Chi nhiều nhất', ['Ăn uống', '100%', '45.000 ₫'])
  })

  step('thêm khoản thu "15tr" Lương, số dư và tổng thu cập nhật', async () => {
    await s.addTx({ kind: 'Thu', amount: '15tr', title: 'Lương tháng này', category: 'Lương' })
    const t = await waitFor(() => tx.byMerchant(s.owner, 'Lương tháng này'))
    assert.equal(t.amount, 15_000_000)
    assert.equal(t.direction, 'in')
    assert.equal(t.category_id, (await category.byKey(s.owner, 'luong', 'income')).id)
    await waitFor(async () => (await account.get(cash.id)).balance === 14_955_000, { message: 'số dư Tiền mặt = 14.955.000' })
    await s.expectSection(`Chi ${monthLabel(today).toLowerCase()}`, ['45.000 ₫', 'Thu 15tr'])
    await s.expectSection('Tài khoản', ['Tiền mặt', '14.955.000 ₫'])
  })

  step('màn Giao dịch nhóm theo ngày, hiện tổng chi/thu và số giao dịch', async () => {
    await s.tab('Giao dịch')
    await s.expectText('2 giao dịch')
    await s.expectText('Chi 45.000 ₫ · Thu 15.000.000 ₫')
    await s.expectText('Hôm nay')
    await s.expectText('Phở Thìn')
    await s.expectText('−45.000 ₫')
    await s.expectText('+15.000.000 ₫')
    await s.expectNoText('Chưa đồng bộ')
  })

  step('sửa giao dịch tay: số tiền, nội dung, thời gian → server áp lại chênh lệch số dư', async () => {
    await s.openTx('Phở Thìn')
    await s.expectText('Nguồn: manual')
    await s.click('Sửa')
    await s.waitSheet('Sửa giao dịch')
    await s.fill('#qa-amount', '50000')
    await s.fill('#qa-title', 'Phở Thìn Lò Đúc')
    await s.setValue('#qa-when', vnInput(today, 6, 0))
    await s.click({ text: 'Lưu', selector: 'button[type="submit"]' })
    await s.toast('Đã cập nhật giao dịch')
    const t = await waitFor(() => tx.byMerchant(s.owner, 'Phở Thìn Lò Đúc'), { message: 'giao dịch đã sửa' })
    assert.equal(t.amount, 50000)
    assert.equal(new Date(t.occurred_at).getTime(), vnInstant(today, 6, 0).getTime())
    for (const f of ['amount', 'merchant', 'occurred_at']) assert.ok(t.field_ts[f], `field_ts.${f}`)
    assert.equal(await value`select count(*)::int from thuchi_transactions where owner_id = ${s.owner}`, 2, 'sửa không tạo bản ghi mới')
    await waitFor(async () => (await account.get(cash.id)).balance === 14_950_000, { message: 'số dư Tiền mặt = 14.950.000' })
    await s.expectText('Phở Thìn Lò Đúc')
    await s.expectText('06:00')
    await s.expectText('Chi 50.000 ₫')
  })

  step('ghi chú, "Không tính vào thống kê" và gắn vị trí lưu lên server', async () => {
    await s.openTx('Phở Thìn Lò Đúc')
    await s.fill('#tx-note', 'ăn với đồng nghiệp')
    await s.blur()
    await s.toggle('Không tính vào thống kê')
    await s.click('Gắn vị trí hiện tại')
    await s.toast('Đã gắn vị trí hiện tại')
    const t = await waitFor(async () => {
      const r = await tx.byMerchant(s.owner, 'Phở Thìn Lò Đúc')
      return r.note === 'ăn với đồng nghiệp' && r.excluded && r.lat !== null && r
    }, { message: 'note/excluded/lat lên server' })
    assert.ok(Math.abs(t.lat - GEO.latitude) < 1e-6 && Math.abs(t.lng - GEO.longitude) < 1e-6)
    await s.expectText('Cập nhật vị trí')
    await s.closeSheet()
    await s.expectText('Không tính')
    await s.expectText('Chi 0 ₫ · Thu 15.000.000 ₫')
    assert.equal((await account.get(cash.id)).balance, 14_950_000, 'không tính thống kê vẫn giữ số dư')
    await s.tab('Thống kê')
    await s.click('Bản đồ')
    await s.expectText('Tháng này có 1 giao dịch đã gắn vị trí')
    await s.tab('Tổng quan')
    await s.expectSection(`Chi ${monthLabel(today).toLowerCase()}`, '0 ₫')
  })

  step('đổi danh mục kèm "Áp dụng cho giao dịch giống vậy" tạo luật tự học, lần nhập sau tự gợi ý', async () => {
    await s.tab('Giao dịch')
    await s.openTx('Phở Thìn Lò Đúc')
    await s.toggle('Không tính vào thống kê')
    assert.equal(await s.isChecked('Áp dụng cho giao dịch giống vậy', { kind: 'checkbox' }), true)
    await s.click({ text: 'Bữa lớn', selector: 'button' })
    await s.toast('Đã phân loại và ghi nhớ cho giao dịch tương tự')
    const buaLon = await category.byKey(s.owner, 'bua_lon')
    await waitFor(async () => {
      const r = await tx.byMerchant(s.owner, 'Phở Thìn Lò Đúc')
      return r.category_id === buaLon.id && !r.excluded
    }, { message: 'danh mục Bữa lớn + bỏ "không tính"' })
    const rule = await waitFor(() => row`select * from thuchi_rules where owner_id = ${s.owner} and deleted_at is null`, { message: 'luật tự học' })
    assert.equal(rule.match, 'merchant')
    assert.equal(rule.origin, 'learned')
    assert.equal(rule.pattern, 'Phở Thìn Lò Đúc')
    assert.equal(rule.category_id, buaLon.id)

    await s.click({ label: 'Thêm giao dịch', selector: 'button', within: 'page' })
    await s.waitSheet('Thêm giao dịch')
    await s.fill('#qa-amount', '1')
    await s.fill('#qa-title', 'Phở Thìn Lò Đúc')
    await waitFor(async () => (await selectedCategories()).includes('Bữa lớn'), { message: 'gợi ý theo luật tự học' })
    await s.closeSheet()
    await s.settings('Luật phân loại')
    await s.expectText('Đúng tên → Bữa lớn · tự học')
  })

  step('tìm kiếm theo nội dung/số tiền và lọc theo danh mục', async () => {
    await s.tab('Giao dịch')
    await s.fill('input[placeholder="Tìm nội dung, số tiền…"]', 'lo duc')
    await s.expectText('1 giao dịch')
    await s.expectNoText('Lương tháng này')
    await s.fill('input[placeholder="Tìm nội dung, số tiền…"]', '15000000')
    await s.expectText('Lương tháng này')
    await s.expectText('1 giao dịch')
    await s.click({ label: 'Xoá bộ lọc', selector: 'button', within: 'page' })
    await s.expectText('2 giao dịch')
    await s.choose({ current: 'Mọi danh mục', within: 'page' }, 'Lương (thu)')
    await s.expectText('1 giao dịch')
    await s.expectText('Lương tháng này')
    await s.click({ label: 'Xoá bộ lọc', selector: 'button', within: 'page' })
    await s.expectText('2 giao dịch')
  })

  step('"Mọi lúc" hiện cả giao dịch tháng khác và được giữ khi tải lại trang', async () => {
    const old = vnMonth(-2, 15)
    await s.addTx({ amount: '320k', title: 'Sửa xe tháng cũ', category: 'Đi lại', when: vnInput(old, 10, 0) })
    const t = await waitFor(() => tx.byMerchant(s.owner, 'Sửa xe tháng cũ'))
    assert.equal(new Date(t.occurred_at).getTime(), vnInstant(old, 10, 0).getTime())
    await s.expectText('2 giao dịch')
    await s.expectNoText('Sửa xe tháng cũ')
    await s.click('Mọi lúc')
    await s.expectText('Mọi thời gian')
    await s.expectText('3 giao dịch')
    await s.expectText('Sửa xe tháng cũ')
    await s.page.reload({ waitUntil: 'domcontentloaded' })
    await s.waitReady()
    await s.expectText('Mọi thời gian')
    await s.expectText('Sửa xe tháng cũ')
    await s.click('Mọi lúc')
    await s.expectText('Trong tháng')
  })

  step('chọn tháng ở header: tháng trước hiện số liệu của tháng đó, không đi tới tháng tương lai', async () => {
    const prev = vnMonth(-1, 10)
    await s.addTx({ amount: '120k', title: 'Siêu thị tháng trước', when: vnInput(prev, 9, 30) })
    const t = await waitFor(() => tx.byMerchant(s.owner, 'Siêu thị tháng trước'))
    assert.equal(t.category_id, (await category.byKey(s.owner, 'mua_sam')).id, 'gợi ý Mua sắm từ "Siêu thị"')
    await s.tab('Tổng quan')
    assert.equal(await s.page.$eval('[aria-label="Tháng sau"]', (b) => b.disabled), true)
    await s.click({ label: 'Tháng trước', selector: 'button', within: 'page' })
    await s.expectText(monthLabel(prev), { within: 'header' })
    await s.expectSection(`Chi ${monthLabel(prev).toLowerCase()}`, '120.000 ₫')
    await s.expectSection('Chi nhiều nhất', ['Mua sắm', '120.000 ₫'])
    await s.tab('Giao dịch')
    await s.expectText('Siêu thị tháng trước')
    await s.expectText('1 giao dịch')
    await s.click({ text: monthLabel(prev), selector: 'header button', within: 'page' })
    await s.expectText(monthLabel(today), { within: 'header' })
    await s.expectText('2 giao dịch')
  })

  step('xoá giao dịch tay = tombstone trên server, hoàn lại số dư', async () => {
    const before = (await account.get(cash.id)).balance
    await s.openTx('Lương tháng này')
    await s.click('Xoá')
    await s.toast('Đã xoá giao dịch')
    const t = await waitFor(async () => {
      const r = await row`select * from thuchi_transactions where owner_id = ${s.owner} and merchant = 'Lương tháng này'`
      return r?.deleted_at && r
    }, { message: 'tombstone deleted_at' })
    assert.ok(t.field_ts.deleted_at)
    await waitFor(async () => (await account.get(cash.id)).balance === before - 15_000_000, { message: 'hoàn số dư' })
    await s.expectNoText('Lương tháng này')
    await s.expectText('1 giao dịch')
    await s.tab('Tổng quan')
    await s.expectSection(`Chi ${monthLabel(today).toLowerCase()}`, ['50.000 ₫', 'Thu 0'])
    const live = await tx.live(s.owner)
    const balance = live.reduce((sum, r) => sum + (r.direction === 'in' ? r.amount : -r.amount), 0)
    assert.equal((await account.get(cash.id)).balance, balance, 'số dư = tổng giao dịch còn lại')
    await s.expectSection('Tài khoản', vnd(balance))
  })

  step('xuất CSV chứa đúng các giao dịch trên máy', async () => {
    const dir = await s.enableDownloads()
    await s.settings()
    await s.click({ text: 'Xuất dữ liệu (CSV)', exact: false })
    const file = await waitFor(() => readdirSync(dir).find((f) => f.endsWith('.csv')), { message: 'file CSV tải về' })
    assert.match(file, /^thu-chi-\d{4}-\d{2}-\d{2}\.csv$/)
    const csv = readFileSync(path.join(dir, file), 'utf8')
    const lines = csv.replace(/^﻿/, '').trim().split('\n')
    assert.equal(lines[0], 'Thời gian,Tài khoản,Chiều,Số tiền,Danh mục,Loại,Nội dung,Ghi chú')
    const live = await rows`select merchant, amount from thuchi_transactions where owner_id = ${s.owner} and deleted_at is null`
    assert.equal(lines.length - 1, live.length)
    for (const r of live) assert.ok(lines.some((l) => l.includes(`"${r.merchant}"`) && l.includes(`"${r.amount}"`)), `CSV có ${r.merchant}`)
    assert.ok(lines.some((l) => l.includes('"Bữa lớn"') && l.includes('"ăn với đồng nghiệp"')))
    assert.ok(!csv.includes('Lương tháng này'), 'giao dịch đã xoá không có trong CSV')
  })
})
