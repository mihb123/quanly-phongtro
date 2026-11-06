import assert from 'node:assert/strict'
import { after, before, describe } from 'node:test'
import { closeAll, compact, newSession, step, vnd } from '../lib/app.mjs'
import { rangeTotals, value, waitFor } from '../lib/db.mjs'
import { monthLabel, shortMonth, vnDate, vnDay, vnInput, vnInstant, vnMonth } from '../lib/time.mjs'

describe('Thống kê: Danh mục theo kỳ, Xu hướng nhiều tháng, Lịch chi tiêu', () => {
  let s
  const today = vnDay()
  const m1 = vnMonth(-1, 10)
  const m2 = vnMonth(-2, 20)
  const pct = (v, total) => `${Math.round((v / total) * 100)}%`

  before(async () => {
    s = await newSession({ prefix: 'stats' })
  })
  after(closeAll)

  async function statsView(name) {
    await s.tab('Thống kê')
    await s.click({ text: name, selector: 'main button', within: 'page' })
  }

  step('nhập dữ liệu 3 tháng qua form', async () => {
    await s.addTx({ amount: '100k', title: 'Cơm trưa', category: 'Ăn uống', when: vnInput(today, 0, 30) })
    await s.addTx({ amount: '50k', title: 'Xe ôm', category: 'Đi lại', when: vnInput(today, 0, 40) })
    await s.addTx({ kind: 'Thu', amount: '10tr', title: 'Lương', category: 'Lương', when: vnInput(today, 0, 50) })
    await s.addTx({ amount: '200k', title: 'Áo khoác', category: 'Mua sắm', when: vnInput(m1, 9, 0) })
    await s.addTx({ kind: 'Thu', amount: '8tr', title: 'Lương tháng trước', category: 'Lương', when: vnInput(m1, 9, 30) })
    await s.addTx({ amount: '300k', title: 'Tiền điện', category: 'Hoá đơn & tiện ích', when: vnInput(m2, 8, 0) })
    await waitFor(async () => (await value`select count(*)::int from thuchi_transactions where owner_id = ${s.owner} and change_seq > 0`) === 6, { message: '6 giao dịch lên server' })
  })

  step('Danh mục theo tháng: tỉ lệ % và tổng ở giữa biểu đồ khớp DB', async () => {
    await statsView('Danh mục')
    const totals = await rangeTotals(s.owner, vnInstant({ ...today, d: 1 }, 0), vnInstant({ ...vnMonth(1) }, 0))
    assert.equal(totals.expense, 150_000)
    await s.expectSection(monthLabel(today), ['Tổng chi', compact(totals.expense), `Ăn uống ${pct(100, 150)} 100.000 ₫`, `Đi lại ${pct(50, 150)} 50.000 ₫`])
    await s.click({ text: 'Thu', selector: 'main button', within: 'page' })
    await s.expectSection(monthLabel(today), ['Tổng thu', '10tr', 'Lương 100% 10.000.000 ₫'])
    await s.click({ text: 'Chi', selector: 'main button', within: 'page' })
  })

  step('Danh mục 3 tháng gần nhất và năm nay', async () => {
    await s.choose({ current: 'Theo tháng', within: 'page' }, '3 tháng gần nhất')
    const title = `${shortMonth(vnMonth(-2))} – ${shortMonth(today)}`
    await s.expectSection(title, ['Tổng chi', '650k', `Hoá đơn & tiện ích ${pct(300, 650)} 300.000 ₫`, `Mua sắm ${pct(200, 650)} 200.000 ₫`, `Ăn uống ${pct(100, 650)} 100.000 ₫`, `Đi lại ${pct(50, 650)} 50.000 ₫`])
    await s.choose({ current: '3 tháng gần nhất', within: 'page' }, 'Năm nay')
    const year = await rangeTotals(s.owner, vnInstant({ y: today.y, m: 1, d: 1 }, 0), vnInstant({ y: today.y + 1, m: 1, d: 1 }, 0))
    await s.expectSection(`Năm ${today.y}`, ['Tổng chi', compact(year.expense)])
  })

  step('Danh mục khoảng tuỳ chọn, kỳ đã chọn được nhớ khi tải lại', async () => {
    await s.choose({ current: 'Năm nay', within: 'page' }, 'Tuỳ chọn')
    const last = new Date(Date.UTC(m1.y, m1.m, 0)).getUTCDate()
    await s.setValue('input[aria-label="Từ ngày"]', vnDate({ ...m1, d: 1 }))
    await s.setValue('input[aria-label="Đến ngày"]', vnDate({ ...m1, d: last }))
    const title = `1/${m1.m}/${m1.y} – ${last}/${m1.m}/${m1.y}`
    await s.expectSection(title, ['Tổng chi', '200k', 'Mua sắm 100% 200.000 ₫'])
    await s.page.reload({ waitUntil: 'domcontentloaded' })
    await s.waitReady()
    await s.expectSection(title, 'Mua sắm 100% 200.000 ₫')
    await s.choose({ current: 'Tuỳ chọn', within: 'page' }, 'Theo tháng')
  })

  step('Xu hướng 6 tháng: bảng theo tháng, tổng, trung bình và chênh lệch', async () => {
    await statsView('Xu hướng')
    const start = vnMonth(-5)
    await s.expectSection(`${shortMonth(start)} – ${shortMonth(today)}`, [])
    const expense = 650_000
    const income = 18_000_000
    await s.expectText(`Tổng chi ${vnd(expense)} TB ${compact(Math.round(expense / 6))}/tháng`)
    await s.expectText(`Tổng thu ${vnd(income)} TB ${compact(Math.round(income / 6))}/tháng`)
    await s.expectText(`${shortMonth(today)} 150k 10tr`)
    await s.expectText(`${shortMonth(m1)} 200k 8tr`)
    await s.expectText(`${shortMonth(m2)} 300k 0`)
    await s.expectText(`Tổng · chênh ${compact(income - expense)} ${compact(expense)} ${compact(income)}`)
    await s.click({ text: '12 tháng', selector: 'main button', within: 'page' })
    await s.expectText(`${shortMonth(vnMonth(-11))} – ${shortMonth(today)}`)
    await s.expectText(`TB ${compact(Math.round(expense / 12))}/tháng`)
    await s.click({ text: '6 tháng', selector: 'main button', within: 'page' })
  })

  step('Lịch: ô ngày hiện tổng chi, chọn ngày xem giao dịch, lọc theo danh mục', async () => {
    await statsView('Lịch')
    const cell = () => s.page.evaluate((d) => [...document.querySelectorAll('main section .grid button')].map((b) => window.__e2e.norm(b.innerText)).find((t) => t.startsWith(`${d} `) || t === `${d}`), today.d)
    await waitFor(async () => (await cell()) === `${today.d} 150k`, { message: `ô ngày ${today.d} = 150k` })
    await s.click({ text: `${today.d} 150k`, selector: 'main section .grid button', within: 'page' })
    await s.expectSection('Hôm nay', ['150.000 ₫', 'Cơm trưa', 'Xe ôm', 'Lương'])
    await s.choose({ current: 'Tất cả danh mục', within: 'page' }, 'Ăn uống')
    await waitFor(async () => (await cell()) === `${today.d} 100k`, { message: 'lọc Ăn uống còn 100k' })
    await s.expectSection('Hôm nay', ['100.000 ₫', 'Cơm trưa'])
    const day = await s.section('Hôm nay')
    assert.ok(!day.includes('Xe ôm') && !day.includes('Lương'), 'lọc danh mục chỉ còn giao dịch Ăn uống')
  })

  step('chế độ thống kê đang mở được nhớ khi tải lại', async () => {
    await s.page.reload({ waitUntil: 'domcontentloaded' })
    await s.waitReady()
    await s.expectText('Tất cả danh mục')
    await s.expectText('T2 T3 T4 T5 T6 T7 CN')
  })
})
