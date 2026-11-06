import { useEffect, useMemo, useState } from 'react'
import { ThuChiDB, dbName } from './db'
import { SyncEngine } from './sync'
import type { Transport } from './types'
import { Ctx } from './ctx'

export function ThuChiProvider({ ownerId, transport, month, setMonth, children }: {
  ownerId: string
  transport: Transport
  month: string
  setMonth: (m: string) => void
  children: React.ReactNode
}) {
  const [{ db, engine }] = useState(() => {
    const db = new ThuChiDB(dbName(ownerId))
    return { db, engine: new SyncEngine(db, transport) }
  })
  useEffect(() => {
    engine.start()
    return () => engine.stop()
  }, [engine])
  const value = useMemo(() => ({ db, engine, transport, month, setMonth }), [db, engine, transport, month, setMonth])
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}
