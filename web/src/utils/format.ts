export function fmtDate(iso: string | number | Date): string {
  return new Date(iso).toLocaleString("ja-JP", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function fmtDateFull(iso: string | Date): string {
  return new Date(iso).toLocaleString("ja-JP", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

export function fmtDuration(sec: number): string {
  if (sec < 60) return `${sec}秒`;
  if (sec < 3600) return `${Math.floor(sec / 60)}分${String(sec % 60).padStart(2, "0")}秒`;
  const h = Math.floor(sec / 3600);
  const m = Math.floor((sec % 3600) / 60);
  return `${h}時間${String(m).padStart(2, "0")}分`;
}

export function instanceRangeLabel(inst: { opened_at: string; closed_at?: string | null }): string {
  return inst.closed_at
    ? `${fmtDate(inst.opened_at)} 〜 ${fmtDate(inst.closed_at)}`
    : `${fmtDate(inst.opened_at)} 〜`;
}

export function extractInstanceNumber(locationId: string | null | undefined): string {
  const m = String(locationId ?? "").match(/:(\d+)/);
  return m ? m[1] : "";
}
