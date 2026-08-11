import assert from 'node:assert/strict'
import { createHmac, randomBytes } from 'node:crypto'
import { mkdir } from 'node:fs/promises'
import { test } from 'node:test'
import { fileURLToPath } from 'node:url'
import puppeteer from 'puppeteer'

const appURL = process.env.SEPAY_APP_URL ?? 'http://127.0.0.1:5173'
const appEmail = process.env.SEPAY_APP_EMAIL
const appPassword = process.env.SEPAY_APP_PASSWORD
const sePayEmail = process.env.SEPAY_LOGIN_EMAIL
const sePayPassword = process.env.SEPAY_LOGIN_PASSWORD
const artifactDirectory = new URL('./artifacts/', import.meta.url)
const hasCredentials = Boolean(appEmail && appPassword && sePayEmail && sePayPassword)

// Pause briefly for third-party pages whose UI updates outside navigation events.
async function settle(milliseconds = 800) {
  await new Promise((resolve) => setTimeout(resolve, milliseconds))
}

// Click the first visible interactive element whose trimmed text exactly matches the label.
async function clickExactText(page, label) {
  const candidates = await page.$$('button, a, [role="button"], [role="option"]')
  for (const candidate of candidates) {
    const text = await candidate.evaluate((element) => element.textContent?.trim())
    const box = await candidate.boundingBox()
    if (text === label && box) {
      await candidate.click()
      return
    }
  }
  assert.fail(`Không tìm thấy nút hiển thị: ${label}`)
}

// Authenticate through the site's real login form without exposing credentials in output.
async function login(page, url, email, password) {
  await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 30_000 })
  const emailSelector = 'input[type="email"], input[name="email"], input[autocomplete="username"]'
  await page.waitForSelector(emailSelector, { visible: true, timeout: 20_000 })
  await page.type(emailSelector, email)
  await page.type('input[type="password"]', password)
  await Promise.all([
    page.waitForNavigation({ waitUntil: 'domcontentloaded', timeout: 30_000 }).catch(() => undefined),
    page.click('button[type="submit"], input[type="submit"]'),
  ])
  await settle()
}

