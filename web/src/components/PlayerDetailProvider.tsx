import { lazy, Suspense, useCallback, useMemo, useState, type ReactNode } from "react";
import type { PlayerDetailCtx } from "@/components/dialogs/PlayerDetailDialog";
import { PlayerDetailContext } from "@/components/usePlayerDetailDialog";

const PlayerDetailDialog = lazy(() => import("@/components/dialogs/PlayerDetailDialog"));

export function PlayerDetailProvider({ children }: { children: ReactNode }) {
  const [isOpen, setIsOpen] = useState(false);
  const [ctx, setCtx] = useState<PlayerDetailCtx | null>(null);

  const open = useCallback((c: PlayerDetailCtx) => {
    setCtx(c);
    setIsOpen(true);
  }, []);

  const close = useCallback(() => {
    setIsOpen(false);
  }, []);

  const value = useMemo(() => ({ open }), [open]);

  return (
    <PlayerDetailContext value={value}>
      {children}
      {ctx && (
        <Suspense fallback={null}>
          <PlayerDetailDialog
            key={`${ctx.userId}:${ctx.instanceId ?? ""}`}
            open={isOpen}
            onClose={close}
            ctx={ctx}
          />
        </Suspense>
      )}
    </PlayerDetailContext>
  );
}
