import { createContext, lazy, Suspense, useCallback, useContext, useState, type ReactNode } from 'react'
import type { PlayerDetailCtx } from './dialogs/PlayerDetailDialog'

const PlayerDetailDialog = lazy(() => import('./dialogs/PlayerDetailDialog'))

interface ContextValue {
  open: (ctx: PlayerDetailCtx) => void
}

const PlayerDetailContext = createContext<ContextValue | null>(null)

export function PlayerDetailProvider({ children }: { children: ReactNode }) {
  const [isOpen, setIsOpen] = useState(false)
  const [ctx, setCtx] = useState<PlayerDetailCtx | null>(null)

  const open = useCallback((c: PlayerDetailCtx) => {
    setCtx(c)
    setIsOpen(true)
  }, [])

  const close = useCallback(() => {
    setIsOpen(false)
  }, [])

  return (
    <PlayerDetailContext.Provider value={{ open }}>
      {children}
      {ctx && (
        <Suspense fallback={null}>
          <PlayerDetailDialog open={isOpen} onClose={close} ctx={ctx} />
        </Suspense>
      )}
    </PlayerDetailContext.Provider>
  )
}

export function usePlayerDetailDialog() {
  const value = useContext(PlayerDetailContext)
  if (!value) throw new Error('PlayerDetailProvider が必要です')
  return value
}