// Enter SePay's isolated Test Mode for the current authenticated browser session.
async function enableSePayTestMode(page) {
  await page.waitForSelector('#testmodeToggle', { timeout: 20_000 })
  const enabled = await page.$eval('#testmodeToggle', (input) => input.checked)
  if (!enabled) {
    await Promise.all([
      page.waitForNavigation({ waitUntil: 'domcontentloaded', timeout: 30_000 }),
      page.click('#testmodeToggle'),
    ])
  }
  assert.match(page.url(), /\/testmode\//)
}

// Read the receiving account created in Test Mode without copying any unrelated account data.
async function readSandboxBankAccount(page) {
  await page.goto('https://my.sepay.vn/testmode/bankaccount', { waitUntil: 'domcontentloaded', timeout: 30_000 })
  await settle(1_500)
  const text = await page.evaluate(() => document.body.innerText)
  const accountNumber = text.match(/\b0000000001\b/)?.[0]
  const accountName = text.match(/CONG TY TNHH TEST D838/)?.[0]
  assert.ok(accountNumber, `Không đọc được số tài khoản Test Mode tại ${page.url()}`)
  assert.ok(accountName, `Không đọc được chủ tài khoản Test Mode tại ${page.url()}`)
  return { bankShortName: 'Vietcombank', accountNumber, accountName }
}

// Find a token value in the shape returned by SePay's API-key list endpoint.
function extractToken(item) {
  for (const [key, value] of Object.entries(item ?? {})) {
    if (typeof value === 'string' && /(token|api.?key)/i.test(key) && value.length > 16) {
      return value
    }
  }
  return ''
}

// Remove API keys created by earlier E2E runs while leaving unrelated Test Mode keys untouched.
async function removePreviousSandboxAPITokens(page) {
  const results = await page.evaluate(async () => {
    const e2eKeys = window._allApiItems
      .filter((item) => item.name.startsWith('quanly-phongtro-e2e'))
      .map((item) => item.id)
    const deleted = []
    for (const id of e2eKeys) {
      const result = await new Promise((resolve, reject) => {
        window.$.ajax({
          url: 'https://my.sepay.vn/testmode/companyapi/ajax_api_delete',
          type: 'POST',
          data: window.$.extend({ id }, window.SePay.csrf.data()),
          dataType: 'json',
          success: resolve,
          error: reject,
        })
      })
      deleted.push(result)
    }
    return deleted
  })
  assert.ok(results.every((result) => result.status === true), 'Không thể dọn API key từ lần E2E trước')
}

// Create one dedicated API key after clearing keys left by earlier E2E runs.
async function getOrCreateSandboxAPIToken(page) {
  const keyName = `quanly-phongtro-e2e-${Date.now()}`
  await page.goto('https://my.sepay.vn/testmode/companyapi', { waitUntil: 'domcontentloaded', timeout: 30_000 })
  const initialListResponse = page.waitForResponse(
    (response) => response.url().includes('/testmode/companyapi/ajax_api_list') && response.status() === 200,
    { timeout: 20_000 },
  )
  await page.evaluate(() => window.loadApiList())
  await initialListResponse
  await page.waitForFunction(() => Array.isArray(window._allApiItems), { timeout: 20_000 })
  await removePreviousSandboxAPITokens(page)

  await page.evaluate(() => window.show_add_api())
  await page.waitForSelector('#api_name_input', { visible: true, timeout: 10_000 })
  await page.type('#api_name_input', keyName)
  const listResponse = page.waitForResponse(
    (response) => response.url().includes('/testmode/companyapi/ajax_api_list') && response.status() === 200,
    { timeout: 20_000 },
  )
  await page.evaluate(() => window.save())
  await listResponse
  await page.waitForFunction(
    (name) => window._allApiItems.some((candidate) => candidate.name === name),
    { timeout: 20_000 },
    keyName,
  )

  const items = await page.evaluate(() => window._allApiItems)
  const item = items.find((candidate) => candidate.name === keyName)
  let token = extractToken(item)
  if (!token) {
    token = await page.evaluate((name) => {
      const row = [...document.querySelectorAll('tr, .card, [data-api-id]')]
        .find((element) => element.textContent?.includes(name))
      return row?.querySelector('[data-clipboard-text]')?.getAttribute('data-clipboard-text') ?? ''
    }, keyName)
  }
  assert.ok(token, 'SePay đã tạo API key nhưng không trả token có thể sao chép')
  return token
}

// Confirm the Test Mode token is accepted by the official sandbox API.
async function verifySandboxAPI(apiToken) {
  const response = await fetch('https://userapi-sandbox.sepay.vn/v2/transactions?per_page=1', {
    headers: { Authorization: `Bearer ${apiToken}`, Accept: 'application/json' },
  })
  assert.equal(response.status, 200, `Sandbox API trả HTTP ${response.status}`)
  const payload = await response.json()
  assert.ok(Array.isArray(payload.data), 'Sandbox API không trả danh sách giao dịch')
  return payload.data.length
}

// Choose a Base UI select option by opening its trigger and clicking the exact label.
async function chooseSelectOption(page, triggerSelector, label, selectedValue = label) {
  const triggerHTML = await page.$eval(triggerSelector, (element) => element.outerHTML)
  const currentLabel = await page.$eval(triggerSelector, (element) => element.textContent?.trim() ?? '')
  if (currentLabel === label || currentLabel === selectedValue || label.startsWith(`${currentLabel} (`)) {
    return
  }
  await page.click(triggerSelector)
  try {
    await page.waitForSelector('[data-slot="select-item"]', { visible: true, timeout: 10_000 })
  } catch (error) {
    throw new Error(`Không mở được select ${triggerSelector}: ${triggerHTML}`, { cause: error })
  }
  await clickExactText(page, label)
  await page.waitForFunction(
    (selector) => document.querySelector(selector)?.getAttribute('aria-expanded') === 'false',
    { timeout: 5_000 },
    triggerSelector,
  )
  await settle(300)
}

// Verify the settings UI exposes exactly the supported VietQR bank snapshot and excludes a known unsupported bank.
async function verifySupportedBankOptions(page) {
  await page.click('#sepay-bank-short-name')
  await page.waitForSelector('[data-slot="select-item"]', { visible: true, timeout: 10_000 })
  const labels = await page.$$eval(
    '[data-slot="select-item"]',
    (items) => items.map((item) => item.textContent?.trim() ?? ''),
  )
  assert.equal(labels.length, 23, 'Select ngân hàng phải chỉ có 23 mục SePay supported=true')
  assert.ok(labels.includes('Vietcombank (VCB)'), 'Select phải có Vietcombank')
  assert.ok(!labels.some((label) => label.startsWith('NamABank')), 'Select không được có ngân hàng supported=false')
  await page.keyboard.press('Escape')
  await settle(300)
}

// Fill a text field after clearing any previous value.
async function fillInput(page, selector, value) {
  await page.focus(selector)
  await page.keyboard.down('Control')
  await page.keyboard.press('KeyA')
  await page.keyboard.up('Control')
  await page.keyboard.press('Backspace')
  await page.type(selector, value)
}

// Configure the local app through its real mobile-responsive settings UI.
async function configureApplication(page, bankAccount, apiToken, webhookSecret) {
  await login(page, `${appURL}/login`, appEmail, appPassword)
  await clickExactText(page, 'Cài đặt')
  await page.waitForFunction(() => document.body.innerText.includes('Tích hợp SePay'), { timeout: 20_000 })
  await settle(1_500)

  for (let attempt = 0; attempt < 3 && await page.$('#sepay-environment') === null; attempt += 1) {
    await page.click('[data-testid="sepay-edit-button"]')
    await page.waitForSelector('#sepay-environment', { visible: true, timeout: 3_000 }).catch(() => undefined)
  }
  await page.waitForSelector('#sepay-environment', { visible: true, timeout: 10_000 })
  await settle(600)
  await verifySupportedBankOptions(page)
  await chooseSelectOption(page, '#sepay-bank-short-name', `${bankAccount.bankShortName} (VCB)`, bankAccount.bankShortName)
  await chooseSelectOption(page, '#sepay-environment', 'Test Mode (sandbox)', 'sandbox')
  await fillInput(page, '#sepay-account-number', bankAccount.accountNumber)
  await fillInput(page, '#sepay-account-name', bankAccount.accountName)
  await fillInput(page, '#sepay-code-prefix', 'PH')
  if (await page.$('#sepay-webhook-secret') === null) {
    await chooseSelectOption(page, '#sepay-webhook-auth-method', 'HMAC-SHA256 (khuyến nghị)', 'hmac')
  }
  await fillInput(page, '#sepay-webhook-secret', webhookSecret)
  await fillInput(page, '#sepay-api-token', apiToken)
  const saveResponse = page.waitForResponse(
    (response) => response.url().includes('/payments/providers/sepay/config')
      && response.request().method() === 'POST',
    { timeout: 20_000 },
  )
  await clickExactText(page, 'Lưu cấu hình')
  const savedResponse = await saveResponse
  assert.equal(savedResponse.status(), 200, 'API lưu cấu hình SePay phải thành công')
  const savedPayload = await savedResponse.json()
  assert.equal(savedPayload.bank_account_verified, true, 'API phải xác thực tài khoản đã liên kết qua SePay')
  assert.equal(savedPayload.account_holder_name, bankAccount.accountName, 'API phải trả tên chủ tài khoản chuẩn từ SePay')
  await page.waitForFunction(() => document.body.innerText.includes('Đã cấu hình SePay'), { timeout: 20_000 })

  const environmentText = await page.evaluate(() => document.body.innerText)
  assert.match(environmentText, /Test Mode · Vietcombank/)
  const configuredWebhookURL = await page.$eval('#sepay-webhook-url', (input) => input.value)
  return new URL(new URL(configuredWebhookURL).pathname, appURL).toString()
}

// Compute the HMAC header exactly as SePay signs timestamp plus raw request body.
function signWebhook(secret, timestamp, body) {
  return `sha256=${createHmac('sha256', secret).update(`${timestamp}.${body}`).digest('hex')}`
}

// POST one signed webhook payload to the configured application endpoint.
async function postSignedWebhook(webhookURL, secret, payload, timestamp = Math.floor(Date.now() / 1000)) {
  const body = JSON.stringify(payload)
  return fetch(webhookURL, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-SePay-Timestamp': String(timestamp),
      'X-SePay-Signature': signWebhook(secret, String(timestamp), body),
    },
    body,
  })
}

