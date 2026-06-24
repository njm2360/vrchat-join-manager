import { createContext, use } from "react";
import type { PlayerDetailCtx } from "@/components/dialogs/PlayerDetailDialog";

export interface PlayerDetailContextValue {
  open: (ctx: PlayerDetailCtx) => void;
}

export const PlayerDetailContext = createContext<PlayerDetailContextValue | null>(null);

export function usePlayerDetailDialog() {
  const value = use(PlayerDetailContext);
  if (!value) throw new Error("PlayerDetailProvider が必要です");
  return value;
}
