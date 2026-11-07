import { createHmac, randomBytes } from 'node:crypto'
import { BASE_URL } from './config.mjs'
import { row, waitFor } from './db.mjs'
import { sepayTime } from './time.mjs'
import { buildXLS } from './xls.mjs'

const hmacHex = (secret, data) => createHmac('sha256', secret).update(data).digest('hex')
let seq = 0

export const ingestUrl = (owner, source) => `${BASE_URL}/api/v1/plugins/thu-chi/public/${owner}/ingest/${source}`
export const shownIngestUrl = (owner, source) => ingestUrl(owner, source).replace(/^http:/, 'https:')

export function sepayPayload({ account, gateway = 'TPBank', type = 'out', amount, content, at = new Date(), accumulated, ref }) {
  seq += 1
  return {
    id: Number(`${Date.now()}`.slice(-9)) * 100 + seq,
    gateway,
    transactionDate: sepayTime(at),
    accountNumber: account,
    subAccount: null,
    code: null,
    content,
    transferType: type,
    transferAmount: amount,
    accumulated: accumulated ?? 0,
    referenceCode: ref ?? `FT${Date.now().toString(36).toUpperCase()}${seq}`,
    description: `BankAPINotify ${content}`,
  }
}

async function send(url, { headers, body, ip }) {
  const res = await fetch(url, { method: 'POST', headers: { ...headers, 'X-Forwarded-For': ip }, body })
  return { status: res.status, body: await res.text() }
}

export function postSePay(s, payload, { secret = s.sepaySecret, signature, timestamp } = {}) {
  const body = typeof payload === 'string' ? payload : JSON.stringify(payload)
  const ts = timestamp ?? String(Math.floor(Date.now() / 1000))
  return send(ingestUrl(s.owner, 'sepay'), {
    ip: s.ip,
    body,
    headers: { 'Content-Type': 'application/json', 'X-SePay-Timestamp': ts, 'X-SePay-Signature': signature ?? `sha256=${hmacHex(secret, `${ts}.${body}`)}` },
  })
}

function tokenHeaders(token, body, mode, timestamp) {
  if (mode === 'bearer') return { Authorization: `Bearer ${token}` }
  const [id, secret] = token.split(/\.(.*)/s)
  const ts = timestamp ?? String(Math.floor(Date.now() / 1000))
  return { 'X-ThuChi-Token': id, 'X-ThuChi-Timestamp': ts, 'X-ThuChi-Signature': `sha256=${hmacHex(secret, `${ts}.${body}`)}` }
}

export function postNotify(s, payload, { token, mode = 'bearer', timestamp } = {}) {
  const body = JSON.stringify(payload)
  return send(ingestUrl(s.owner, 'notify'), { ip: s.ip, body, headers: { 'Content-Type': 'application/json', ...tokenHeaders(token, body, mode, timestamp) } })
}

export function postEmail(s, raw, { token, mode = 'hmac', owner = s.owner } = {}) {
  return send(ingestUrl(owner, 'email'), { ip: s.ip, body: raw, headers: { 'Content-Type': 'message/rfc822', ...tokenHeaders(token, raw, mode) } })
}

export async function waitRaw(owner, needle, { timeout = 15000 } = {}) {
  return waitFor(
    async () => {
      const r = await row`select * from thuchi_raw_events where owner_id = ${owner} and payload like ${`%${needle}%`} order by received_at desc limit 1`
      return r && r.status !== 'pending' && r
    },
    { timeout, message: `raw event chứa "${needle}" được xử lý` },
  )
}

const wrap = (b64) => b64.match(/.{1,76}/g).join('\r\n')

function mime({ from, subject, date = new Date(), parts, messageId = `${randomBytes(8).toString('hex')}@e2e.local` }) {
  const boundary = `b-${randomBytes(6).toString('hex')}`
  const head = [`From: ${from}`, 'To: me@example.com', `Subject: ${subject}`, `Date: ${date.toUTCString()}`, `Message-ID: <${messageId}>`, 'MIME-Version: 1.0']
  if (parts.length === 1 && !parts[0].filename) {
    const p = parts[0]
    return [...head, `Content-Type: ${p.type}; charset=utf-8`, 'Content-Transfer-Encoding: base64', '', wrap(Buffer.from(p.body).toString('base64')), ''].join('\r\n')
  }
  const body = parts.flatMap((p) => [
    `--${boundary}`,
    p.filename ? `Content-Type: ${p.type}; name="${p.filename}"` : `Content-Type: ${p.type}; charset=utf-8`,
    ...(p.filename ? [`Content-Disposition: attachment; filename="${p.filename}"`] : []),
    'Content-Transfer-Encoding: base64',
    '',
    wrap(Buffer.from(p.body).toString('base64')),
  ])
  return [...head, `Content-Type: multipart/mixed; boundary="${boundary}"`, '', ...body, `--${boundary}--`, ''].join('\r\n')
}

