import { createContext } from 'react'
import type { ThuChiDB } from './db'
import type { SyncEngine } from './sync'
import type { Transport } from './types'

export interface ThuChiCtx {
  db: ThuChiDB
  engine: SyncEngine
  transport: Transport
  month: string
  setMonth: (m: string) => void
}

export const Ctx = createContext<ThuChiCtx | null>(null)