// Exercise account binding, replay protection, valid webhook processing, and reconciliation.
async function verifyApplicationFlow(page, webhookURL, bankAccount, webhookSecret) {
  const basePayload = {
    id: Date.now(),
    gateway: bankAccount.bankShortName,
    transactionDate: '2026-08-04 11:00:00',
    accountNumber: bankAccount.accountNumber,
    code: `PHUNMATCHED${Date.now()}`,
    content: 'Puppeteer E2E sandbox verification',
    transferType: 'in',
    transferAmount: 1000,
    referenceCode: `E2E-${Date.now()}`,
  }

  const wrongAccountResponse = await postSignedWebhook(
    webhookURL,
    webhookSecret,
    { ...basePayload, id: basePayload.id + 1, accountNumber: '9999999999' },
  )
  assert.equal(wrongAccountResponse.status, 400, 'Webhook sai tài khoản phải bị từ chối')

  const expiredResponse = await postSignedWebhook(
    webhookURL,
    webhookSecret,
    { ...basePayload, id: basePayload.id + 2 },
    Math.floor(Date.now() / 1000) - 360,
  )
  assert.equal(expiredResponse.status, 400, 'Webhook quá hạn phải bị từ chối')

  const validResponse = await postSignedWebhook(webhookURL, webhookSecret, basePayload)
  assert.equal(validResponse.status, 200, 'Webhook HMAC hợp lệ phải được chấp nhận')

  await clickExactText(page, 'Đối soát giao dịch')
  await page.waitForFunction(() => document.body.innerText.includes('Đối soát xong'), { timeout: 30_000 })
}

