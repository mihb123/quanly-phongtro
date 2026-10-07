export type SePayEnvironment = 'production' | 'sandbox'
export type SePayWebhookAuthMethod = 'hmac' | 'apikey' | 'none' | ''

// Empty fields keep the values already stored on the server, so edits only send what changed.
export interface SePayConfigInput {
  environment: SePayEnvironment
  bank_short_name: string
  account_number: string
  account_name: string
  code_prefix: string
  webhook_auth_method: SePayWebhookAuthMethod | (string & {})
  webhook_api_key?: string
  webhook_secret?: string
  api_token?: string
}

export interface SePayConfigStatus {
  has_config: boolean
  is_active: boolean
  provider: string
  environment: SePayEnvironment
  masked_account_number: string
  account_name: string
  bank_short_name: string
  code_prefix: string
  webhook_auth_method: SePayWebhookAuthMethod
  has_webhook_api_key: boolean
  has_webhook_secret: boolean
  has_api_token: boolean
  webhook_url: string
}

export interface SePayConfigSaveResult {
  success: boolean
  bank_account_verified: boolean
  account_holder_name?: string
}

export interface SePayReconcileResult {
  pages_fetched: number
  scanned: number
  processed: number
  failed: number
  truncated: boolean
}

export type SePayTransport = <T>(method: 'GET' | 'POST' | 'DELETE', path: string, body?: unknown) => Promise<T>

export interface SePayApiPaths {
  publicKey: string
  config: string
  reconcile: string
}

export const DEFAULT_SEPAY_API_PATHS: SePayApiPaths = {
  publicKey: '/payments/public-key',
  config: '/payments/providers/sepay/config',
  reconcile: '/payments/providers/sepay/reconcile',
}

const SECRET_FIELDS = ['webhook_api_key', 'webhook_secret', 'api_token'] as const

export async function encryptRSA(text: string, spkiBase64: string): Promise<string> {
  const binary = atob(spkiBase64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i)
  }
  const key = await crypto.subtle.importKey('spki', bytes, { name: 'RSA-OAEP', hash: 'SHA-256' }, false, ['encrypt'])
  const encrypted = await crypto.subtle.encrypt({ name: 'RSA-OAEP' }, key, new TextEncoder().encode(text))
  return btoa(String.fromCharCode(...new Uint8Array(encrypted)))
}

export function fetchTransport(baseURL: string, init?: () => RequestInit): SePayTransport {
  return async <T>(method: string, path: string, body?: unknown): Promise<T> => {
    const extra = init?.() ?? {}
    const response = await fetch(`${baseURL.replace(/\/$/, '')}${path}`, {
      credentials: 'include',
      ...extra,
      method,
      headers: { 'Content-Type': 'application/json', ...(extra.headers ?? {}) },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
    if (!response.ok) {
      throw new Error((await response.text()) || `SePay API ${response.status}`)
    }
    return (response.status === 204 ? undefined : await response.json()) as T
  }
}

export function createSePayApi(transport: SePayTransport, paths: SePayApiPaths = DEFAULT_SEPAY_API_PATHS) {
  const getPublicKey = () => transport<{ public_key: string }>('GET', paths.publicKey)

  return {
    getPublicKey,
    getConfig: () => transport<SePayConfigStatus>('GET', paths.config),
    deleteConfig: () => transport<{ success: boolean }>('DELETE', paths.config),
    reconcile: (window?: { date_from?: string; date_to?: string }) =>
      transport<SePayReconcileResult>('POST', paths.reconcile, window ?? {}),
    async saveConfig(input: SePayConfigInput): Promise<SePayConfigSaveResult> {
      const { public_key: publicKey } = await getPublicKey()
      const payload: SePayConfigInput = { ...input }
      for (const field of SECRET_FIELDS) {
        const value = input[field]
        payload[field] = value ? await encryptRSA(value, publicKey) : undefined
      }
      return transport<SePayConfigSaveResult>('POST', paths.config, payload)
    },
  }
}

export type SePayApi = ReturnType<typeof createSePayApi>
