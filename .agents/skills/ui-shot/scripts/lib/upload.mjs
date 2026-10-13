import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { UPLOAD_BASE } from './config.mjs'

const ATTEMPTS = 3

function firstUrl(text) {
  return text.split(/\s+/).find((token) => /^https?:\/\//.test(token)) || null
}

async function post(url, body, headers) {
  let lastError = null
  for (let attempt = 1; attempt <= ATTEMPTS; attempt++) {
    try {
      const res = await fetch(url, { method: 'POST', body, headers, signal: AbortSignal.timeout(120_000) })
      const text = (await res.text()).trim()
      if (!res.ok) throw new Error(`HTTP ${res.status}: ${text.slice(0, 200)}`)
      return text
    } catch (err) {
      lastError = err
      await new Promise((r) => setTimeout(r, 1500 * attempt))
    }
  }
  throw lastError
}

async function listDirectory(dirUrl) {
  for (let attempt = 1; attempt <= ATTEMPTS; attempt++) {
    try {
      const res = await fetch(dirUrl, { headers: { 'user-agent': 'curl/8 ui-shot' }, signal: AbortSignal.timeout(30_000) })
      if (res.ok) return (await res.text()).split(/\s+/).filter((t) => /^https?:\/\//.test(t))
    } catch {
      await new Promise((r) => setTimeout(r, 1500 * attempt))
    }
  }
  return []
}

async function uploadDirectory(dir, name) {
  const archive = execFileSync('tar', ['czf', '-', '-C', dir, '.'], { maxBuffer: 1024 * 1024 * 1024 })
  const text = await post(`${UPLOAD_BASE}/upload/dir?name=${encodeURIComponent(name)}`, archive, { 'content-type': 'application/gzip' })
  const dirUrl = firstUrl(text)
  if (!dirUrl || !dirUrl.includes('/d/')) throw new Error(`server không trả về link thư mục /d/: ${text.slice(0, 160)}`)
  const files = await listDirectory(dirUrl.endsWith('/') ? dirUrl : `${dirUrl}/`)
  const pdfUrl = files.find((u) => u.endsWith('/report.pdf'))
  if (!pdfUrl) throw new Error('thư mục đã upload không có report.pdf (server chưa giải nén?)')
  return { dirUrl, pdfUrl, files: files.length }
}

export async function uploadFile(file) {
  const text = await post(`${UPLOAD_BASE}/upload`, fs.readFileSync(file), { 'X-Filename': path.basename(file) })
  const url = firstUrl(text)
  if (!url) throw new Error(`server không trả về link cho ${path.basename(file)}: ${text.slice(0, 160)}`)
  return url
}

export async function uploadResults(outDir, name) {
  try {
    return { mode: 'directory', ...(await uploadDirectory(outDir, name)) }
  } catch (dirError) {
    const pdfUrl = await uploadFile(path.join(outDir, 'report.pdf'))
    const jsonUrl = await uploadFile(path.join(outDir, 'report.json'))
    return { mode: 'files', pdfUrl, jsonUrl, note: `upload thư mục không được (${dirError.message}); đã upload report.pdf (chứa toàn bộ ảnh) và report.json` }
  }
}
