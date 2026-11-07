import { createContext } from 'react'
import type { ThuChiDB } from './db'
import type { SyncEngine } from './sync'
import type { Transport } from './types'
import type { PresetName } from './presets'

export interface ThuChiCtx {
  db: ThuChiDB
  engine: SyncEngine
  transport: Transport
  month: string
  setMonth: (m: string) => void
  preset: PresetName
}

export const Ctx = createContext<ThuChiCtx | null>(null)
