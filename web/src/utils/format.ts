export function fmtDate(iso: string | Date): string {
  return new Date(iso).toLocaleString('ja-JP', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function fmtDateFull(iso: string | Date): string {
  return new Date(iso).toLocaleString('ja-JP', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

export function fmtDuration(sec: number): string {
  if (sec < 60) return `${sec}秒`
  if (sec < 3600) return `${Math.floor(sec / 60)}分${sec % 60}秒`
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  return `${h}時間${m}分`
}

export function extractInstanceNumber(locationId: string | null | undefined): string {
  const m = String(locationId ?? '').match(/:(\d+)/)
  return m ? m[1] : ''
}

// datetime-local 入力値 (ローカル時刻) を ISO 8601 (UTC) に変換
export function localDatetimeToIso(value: string): string {
  return new Date(value).toISOString()
}

// 現在時刻を datetime-local 入力に詰める形式 (YYYY-MM-DDTHH:mm)
export function nowLocalDatetimeValue(): string {
  const d = new Date()
  d.setMinutes(d.getMinutes() - d.getTimezoneOffset())
  return d.toISOString().slice(0, 16)
}
