import assert from 'node:assert/strict'
import { randomUUID } from 'node:crypto'
import { after, before, describe } from 'node:test'
import { closeAll, newSession, step } from '../lib/app.mjs'
import { BASE_URL } from '../lib/config.mjs'
import { account, category, row, tx, value, waitFor } from '../lib/db.mjs'
import { ingestUrl } from '../lib/ingest.mjs'

describe('Bảo mật & kiểm tra dữ liệu phía server', () => {
  let a
  let b
  let aTx
  let aCat

  before(async () => {
    a = await newSession({ prefix: 'sec-a', label: 'userA' })
    b = await newSession({ prefix: 'sec-b', label: 'userB' })
    await a.addTx({ amount: '77k', title: 'Bí mật của A' })
    aTx = await waitFor(() => tx.byMerchant(a.owner, 'Bí mật của A'))
    aCat = await category.byKey(a.owner, 'an_uong')
  })
  after(closeAll)

  const results = (res) => {
    assert.equal(res.status, 200, JSON.stringify(res.body))
    return res.body.data.results.map((r) => (r.ok ? 'ok' : r.error))
  }

  step('API thu chi và ingest cần xác thực', async () => {
    const call = async (path, init) => {
      const res = await fetch(`${BASE_URL}/api/v1/plugins/thu-chi${path}`, init)
      return [res.status, (await res.json()).message]
    }
    assert.deepEqual(await call('/sync/pull?since=0'), [400, 'missing authentication token'])
    assert.deepEqual(await call('/sync/push', { method: 'POST', body: '{"mutations":[]}' }), [400, 'missing authentication token'])
    assert.deepEqual(await call('/ingest/info', { headers: { Authorization: 'DPoP abc' } }), [400, 'invalid token'])
    assert.deepEqual(await call('/sync/pull?since=0', { headers: { Authorization: `DPoP ${a.accessToken}` } }), [401, 'missing DPoP proof'], 'lộ access token mà không có khoá DPoP thì vô dụng')
    const ip = { 'X-Forwarded-For': a.ip }
    assert.equal((await fetch(ingestUrl(a.owner, 'notify'), { method: 'POST', headers: ip, body: '{}' })).status, 401)
    assert.equal((await fetch(ingestUrl(a.owner, 'sepay'), { method: 'POST', headers: ip, body: '{}' })).status, 401, 'chưa cấu hình SePay')
    assert.equal((await fetch(ingestUrl(a.owner, 'khac'), { method: 'POST', headers: ip, body: '{}' })).status, 404)
    assert.equal((await fetch(ingestUrl('khong-phai-uuid', 'notify'), { method: 'POST', headers: ip, body: '{}' })).status, 404)
    assert.equal(await value`select count(*)::int from thuchi_raw_events where owner_id = ${a.owner}`, 0)
  })

  step('user B không đọc được dữ liệu của A', async () => {
    const pull = await b.api('GET', '/plugins/thu-chi/sync/pull?since=0')
    assert.equal(pull.status, 200)
    const ids = Object.values(pull.body.data.changes).flat().map((r) => r.id)
    assert.ok(ids.length > 0)
    assert.ok(!ids.includes(aTx.id) && !ids.includes(aCat.id), 'pull của B không có bản ghi của A')
    await b.tab('Giao dịch')
    await b.expectText('Không có giao dịch')
    const info = await b.api('GET', '/plugins/thu-chi/ingest/info')
    assert.equal(info.body.data.owner_id, b.owner)
  })

  step('user B không sửa/xoá/ghi đè được bản ghi của A (patch, delete, upsert trùng id, action)', async () => {
    const r = results(
      await b.push([
        { entity: 'transactions', op: 'patch', id: aTx.id, changes: { note: 'B sửa' } },
        { entity: 'transactions', op: 'delete', id: aTx.id },
        { entity: 'transactions', op: 'upsert', id: aTx.id, changes: { account_id: null, direction: 'out', amount: 1, occurred_at: new Date().toISOString(), merchant: 'ghi đè' } },
        { entity: 'categories', op: 'upsert', id: aCat.id, changes: { name: 'B chiếm', kind: 'expense' } },
        { op: 'action', action: 'set_category', args: { id: aTx.id, category_id: null } },
      ]),
    )
    assert.deepEqual(r, ['thuchi: not found', 'thuchi: not found', 'thuchi: forbidden', 'thuchi: forbidden', 'thuchi: not found'])
    const t = await tx.get(aTx.id)
    assert.equal(t.owner_id, a.owner)
    assert.equal(t.note, '')
    assert.equal(t.deleted_at, null)
    assert.equal(t.merchant, 'Bí mật của A')
    assert.equal((await category.byKey(a.owner, 'an_uong')).name, 'Ăn uống')
  })

  step('validator chặn dữ liệu sai; mỗi mutation chạy trong savepoint nên mutation hợp lệ cùng lô vẫn được ghi', async () => {
    const cash = await account.byName(a.owner, 'Tiền mặt')
    const luong = await category.byKey(a.owner, 'luong', 'income')
    const parentId = randomUUID()
    const childId = randomUUID()
    const now = new Date().toISOString()
    const r = results(
      await a.push([
        { entity: 'categories', op: 'upsert', id: parentId, changes: { name: 'Cha', kind: 'expense', icon: 'tag', color: '#000000' } },
        { entity: 'categories', op: 'upsert', id: childId, changes: { name: 'Con', kind: 'expense', icon: 'tag', color: '#000000', parent_id: parentId } },
        { entity: 'categories', op: 'patch', id: parentId, changes: { parent_id: childId } },
        { entity: 'categories', op: 'patch', id: parentId, changes: { parent_id: parentId } },
        { entity: 'categories', op: 'upsert', id: randomUUID(), changes: { name: 'Con khác loại', kind: 'expense', parent_id: luong.id } },
        { entity: 'categories', op: 'upsert', id: randomUUID(), changes: { name: '', kind: 'expense' } },
        { entity: 'transactions', op: 'upsert', id: randomUUID(), changes: { account_id: cash.id, direction: 'out', amount: 0, occurred_at: now } },
        { entity: 'transactions', op: 'upsert', id: randomUUID(), changes: { account_id: cash.id, direction: 'out', amount: 1000, occurred_at: now, kind: 'borrow', party: 'X' } },
        { entity: 'transactions', op: 'upsert', id: randomUUID(), changes: { account_id: cash.id, direction: 'out', amount: 1000, occurred_at: now, kind: 'lend', party: 'x'.repeat(101) } },
        { entity: 'budgets', op: 'upsert', id: randomUUID(), changes: { month: '2026-13', limit_amount: 1000 } },
        { entity: 'budgets', op: 'upsert', id: randomUUID(), changes: { month: '2026-01', limit_amount: -1 } },
        { entity: 'accounts', op: 'upsert', id: randomUUID(), changes: { name: 'Ví lạ', kind: 'crypto' } },
        { entity: 'accounts', op: 'upsert', id: randomUUID(), changes: { name: 'Số TK ngắn', kind: 'bank', account_no: '123' } },
        { entity: 'recurrences', op: 'upsert', id: randomUUID(), changes: { account_id: cash.id, direction: 'out', amount: 1000, freq: 'monthly', interval: 0, start_at: now, active: true } },
        { entity: 'khong_co', op: 'upsert', id: randomUUID(), changes: {} },
        { op: 'action', action: 'khong_co', args: {} },
      ]),
    )
    assert.deepEqual(r, [
      'ok',
      'ok',
      'thuchi: invalid input: parent_id',
      'thuchi: invalid input: parent_id',
      'thuchi: invalid input: parent_id',
      'thuchi: invalid input: category name',
      'thuchi: invalid input: amount',
      'thuchi: invalid input: kind',
      'thuchi: invalid input: party',
      'thuchi: invalid input: budget',
      'thuchi: invalid input: budget',
      'thuchi: invalid input: account kind',
      'thuchi: invalid input: account_no',
      'thuchi: invalid input: interval',
      'thuchi: invalid input: entity "khong_co"',
      'thuchi: invalid input: action "khong_co"',
    ])
    assert.equal((await row`select parent_id from thuchi_categories where id = ${childId}`).parent_id, parentId)
    assert.equal((await row`select parent_id from thuchi_categories where id = ${parentId}`).parent_id, null)
    assert.equal(await value`select count(*)::int from thuchi_budgets where owner_id = ${a.owner}`, 0)
  })

  step('đồng hồ client chạy nhanh bị ép về giờ server khi ghi field_ts', async () => {
    const future = Date.now() + 24 * 3600_000
    const r = results(await a.push([{ entity: 'transactions', op: 'patch', id: aTx.id, changes: { note: 'giờ máy sai' }, ts: future }]))
    assert.deepEqual(r, ['ok'])
    const t = await tx.get(aTx.id)
    assert.equal(t.note, 'giờ máy sai')
    assert.ok(t.field_ts.note <= Date.now() + 60_000, `field_ts.note (${t.field_ts.note}) không vượt giờ server`)
  })

  step('route công khai của plugin: plugin lạ hoặc manager chưa bật đều 404', async () => {
    const ip = { 'X-Forwarded-For': a.ip }
    assert.equal((await fetch(`${BASE_URL}/api/v1/plugins/khong-co/public/${a.owner}/ingest/notify`, { method: 'POST', headers: ip, body: '{}' })).status, 404)
    assert.equal((await fetch(`${BASE_URL}/api/v1/plugins/bank-statement/public/${a.owner}/x`, { method: 'POST', headers: ip, body: '{}' })).status, 404, 'plugin không có route công khai')
    assert.equal((await fetch(ingestUrl(randomUUID(), 'notify'), { method: 'POST', headers: ip, body: '{}' })).status, 404, 'manager không tồn tại / chưa bật')
    const health = await fetch(`${BASE_URL}/health`)
    assert.deepEqual(await health.json(), { status: 'ok' })
  })
})