const pad = (n) => String(n).padStart(2, '0')
const vnParts = (date) => {
  const t = new Date(date.getTime() + 7 * 3600_000)
  return { d: t.getUTCDate(), m: t.getUTCMonth() + 1, y: t.getUTCFullYear(), hh: t.getUTCHours(), mm: t.getUTCMinutes(), ss: t.getUTCSeconds() }
}

export function vcbReceiptEmail({ at, orderNo, debit, amount, fee = 0, payee, payeeAccount, payeeBank, content, messageId }) {
  const p = vnParts(at)
  const cell = (label, value) => `<tr><td style="padding:4px">${label}</td><td style="padding:4px"><b>${value}</b></td></tr>`
  const html = `<html><body><p>Kính gửi Quý khách,</p><table>${[
    cell('Ngày, giờ giao dịch<br>Trans. Date, Time', `${pad(p.hh)}:${pad(p.mm)}:${pad(p.ss)} Thứ Tư ${pad(p.d)}/${pad(p.m)}/${p.y}`),
    cell('Số lệnh giao dịch<br>Order Number', orderNo),
    cell('Tài khoản nguồn<br>Debit Account', debit),
    cell('Tài khoản người hưởng<br>Credit Account', payeeAccount),
    cell('Tên người hưởng<br>Beneficiary Name', payee),
    cell('Tên ngân hàng hưởng<br>Beneficiary Bank Name', payeeBank),
    cell('Số tiền<br>Amount', `${amount.toLocaleString('en-US')} VND`),
    cell('Loại phí<br>Charge Code', 'Người chuyển trả'),
    cell('Số tiền phí<br>Charge Amount', `${fee.toLocaleString('en-US')} VND`),
    cell('Nội dung chuyển tiền<br>Details of Payment', content),
  ].join('')}</table></body></html>`
  return mime({ from: 'Vietcombank <VCBDigibank@info.vietcombank.com.vn>', subject: 'Biên lai chuyển tiền qua tài khoản', date: at, messageId, parts: [{ type: 'text/html', body: html }] })
}

export function timoStatementEmail({ account, rows }) {
  const fmt = (n) => (n ? n.toLocaleString('en-US') : '')
  const date = (at) => {
    const p = vnParts(at)
    return `${pad(p.d)}/${pad(p.m)}/${p.y} ${pad(p.hh)}:${pad(p.mm)}:${pad(p.ss)}`
  }
  const sheet = [
    ['SAO KÊ TÀI KHOẢN / ACCOUNT STATEMENT'],
    [],
    ['Số tài khoản chính:', '', account],
    ['Tên chủ tài khoản:', '', 'NGUYEN VAN E2E'],
    [],
    ['Ngày GD', 'Mã giao dịch', 'Số tài khoản truy vấn', 'Phát sinh có', 'Phát sinh nợ', 'Số dư', 'Diễn giải'],
    ...rows.map((r) => [date(r.at), r.ref, account, fmt(r.credit), fmt(r.debit), r.balance, r.desc]),
  ]
  return mime({
    from: 'Timo <noreply@timo.vn>',
    subject: 'Transaction report for Timo account',
    parts: [
      { type: 'text/plain', body: 'Quý khách vui lòng xem sao kê đính kèm.' },
      { type: 'application/vnd.ms-excel', filename: 'Timo-SaoKeDienTu.xls', body: buildXLS(sheet, 'Sao ke') },
    ],
  })
}

export function gmailForwardingEmail(link) {
  return mime({
    from: 'Gmail Team <forwarding-noreply@google.com>',
    subject: '(#123456789) Gmail Forwarding Confirmation - Receive Mail from me@gmail.com',
    parts: [{ type: 'text/plain', body: `me@gmail.com has requested to automatically forward mail to your email address.\nTo allow, click the link below:\n\n${link}\n` }],
  })
}

export function otherEmail() {
  return mime({ from: 'Shop <news@shop.example>', subject: 'Khuyến mãi tháng này', parts: [{ type: 'text/plain', body: 'Giảm giá 50%' }] })
}
