import fs from 'node:fs'
import path from 'node:path'
import { pathToFileURL } from 'node:url'

const LEVEL_MARK = { error: '✗', warn: '!', info: '·' }

export function esc(value) {
  return String(value).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c])
}

function count(result, level) {
  return result.issues.filter((i) => i.level === level).length
}

function comboTitle(r) {
  return `${r.target} · ${r.device} ${r.viewport} · ${r.theme}`
}

export function itemLine(item) {
  const comps = item.components?.length ? ` <${item.components.join(' < ')}>` : ''
  const extra = item.ratio ? ` ratio=${item.ratio}/${item.need} ${item.fg} on ${item.bg}` : item.fontSize ? ` font=${item.fontSize}` : ''
  const text = item.text ? ` "${item.text.slice(0, 50)}"` : ''
  return `${item.mark ? `[#${item.mark}] ` : ''}${item.selector}${text}${comps} @${item.rect.x},${item.rect.y} ${item.rect.w}x${item.rect.h}${extra}`
}

function issuesHtml(r) {
  if (!r.issues.length) return '<p class="clean">Không phát hiện lỗi tự động. Vẫn cần nhìn ảnh để duyệt.</p>'
  return `<ul class="issues">${r.issues.map((issue) => {
    const items = issue.items.filter((item) => item.role !== 'cover').slice(0, 6).map((item) => `<li>${esc(itemLine(item))}</li>`)
    const samples = (issue.samples || []).map((s) => `<li>${esc(s)}</li>`)
    const sub = [...items, ...samples].join('')
    return `<li class="${issue.level}"><b>${LEVEL_MARK[issue.level]} ${esc(issue.code)}</b> <span class="at">${esc(issue.at || '')}</span> ${esc(issue.message)}${sub ? `<ul>${sub}</ul>` : ''}</li>`
  }).join('')}</ul>`
}

function figure(file, label, cls) {
  return `<figure class="${cls}"><img src="${esc(path.basename(file))}"><figcaption>${esc(label)}</figcaption></figure>`
}

function shotLabel(file) {
  const name = path.basename(file, '.png')
  const part = name.split('.').slice(3).join('.')
  if (part.endsWith('annotated')) return `${part.replace('.annotated', '')} · đánh dấu lỗi`
  if (part.startsWith('focus')) return `${part} · cận cảnh`
  return part === 'p1' ? 'p1 · đầu trang' : part
}

function comboSection(r, index) {
  const cls = r.mobile ? 'mobile' : 'desktop'
  const lead = r.annotated[0] || r.shots[0]
  const rest = [...r.annotated.slice(1), ...r.shots.filter((f) => f !== lead), ...(r.focus || [])]
  const head = `<h2><span class="idx">${index}</span>${esc(comboTitle(r))} <span class="${count(r, 'error') ? 'bad' : 'ok'}">${count(r, 'error')} lỗi · ${count(r, 'warn')} cảnh báo</span></h2>
<p class="meta">Trang: ${esc(r.finalPath || '?')}${r.scroller ? ` · vùng cuộn ${esc(r.scroller.tag)} cao ${r.scroller.scrollHeight}px` : ''}${(r.notes || []).map((n) => ` · ${esc(n)}`).join('')}</p>`
  if (r.mobile) {
    return `<section class="combo">${head}<div class="lead">${lead ? figure(lead, shotLabel(lead), cls) : ''}<div class="side">${issuesHtml(r)}</div></div>
${rest.length ? `<div class="grid">${rest.map((f) => figure(f, shotLabel(f), cls)).join('')}</div>` : ''}</section>`
  }
  return `<section class="combo">${head}${issuesHtml(r)}${[lead, ...rest].filter(Boolean).map((f) => figure(f, shotLabel(f), cls)).join('')}</section>`
}

export function reportHtml(report) {
  const rows = report.results.map((r, i) => `<tr><td>${i + 1}</td><td>${esc(comboTitle(r))}</td><td class="${count(r, 'error') ? 'bad' : ''}">${count(r, 'error')}</td><td>${count(r, 'warn')}</td><td>${esc([...new Set(r.issues.filter((x) => x.level !== 'info').map((x) => x.code))].join(', ') || '—')}</td></tr>`).join('')
  return `<!doctype html><html lang="vi"><head><meta charset="utf-8"><title>ui-shot report</title><style>
@page{size:A4 landscape;margin:8mm}
*{box-sizing:border-box}
body{margin:0;color:#111827;font:11px/1.45 ui-sans-serif,system-ui,sans-serif}
h1{font-size:20px;margin:0 0 4px}h2{font-size:15px;margin:0 0 2px;display:flex;gap:8px;align-items:baseline}
.idx{display:inline-block;min-width:22px;padding:0 6px;border-radius:4px;background:#111827;color:#fff;font-size:12px;text-align:center}
.meta{color:#6b7280;margin:0 0 8px}
table{border-collapse:collapse;width:100%;margin-top:12px}td,th{border-bottom:1px solid #e5e7eb;padding:5px 8px;text-align:left;vertical-align:top}th{background:#f3f4f6}
.combo{break-before:page}
.lead{display:flex;gap:16px;align-items:flex-start}.side{flex:1;min-width:0}
.grid{display:flex;flex-wrap:wrap;gap:14px;margin-top:12px}
figure{margin:0;break-inside:avoid}figure.mobile img{height:640px;width:auto}
figure.desktop{margin-top:10px}figure.desktop img{width:100%;max-height:665px;object-fit:contain;object-position:left top}
img{display:block;border:1px solid #cbd5e1}figcaption{color:#6b7280;font-size:10px;margin-top:2px}
ul.issues{margin:0;padding-left:16px}ul.issues>li{margin-bottom:6px;break-inside:avoid}ul.issues ul{margin:2px 0 0;padding-left:14px;font:9.5px/1.4 ui-monospace,monospace;color:#374151;word-break:break-all}
li.error>b,.bad{color:#be123c}li.warn>b{color:#b45309}li.info>b{color:#6b7280}.ok{color:#15803d}.at{color:#9ca3af}.clean{color:#15803d}
.note{margin-top:14px;padding:8px 10px;background:#fef3c7;border-radius:4px}
</style></head><body>
<h1>ui-shot · báo cáo giao diện</h1>
<p class="meta">${esc(report.base)} · ${esc(new Date(report.startedAt).toLocaleString('vi-VN'))} · màn hình: ${esc(report.targets.join(', '))} · thiết bị: ${esc(report.devices.join(', '))} · theme: ${esc(report.themes.join(', '))}${report.steps.length ? ` · thao tác: ${esc(report.steps.join(' → '))}` : ''}</p>
<p><b class="${report.totals.error ? 'bad' : 'ok'}">${report.totals.error} lỗi</b> · ${report.totals.warn} cảnh báo · ${report.totals.info} ghi chú</p>
<table><tr><th>#</th><th>Màn hình · thiết bị · theme</th><th>Lỗi</th><th>Cảnh báo</th><th>Loại</th></tr>${rows}</table>
<p class="note">Kiểm tra tự động chỉ khoanh vùng nghi vấn. Mỗi phần sau là một thiết bị × theme: ảnh có khung đỏ (lỗi) / vàng (cảnh báo) đánh số [#n] khớp danh sách bên cạnh, tiếp theo là ảnh từng màn hình khi cuộn. Hãy nhìn ảnh trước khi kết luận giao diện ổn.</p>
${report.results.map((r, i) => comboSection(r, i + 1)).join('\n')}
</body></html>`
}

export async function writePdf(browser, report, outDir) {
  const html = path.join(outDir, 'report.html')
  fs.writeFileSync(html, reportHtml(report))
  const page = await browser.newPage()
  try {
    await page.goto(pathToFileURL(html).href, { waitUntil: 'load' })
    const file = path.join(outDir, 'report.pdf')
    await page.pdf({ path: file, format: 'A4', landscape: true, printBackground: true, preferCSSPageSize: true })
    return file
  } finally {
    await page.close().catch(() => {})
    fs.rmSync(html, { force: true })
  }
}

export function printSummary(report) {
  const lines = [`UI-SHOT ${report.totals.error ? 'FAIL' : 'PASS'}  lỗi=${report.totals.error} cảnh báo=${report.totals.warn} ghi chú=${report.totals.info}`]
  if (report.upload?.pdfUrl) lines.push(`báo cáo PDF: ${report.upload.pdfUrl}`)
  if (report.upload?.dirUrl) lines.push(`toàn bộ ảnh: ${report.upload.dirUrl}`)
  if (report.upload?.jsonUrl) lines.push(`report.json: ${report.upload.jsonUrl}`)
  if (report.upload?.note) lines.push(`ghi chú upload: ${report.upload.note}`)
  if (report.upload?.error) lines.push(`upload lỗi: ${report.upload.error}`)
  if (report.kept) lines.push(`trên máy: ${report.out} (report.pdf, report.json, *.png)`)
  else lines.push('đã xoá file tạm trên máy sau khi upload')
  report.results.forEach((r, index) => {
    lines.push('')
    lines.push(`[${index + 1}] ${comboTitle(r)} → ${r.finalPath || '?'}  lỗi=${count(r, 'error')} cảnh báo=${count(r, 'warn')}`)
    if (report.kept) {
      lines.push(`  ảnh: ${r.shots.map((f) => path.basename(f)).join(', ')}`)
      if (r.annotated.length) lines.push(`  ảnh đánh dấu lỗi: ${r.annotated.map((f) => path.basename(f)).join(', ')}`)
      if (r.focus?.length) lines.push(`  ảnh cận cảnh: ${r.focus.map((f) => path.basename(f)).join(', ')}`)
    }
    for (const issue of r.issues) {
      lines.push(`  ${LEVEL_MARK[issue.level]} ${issue.code} (${issue.at || '-'}) ${issue.message}`)
      for (const item of issue.items.filter((x) => x.role !== 'cover').slice(0, 4)) lines.push(`      ${itemLine(item)}`)
      for (const sample of issue.samples || []) lines.push(`      ${sample}`)
    }
  })
  lines.push('')
  lines.push('visualReview: PENDING — số liệu tự động không thay được việc NHÌN ảnh. Mở report.pdf (mỗi thiết bị × theme một phần) trước khi kết luận giao diện ổn.')
  console.log(lines.join('\n'))
}
