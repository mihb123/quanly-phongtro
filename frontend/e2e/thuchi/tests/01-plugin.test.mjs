import assert from 'node:assert/strict'
import { after, before, describe } from 'node:test'
import { closeAll, PLUGIN_NAME, Session, step, uniqueEmail } from '../lib/app.mjs'
import { PASSWORD } from '../lib/config.mjs'
import { row, rows, value, waitFor } from '../lib/db.mjs'
import { ingestUrl } from '../lib/ingest.mjs'
import { monthLabel, vnDay } from '../lib/time.mjs'

describe('Tiện ích Thu chi trong qlpt: bật/tắt, seed, giao diện nhúng, đăng xuất', () => {
  let s
  const email = uniqueEmail('plugin')

  before(async () => {
    s = await Session.create({ email, password: PASSWORD })
  })
  after(closeAll)

  const pull = () => s.api('GET', '/plugins/thu-chi/sync/pull?since=0')
  const ingest = () => fetch(ingestUrl(s.owner, 'notify'), { method: 'POST', headers: { 'X-Forwarded-For': s.ip }, body: '{}' })

  step('tiện ích tắt mặc định: API trả 403, URL ingest công khai trả 404, chưa có dữ liệu', async () => {
    await s.register({ enable: false })
    await s.find({ label: `Bật ${PLUGIN_NAME}`, within: 'page' })
    assert.equal(await s.page.evaluate(() => !!document.querySelector('[data-thuchi]')), false)
    const res = await pull()
    assert.equal(res.status, 403, JSON.stringify(res.body))
    assert.equal(res.body.message, 'plugin đang tắt')
    assert.equal((await ingest()).status, 404)
    assert.equal(await value`select count(*)::int from thuchi_sync_state where owner_id = ${s.owner}`, 0)
    assert.equal(await value`select count(*)::int from thuchi_raw_events where owner_id = ${s.owner}`, 0)
  })

  step('bật trong tab Tiện ích → mở app Thu chi, lần đồng bộ đầu seed 20 danh mục + Tiền mặt', async () => {
    await s.enablePlugin()
    await s.waitReady()
    assert.equal(await value`select enabled from plugin_settings where manager_id = ${s.owner} and plugin_id = 'thu-chi'`, true)
    const state = await row`select current_seq, seeded_at from thuchi_sync_state where owner_id = ${s.owner}`
    assert.ok(state?.seeded_at, 'thuchi_sync_state.seeded_at đã đặt')
    const cats = await rows`select kind, count(*)::int as n from thuchi_categories where owner_id = ${s.owner} and deleted_at is null group by kind order by kind`
    assert.deepEqual(cats, [{ kind: 'expense', n: 16 }, { kind: 'income', n: 4 }])
    const accounts = await rows`select name, kind, connector, balance from thuchi_accounts where owner_id = ${s.owner} and deleted_at is null`
    assert.deepEqual(accounts, [{ name: 'Tiền mặt', kind: 'cash', connector: 'manual', balance: 0 }])
    await s.expectText(`Chi ${monthLabel(vnDay()).toLowerCase()}`)
    await s.expectText(monthLabel(vnDay()), { within: 'header' })
    await s.expectSection('Tài khoản', ['Tiền mặt', 'Nhập tay', '0 ₫'])
    assert.equal((await s.localRows('categories')).length, 20, 'Dexie có đủ 20 danh mục')
    const meta = await s.localRows('meta')
    assert.equal(meta.find((m) => m.key === 'cursor')?.value, Number(state.current_seq), 'cursor local = current_seq server')
    const res = await ingest()
    assert.equal(res.status, 401, 'đã bật: ingest tới được plugin và đòi token thiết bị')
  })

  step('giao diện nhúng: tab Thu chi không cố định đè thanh điều hướng của qlpt, nút thêm nằm trên thanh đó', async () => {
    const layout = await s.page.evaluate(() => {
      const app = document.querySelector('[data-thuchi]')
      const nav = app.querySelector('nav')
      const fab = app.querySelector('button[aria-label="Thêm giao dịch"]').getBoundingClientRect()
      const qlptNav = document.querySelector('nav[aria-label="Điều hướng chính"]').getBoundingClientRect()
      return { navPosition: getComputedStyle(nav).position, fabBottom: fab.bottom, qlptNavTop: qlptNav.top, width: document.documentElement.scrollWidth, viewport: innerWidth }
    })
    assert.notEqual(layout.navPosition, 'fixed')
    assert.ok(layout.fabBottom <= layout.qlptNavTop, `nút thêm (${layout.fabBottom}) phải nằm trên thanh điều hướng qlpt (${layout.qlptNavTop})`)
    assert.ok(layout.width <= layout.viewport, 'không tràn ngang')
    await s.tab('Giao dịch')
    await s.expectText('Không có giao dịch')
    await s.tab('Tổng quan')
  })

  step('tắt tiện ích: API 403 + ingest 404, bật lại thì dữ liệu vẫn còn', async () => {
    await s.click({ label: `Tắt ${PLUGIN_NAME}`, within: 'page' })
    await s.toast(`Đã tắt ${PLUGIN_NAME}`)
    await s.expectText('Tiện ích đang tắt')
    assert.equal(await value`select enabled from plugin_settings where manager_id = ${s.owner} and plugin_id = 'thu-chi'`, false)
    assert.equal((await pull()).status, 403)
    assert.equal((await ingest()).status, 404)
    await s.click('Bật tiện ích')
    await s.page.waitForSelector('[data-thuchi]')
    await s.waitReady()
    await s.expectSection('Tài khoản', 'Tiền mặt')
    assert.equal((await pull()).status, 200)
    assert.equal(await value`select count(*)::int from thuchi_categories where owner_id = ${s.owner}`, 20, 'bật lại không seed trùng')
  })

  step('tải lại trang vẫn mở đúng tiện ích và dữ liệu trên máy', async () => {
    await s.page.reload({ waitUntil: 'domcontentloaded' })
    await s.page.waitForSelector('[data-thuchi]', { timeout: 20000 })
    await s.waitReady()
    await s.expectSection('Tài khoản', 'Tiền mặt')
  })

  step('đăng xuất khỏi qlpt xoá IndexedDB thu chi', async () => {
    assert.ok((await s.databases()).includes(`thuchi_${s.owner}`))
    await s.page.evaluate(() => [...document.querySelectorAll('header button')].find((b) => b.textContent.includes('Tài khoản'))?.click())
    await s.page.waitForFunction(() => [...document.querySelectorAll('[role="dialog"] button')].some((b) => b.textContent.includes('Đăng xuất')), { polling: 100 })
    await s.page.evaluate(() => [...document.querySelectorAll('[role="dialog"] button')].find((b) => b.textContent.includes('Đăng xuất')).click())
    await s.waitUrl('/login')
    await waitFor(async () => !(await s.databases()).includes(`thuchi_${s.owner}`), { message: 'xoá IndexedDB của user' })
  })

  step('đăng nhập lại: tải lại dữ liệu từ server, không seed trùng', async () => {
    await s.login()
    await s.expectSection('Tài khoản', 'Tiền mặt')
    assert.equal((await s.localRows('categories')).length, 20)
    assert.equal(await value`select count(*)::int from thuchi_categories where owner_id = ${s.owner}`, 20)
  })
})
