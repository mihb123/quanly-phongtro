import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

export const E2E_DIR = path.dirname(path.dirname(fileURLToPath(import.meta.url)))
export const FRONTEND_DIR = path.resolve(E2E_DIR, '../..')
export const ROOT_DIR = path.dirname(FRONTEND_DIR)

export const TMP_DIR = process.env.E2E_TMP || path.join(os.tmpdir(), 'qlpt-thuchi-e2e')
export const ARTIFACTS_DIR = path.join(TMP_DIR, 'artifacts')
export const DOWNLOADS_DIR = path.join(TMP_DIR, 'downloads')
export const SERVER_LOG = process.env.E2E_SERVER_LOG || path.join(TMP_DIR, 'server.log')
export const BINARY = path.join(TMP_DIR, 'qlpt-api-e2e')

export const PORT = Number(process.env.E2E_PORT || 9192)
export const BASE_URL = process.env.E2E_BASE_URL || `http://localhost:${PORT}`
export const TELEGRAM_FAKE_URL = process.env.E2E_TELEGRAM_FAKE_URL || `http://127.0.0.1:${PORT + 1}`

export const PG_ADMIN_URL = process.env.E2E_PG_ADMIN_URL || 'postgres://thuchi@127.0.0.1:5440/postgres?sslmode=disable'
export const DB_NAME = process.env.E2E_DB_NAME || 'qlpt_thuchi_e2e'
export const DB_URL = process.env.E2E_DB_URL || PG_ADMIN_URL.replace(/\/postgres(\?|$)/, `/${DB_NAME}$1`)

export const HEADFUL = !!process.env.E2E_HEADFUL
export const SLOWMO = Number(process.env.E2E_SLOWMO || 0)
export const TIMEZONE = 'Asia/Ho_Chi_Minh'
export const PASSWORD = 'E2e-secret-123'
export const PASSWORD_HASH = '$2a$10$qsaPUEVdCzKFIHtDxEJMKeSVv7rauAjlOwL./t4EMvDlz8LfCV9d.'