// Save desktop and mobile evidence without revealing secrets from edit fields.
async function captureResponsiveEvidence(page) {
  await mkdir(artifactDirectory, { recursive: true })
  await page.setViewport({ width: 1440, height: 1000 })
  await page.screenshot({ path: fileURLToPath(new URL('sepay-settings-desktop.png', artifactDirectory)), fullPage: true })
  await page.setViewport({ width: 390, height: 844, isMobile: true, hasTouch: true })
  await settle()
  const sePayCard = await page.$('[data-testid="sepay-settings-card"]')
  assert.ok(sePayCard, 'Không tìm thấy card SePay để chụp bằng chứng mobile')
  await sePayCard.screenshot({ path: fileURLToPath(new URL('sepay-settings-mobile.png', artifactDirectory)) })
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
  assert.ok(overflow <= 1, `Giao diện mobile tràn ngang ${overflow}px`)
}

test('SePay Test Mode completes sandbox API, settings, webhook security, and reconciliation flows', { skip: !hasCredentials }, async () => {
  const browser = await puppeteer.launch({ headless: true, args: ['--no-sandbox'] })
  try {
    const sePayPage = await browser.newPage()
    await sePayPage.setViewport({ width: 1440, height: 1000 })
    await login(sePayPage, 'https://my.sepay.vn/login', sePayEmail, sePayPassword)
    await enableSePayTestMode(sePayPage)
    const bankAccount = await readSandboxBankAccount(sePayPage)
    const apiToken = await getOrCreateSandboxAPIToken(sePayPage)
    await verifySandboxAPI(apiToken)

    const appPage = await browser.newPage()
    await appPage.setViewport({ width: 1440, height: 1000 })
    const webhookSecret = randomBytes(32).toString('hex')
    const webhookURL = await configureApplication(appPage, bankAccount, apiToken, webhookSecret)
    await verifyApplicationFlow(appPage, webhookURL, bankAccount, webhookSecret)
    await captureResponsiveEvidence(appPage)
  } finally {
    await browser.close()
  }
})
