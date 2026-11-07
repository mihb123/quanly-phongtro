import assert from 'node:assert/strict'
import { after, before, describe } from 'node:test'
import { closeAll, newSession, sleep, step } from '../lib/app.mjs'
import { category, row, rows, tx, waitFor } from '../lib/db.mjs'
import { monthKey, monthLabel, vnDay, vnMonth } from '../lib/time.mjs'

describe('Danh mục cha/con, luật phân loại, hạn mức kế thừa theo tháng', () => {
  let s
  let parent
  let child
  const cur = vnDay()
  const prev = vnMonth(-1)
  const selectedCategories = () => s.page.evaluate(() => [...(window.__e2e.layers().at(-1)?.querySelectorAll('button.border-primary') ?? [])].map((b) => window.__e2e.norm(b.innerText)))

  before(async () => {
    s = await newSession({ prefix: 'cat' })
  })
  after(closeAll)

  async function openCategory(name) {
    await s.settings('Danh mục')
    await s.click({ text: name, selector: 'main section button', within: 'page' })
    await s.waitSheet('Sửa danh mục')
  }

  async function setBudget(name, text) {
    const h = await s.page.waitForFunction(
      (n) => {
        const row = [...document.querySelectorAll('main section .divide-y > div')].find((d) => window.__e2e.norm(d.querySelector('span.min-w-0 > span')?.innerText) === n)
        return row?.querySelector('input')
      },
      { polling: 100 },
      name,
    )
    const el = h.asElement()
    await s.settle(el)
    await el.click()
    await s.page.keyboard.down('Control')
    await s.page.keyboard.press('a')
    await s.page.keyboard.up('Control')
    await s.page.keyboard.press('Backspace')
    if (text) await el.type(text)
    await s.blur()
    await sleep(300)
  }

  async function budgetRow(name) {
    return s.page.evaluate((n) => {
      const row = [...document.querySelectorAll('main section .divide-y > div')].find((d) => window.__e2e.norm(d.querySelector('span.min-w-0 > span')?.innerText) === n)
      return row ? { text: window.__e2e.norm(row.innerText), value: row.querySelector('input')?.value } : null
    }, name)
  }

  const budgets = (month) => rows`select b.*, c.name as category from thuchi_budgets b left join thuchi_categories c on c.id = b.category_id where b.owner_id = ${s.owner} and b.month = ${month} and b.deleted_at is null order by c.name nulls first`

  step('thêm danh mục gốc với biểu tượng + màu', async () => {
    await s.settings('Danh mục')
    await s.click({ text: 'Thêm', selector: 'main section button', within: 'page', nth: 0 })
    await s.waitSheet('Thêm danh mục')
    await s.fill('#cat-name', 'Cà phê')
    await s.click({ label: 'coffee', selector: 'button' })
    await s.click({ label: '#14b8a6', selector: 'button' })
    await s.click('Lưu')
    parent = await waitFor(() => category.byName(s.owner, 'Cà phê'), { message: 'danh mục Cà phê lên server' })
    assert.equal(parent.kind, 'expense')
    assert.equal(parent.icon, 'coffee')
    assert.equal(parent.color, '#14b8a6')
    assert.equal(parent.parent_id, null)
  })

  step('thêm danh mục con, hiện thụt dưới cha; cha đang có con thì không thể làm con', async () => {
    await s.click({ text: 'Thêm', selector: 'main section button', within: 'page', nth: 0 })
    await s.waitSheet('Thêm danh mục')
    await s.fill('#cat-name', 'Cà phê sáng')
    await s.choose({ label: 'Thuộc danh mục' }, 'Cà phê')
    await s.click('Lưu')
    child = await waitFor(() => category.byName(s.owner, 'Cà phê sáng'))
    assert.equal(child.parent_id, parent.id)
    await s.waitNoSheet()
    const indented = await s.page.evaluate(() => [...document.querySelectorAll('main section button')].find((b) => window.__e2e.norm(b.innerText) === 'Cà phê sáng')?.className.includes('pl-10'))
    assert.equal(indented, true, 'danh mục con thụt vào')
    await openCategory('Cà phê')
    await s.expectText('Danh mục đang có danh mục con nên không thể đặt làm con.')
    assert.equal(await s.page.evaluate(() => window.__e2e.findTrigger({ label: 'Thuộc danh mục' })?.disabled), true)
    await s.closeSheet()
  })

  step('luật "Nội dung chứa" do người dùng đặt được ưu tiên khi gợi ý danh mục', async () => {
    await s.settings('Luật phân loại')
    await s.expectText('Chưa có luật')
    await s.fill('input[placeholder="Nội dung chứa… vd: PHO THIN"]', 'BUN CHA')
    await s.choose({ current: 'Chọn danh mục', within: 'page' }, 'Bữa lớn')
    await s.click({ text: 'Thêm', selector: 'main button', within: 'page' })
    const rule = await waitFor(() => row`select * from thuchi_rules where owner_id = ${s.owner} and deleted_at is null`, { message: 'luật lên server' })
    assert.equal(rule.pattern, 'BUN CHA')
    assert.equal(rule.match, 'contains')
    assert.equal(rule.origin, 'manual')
    assert.equal(rule.category_id, (await category.byKey(s.owner, 'bua_lon')).id)
    await s.expectText('Nội dung chứa → Bữa lớn · tự đặt')

    await s.tab('Giao dịch')
    await s.click({ label: 'Thêm giao dịch', selector: 'button', within: 'page' })
    await s.waitSheet('Thêm giao dịch')
    await s.fill('#qa-amount', '45k')
    await s.fill('#qa-title', 'Bún chả Hương Liên')
    await waitFor(async () => (await selectedCategories()).includes('Bữa lớn'), { message: 'gợi ý Bữa lớn theo luật (thay vì Ăn uống)' })
    await s.click({ text: 'Lưu', selector: 'button[type="submit"]' })
    await s.toast('Đã lưu giao dịch')
    const t = await waitFor(() => tx.byMerchant(s.owner, 'Bún chả Hương Liên'))
    assert.equal(t.category_id, rule.category_id)
  })

  step('xoá luật = tombstone', async () => {
    await s.settings('Luật phân loại')
    await s.click({ label: 'Xoá luật', selector: 'button', within: 'page' })
    await waitFor(() => row`select 1 from thuchi_rules where owner_id = ${s.owner} and deleted_at is not null`, { message: 'luật tombstone' })
    await s.expectText('Chưa có luật')
  })

  step('lưới chọn danh mục nhóm con theo cha; thống kê gộp con vào cha, bấm để xem chi tiết', async () => {
    await s.tab('Giao dịch')
    await s.click({ label: 'Thêm giao dịch', selector: 'button', within: 'page' })
    await s.waitSheet('Thêm giao dịch')
    const group = await s.page.evaluate(() => {
      const label = [...window.__e2e.layers().at(-1).querySelectorAll('p')].find((p) => window.__e2e.norm(p.innerText) === 'Cà phê')
      return label ? [...label.parentElement.querySelectorAll('button')].map((b) => window.__e2e.norm(b.innerText)) : null
    })
    assert.deepEqual(group, ['Cà phê', 'Cà phê sáng'])
    await s.closeSheet()
    await s.addTx({ amount: '30k', title: 'Highlands sáng', category: 'Cà phê sáng' })
    await s.addTx({ amount: '20k', title: 'Cà phê muối', category: 'Cà phê' })
    await waitFor(async () => (await tx.byMerchant(s.owner, 'Cà phê muối'))?.category_id === parent.id)
    assert.equal((await tx.byMerchant(s.owner, 'Highlands sáng')).category_id, child.id)

    await s.choose({ current: 'Mọi danh mục', within: 'page' }, 'Cà phê')
    await s.expectText('2 giao dịch')
    await s.click({ label: 'Xoá bộ lọc', selector: 'button', within: 'page' })

    await s.tab('Tổng quan')
    await s.expectSection('Chi nhiều nhất', ['Cà phê', '50.000 ₫', 'Bữa lớn', '45.000 ₫'])
    await s.tab('Thống kê')
    await s.click('Danh mục')
    await s.expectText('95k')
    await s.click({ text: 'Cà phê', exact: false, selector: 'button[aria-expanded]', within: 'page' })
    await s.expectText('Cà phê sáng 60% 30.000 ₫')
    await s.expectText('Cà phê (chung) 40% 20.000 ₫')
  })

  step('hạn mức tháng này: tổng chi và danh mục cha (tính cả con) → cảnh báo trên Tổng quan', async () => {
    await s.settings('Hạn mức')
    await s.expectText(`Hạn mức ${monthLabel(cur).toLowerCase()}`)
    await setBudget('Tổng chi', '100k')
    await setBudget('Cà phê', '40k')
    await waitFor(async () => (await budgets(monthKey(cur))).length === 2, { message: '2 hạn mức tháng này' })
    const list = await budgets(monthKey(cur))
    assert.deepEqual(list.map((b) => [b.category, b.limit_amount]), [[null, 100_000], ['Cà phê', 40_000]])
    await s.tab('Tổng quan')
    await s.expectSection('Hạn mức sắp vượt', ['Tổng chi', '95k / 100k (95%)', 'Cà phê', '50k / 40k (125%)'])
  })

  step('hạn mức kế thừa: đặt ở tháng trước thì tháng này dùng lại, xoá trống để tắt từ tháng này', async () => {
    await s.click({ label: 'Tháng trước', selector: 'button', within: 'page' })
    await s.expectText(monthLabel(prev), { within: 'header' })
    await s.settings('Hạn mức')
    await s.expectText(`Hạn mức ${monthLabel(prev).toLowerCase()}`)
    await setBudget('Đi lại', '300k')
    await setBudget('Tổng chi', '200k')
    await waitFor(async () => (await budgets(monthKey(prev))).length === 2, { message: '2 hạn mức tháng trước' })

    await s.tab('Tổng quan')
    await s.click({ text: monthLabel(prev), selector: 'header button', within: 'page' })
    await s.settings('Hạn mức')
    await s.expectText(`Hạn mức ${monthLabel(cur).toLowerCase()}`)
    await waitFor(async () => (await budgetRow('Đi lại'))?.value === '300000', { message: 'Đi lại kế thừa 300000' })
    assert.ok((await budgetRow('Đi lại')).text.includes(`Từ ${monthLabel(prev).toLowerCase()}`))
    assert.equal((await budgetRow('Tổng chi')).value, '100000', 'tháng này có bản ghi riêng nên không kế thừa')

    await setBudget('Đi lại', '')
    const off = await waitFor(async () => (await budgets(monthKey(cur))).find((b) => b.category === 'Đi lại'), { message: 'bản ghi tắt tháng này' })
    assert.equal(off.limit_amount, 0)
    await waitFor(async () => (await budgetRow('Đi lại'))?.text.includes('Đã tắt từ tháng này'), { message: 'nhãn Đã tắt' })

    await setBudget('Tổng chi', '200000')
    await waitFor(async () => !(await budgets(monthKey(cur))).some((b) => b.category === null), { message: 'bỏ bản ghi riêng tháng này khi trùng giá trị tháng trước' })
    await waitFor(async () => (await budgetRow('Tổng chi'))?.text.includes(`Từ ${monthLabel(prev).toLowerCase()}`), { message: 'Tổng chi kế thừa' })
    await s.tab('Tổng quan')
    await s.expectSection('Hạn mức sắp vượt', ['Cà phê', '50k / 40k (125%)'])
    assert.ok(!(await s.section('Hạn mức sắp vượt')).includes('Tổng chi'), '95k / 200k dưới 80% nên không cảnh báo')
  })

  step('xoá danh mục cha: cha thành tombstone, con thành danh mục gốc', async () => {
    await openCategory('Cà phê')
    await s.click({ label: 'Xoá danh mục', selector: 'button' })
    await waitFor(async () => (await row`select deleted_at from thuchi_categories where id = ${parent.id}`).deleted_at, { message: 'cha tombstone' })
    const c = await waitFor(async () => {
      const r = await row`select * from thuchi_categories where id = ${child.id}`
      return r.parent_id === null && r
    }, { message: 'con bị gỡ cha' })
    assert.ok(c.field_ts.parent_id)
    await s.waitNoSheet()
    await s.expectNoText('Cà phê', { within: 'main section' })
    const indented = await s.page.evaluate(() => [...document.querySelectorAll('main section button')].find((b) => window.__e2e.norm(b.innerText) === 'Cà phê sáng')?.className.includes('pl-10'))
    assert.equal(indented, false)
  })
})
