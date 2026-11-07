import assert from 'node:assert/strict'
import { randomUUID } from 'node:crypto'
import { after, before, describe } from 'node:test'
import { closeAll, newSession, sleep, step } from '../lib/app.mjs'
import { account, execSql, row, tx, value, waitFor } from '../lib/db.mjs'

describe('Local-first: offline, outbox, nhiều máy, xung đột theo trường, reset', () => {
  let s
  let d2
  let water
  const title = () => s.syncTitle()

  before(async () => {
    s = await newSession({ prefix: 'sync' })
  })
  after(closeAll)

  async function offline(session, on) {
    await session.page.setOfflineMode(on)
    if (on) await session.page.waitForFunction(() => document.querySelector('[data-thuchi] header button[title]')?.title.startsWith('Ngoại tuyến'), { polling: 100 })
  }

  async function setNote(session, merchant, note) {
    await session.tab('Giao dịch')
    await session.openTx(merchant)
    await session.fill('#tx-note', note)
    await session.blur()
    await sleep(200)
    await session.closeSheet()
  }

  step('mất mạng vẫn thêm được giao dịch: lưu trên máy, nằm trong outbox, chưa lên server', async () => {
    await offline(s, true)
    await s.addTx({ amount: '15k', title: 'Nước suối offline' })
    await s.tab('Giao dịch')
    await s.expectText('Nước suối offline')
    await s.expectText('Chưa đồng bộ')
    await waitFor(async () => (await title()) === 'Ngoại tuyến · 1 thay đổi chờ gửi', { message: 'badge "1 thay đổi chờ gửi"' })
    const outbox = await s.localRows('outbox')
    assert.equal(outbox.length, 1)
    assert.equal(outbox[0].op, 'upsert')
    assert.equal(outbox[0].entity, 'transactions')
    assert.equal(outbox[0].changes.merchant, 'Nước suối offline')
    const local = (await s.localRows('transactions')).find((t) => t.merchant === 'Nước suối offline')
    assert.equal(local.change_seq, 0)
    await sleep(1500)
    assert.equal(await tx.byMerchant(s.owner, 'Nước suối offline'), undefined, 'server chưa có')
    await s.settings()
    await s.expectSection('Đồng bộ', 'Ngoại tuyến · 1 thay đổi chờ gửi')
  })

  step('có mạng lại thì tự đẩy outbox lên server và cập nhật trạng thái', async () => {
    await offline(s, false)
    water = await waitFor(() => tx.byMerchant(s.owner, 'Nước suối offline'), { message: 'giao dịch offline lên server' })
    assert.equal(water.amount, 15_000)
    await s.waitSynced()
    assert.equal((await s.localRows('outbox')).length, 0)
    await s.tab('Giao dịch')
    await s.expectNoText('Chưa đồng bộ')
    await waitFor(async () => (await account.byName(s.owner, 'Tiền mặt')).balance === -15_000, { message: 'số dư cập nhật' })
  })

  step('đang mở app thì mất mạng: vẫn đọc dữ liệu trên máy và ghi được, có mạng lại tự đẩy lên', async () => {
    await offline(s, true)
    await s.tab('Giao dịch')
    await s.expectText('Nước suối offline')
    await s.addTx({ amount: '20k', title: 'Bánh mì offline' })
    await s.expectText('Chưa đồng bộ')
    await offline(s, false)
    await waitFor(() => tx.byMerchant(s.owner, 'Bánh mì offline'), { message: 'ghi lúc mất mạng lên server' })
    await s.waitSynced()
    await s.expectNoText('Chưa đồng bộ')
  })

  step('máy thứ 2 đăng nhập cùng tài khoản thấy dữ liệu đã đồng bộ', async () => {
    d2 = await s.newDevice('may2')
    await d2.tab('Giao dịch')
    await d2.expectText('Nước suối offline')
    await d2.expectText('Bánh mì offline')
  })

  step('2 máy sửa 2 trường khác nhau của cùng giao dịch → giữ cả hai', async () => {
    await offline(s, true)
    await setNote(s, 'Nước suối offline', 'ghi chú từ máy 1')
    await d2.openTx('Nước suối offline')
    await d2.toggle('Không tính vào thống kê')
    await waitFor(async () => (await tx.get(water.id)).excluded, { message: 'máy 2 đặt excluded' })
    await d2.closeSheet()
    await offline(s, false)
    await waitFor(async () => (await tx.get(water.id)).note === 'ghi chú từ máy 1', { message: 'máy 1 đẩy note' })
    const t = await tx.get(water.id)
    assert.equal(t.excluded, true, 'excluded của máy 2 không bị ghi đè')
    assert.ok(t.field_ts.note && t.field_ts.excluded)
    await s.waitSynced()
    await s.sync()
    await s.expectText('Không tính')
    await d2.sync()
    await d2.openTx('Nước suối offline')
    assert.equal(await d2.inputValue('#tx-note'), 'ghi chú từ máy 1')
    await d2.closeSheet()
  })

  step('cùng một trường: bản sửa sau thắng, bản cũ đẩy lên muộn bị bỏ qua', async () => {
    await offline(s, true)
    await setNote(s, 'Nước suối offline', 'máy 1 sửa trước')
    await sleep(300)
    await setNote(d2, 'Nước suối offline', 'máy 2 sửa sau')
    await waitFor(async () => (await tx.get(water.id)).note === 'máy 2 sửa sau', { message: 'máy 2 lên server' })
    await offline(s, false)
    await s.waitSynced()
    await waitFor(async () => (await s.localRows('outbox')).length === 0, { message: 'máy 1 đã đẩy outbox' })
    assert.equal((await tx.get(water.id)).note, 'máy 2 sửa sau')
    await s.sync()
    await s.openTx('Nước suối offline')
    assert.equal(await s.inputValue('#tx-note'), 'máy 2 sửa sau', 'máy 1 nhận giá trị thắng từ server')
    await s.closeSheet()
  })

  step('thay đổi bị server từ chối: xoá bản ghi tạm trên máy và báo trong Cài đặt', async () => {
    const id = randomUUID()
    const cash = await account.byName(s.owner, 'Tiền mặt')
    const now = new Date().toISOString()
    await s.page.evaluate(
      async (name, item, local) => {
        const db = await new Promise((res, rej) => {
          const r = indexedDB.open(name)
          r.onsuccess = () => res(r.result)
          r.onerror = () => rej(r.error)
        })
        await new Promise((res, rej) => {
          const t = db.transaction(['outbox', 'transactions'], 'readwrite')
          t.objectStore('outbox').add(item)
          t.objectStore('transactions').put(local)
          t.oncomplete = res
          t.onerror = () => rej(t.error)
        })
        db.close()
      },
      `thuchi_${s.owner}`,
      { mid: randomUUID(), entity: 'transactions', op: 'upsert', id, changes: { account_id: cash.id, direction: 'out', amount: 0, occurred_at: now, merchant: 'Số tiền 0' }, ts: Date.now() },
      { id, account_id: cash.id, direction: 'out', amount: 0, occurred_at: now, merchant: 'Số tiền 0', change_seq: 0, sources: ['manual'], kind: 'normal' },
    )
    await s.sync()
    await waitFor(async () => !(await s.localRows('transactions')).some((t) => t.id === id), { message: 'bản ghi tạm bị xoá khỏi máy' })
    assert.equal(await value`select count(*)::int from thuchi_transactions where id = ${id}`, 0)
    await s.settings()
    await s.expectSection('Đồng bộ', '1 thay đổi bị máy chủ từ chối (gần nhất: thuchi: invalid input: amount)')
  })

  step('máy cũ chưa kịp thấy tombstone mà server đã dọn → nhận reset, tải lại sạch dữ liệu', async () => {
    await s.addTx({ amount: '5k', title: 'Sẽ bị dọn' })
    const gone = await waitFor(() => tx.byMerchant(s.owner, 'Sẽ bị dọn'))
    await s.waitSynced()
    await offline(s, true)
    await d2.sync()
    await d2.tab('Giao dịch')
    await d2.openTx('Sẽ bị dọn')
    await d2.click('Xoá')
    await d2.toast('Đã xoá giao dịch')
    const tomb = await waitFor(async () => {
      const r = await row`select change_seq, deleted_at from thuchi_transactions where id = ${gone.id}`
      return r.deleted_at && r
    }, { message: 'tombstone' })
    await execSql`delete from thuchi_transactions where id = ${gone.id}`
    await execSql`update thuchi_sync_state set purged_seq = greatest(purged_seq, ${Number(tomb.change_seq)}) where owner_id = ${s.owner}`
    assert.ok((await s.localRows('transactions')).some((t) => t.id === gone.id), 'máy 1 vẫn còn bản cũ')

    const reset = s.page.waitForResponse(async (r) => r.url().includes('/sync/pull') && (await r.json().catch(() => null))?.data?.reset === true, { timeout: 20000 })
    await offline(s, false)
    await reset
    await s.waitSynced()
    await waitFor(async () => !(await s.localRows('transactions')).some((t) => t.id === gone.id), { message: 'reset xoá bản đã dọn' })
    const serverLive = await value`select count(*)::int from thuchi_transactions where owner_id = ${s.owner} and deleted_at is null`
    assert.equal((await s.localRows('transactions')).length, serverLive)
    assert.equal((await s.localRows('categories')).length, 20)
    await s.tab('Giao dịch')
    await s.expectNoText('Sẽ bị dọn')
    await s.expectText('Nước suối offline')
  })
})
