import { createServer } from 'node:http'

export const FAKE_BOT_TOKEN = '100200:e2e-fake-token'
export const FAKE_BOT_USERNAME = 'thuchi_e2e_bot'

export function startFakeTelegram(port) {
  const queue = []
  const sent = []
  const calls = []
  const waiters = []
  let updateId = 0
  let messageId = 5000

  const flush = () => {
    while (waiters.length && queue.length) waiters.shift()(queue.splice(0))
  }

  const json = (res, status, body) => {
    res.writeHead(status, { 'Content-Type': 'application/json' })
    res.end(JSON.stringify(body))
  }

  const bot = (method, params) => {
    calls.push({ method, params })
    switch (method) {
      case 'getMe':
        return { id: 100200, is_bot: true, first_name: 'Thu chi E2E', username: FAKE_BOT_USERNAME, can_read_all_group_messages: true }
      case 'sendMessage': {
        const msg = { message_id: ++messageId, date: Math.floor(Date.now() / 1000), chat: { id: params.chat_id, type: params.chat_id < 0 ? 'group' : 'private' }, text: params.text, reply_markup: params.reply_markup }
        sent.push(msg)
        return msg
      }
      case 'editMessageText': {
        const msg = { message_id: params.message_id, chat: { id: params.chat_id }, text: params.text, edited: true, reply_markup: params.reply_markup }
        sent.push(msg)
        return msg
      }
      default:
        return true
    }
  }

  const server = createServer((req, res) => {
    let raw = ''
    req.on('data', (c) => (raw += c))
    req.on('end', () => {
      const url = new URL(req.url, 'http://x')
      if (url.pathname === '/_e2e/update' && req.method === 'POST') {
        const u = JSON.parse(raw)
        u.update_id = ++updateId
        if (u.message) u.message.message_id ??= ++messageId
        queue.push(u)
        flush()
        return json(res, 200, { ok: true, update_id: u.update_id, message_id: u.message?.message_id })
      }
      if (url.pathname === '/_e2e/sent') {
        const chat = url.searchParams.get('chat')
        return json(res, 200, sent.filter((m) => !chat || String(m.chat.id) === chat))
      }
      if (url.pathname === '/_e2e/calls') return json(res, 200, calls)
      const m = url.pathname.match(/^\/bot([^/]+)\/(\w+)$/)
      if (!m || m[1] !== FAKE_BOT_TOKEN) return json(res, 401, { ok: false, error_code: 401, description: 'Unauthorized' })
      const params = raw ? JSON.parse(raw) : {}
      if (m[2] === 'getUpdates') {
        if (queue.length) return json(res, 200, { ok: true, result: queue.splice(0) })
        const timer = setTimeout(() => {
          const i = waiters.indexOf(done)
          if (i >= 0) waiters.splice(i, 1)
          json(res, 200, { ok: true, result: [] })
        }, 1000)
        const done = (updates) => {
          clearTimeout(timer)
          json(res, 200, { ok: true, result: updates })
        }
        waiters.push(done)
        return
      }
      return json(res, 200, { ok: true, result: bot(m[2], params) })
    })
  })
  return new Promise((resolve) => server.listen(port, '127.0.0.1', () => resolve(server)))
}

export function fakeTelegramClient(base) {
  let seq = 0
  return {
    async message({ from, chat, text, replyTo }) {
      seq += 1
      const res = await fetch(`${base}/_e2e/update`, {
        method: 'POST',
        body: JSON.stringify({
          message: {
            from: { id: from.id, is_bot: false, first_name: from.name, username: from.username },
            chat: chat ?? { id: from.id, type: 'private' },
            date: Math.floor(Date.now() / 1000),
            text,
            ...(replyTo ? { reply_to_message: { message_id: replyTo, from: { id: 100200, is_bot: true, first_name: 'Thu chi E2E', username: FAKE_BOT_USERNAME }, chat: chat ?? { id: from.id, type: 'private' } } } : {}),
          },
        }),
      })
      return (await res.json()).message_id
    },
    async sent(chatId) {
      return (await fetch(`${base}/_e2e/sent?chat=${chatId}`)).json()
    },
  }
}
