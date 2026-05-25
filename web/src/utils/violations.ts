import type { SessionOut, TimelinePoint } from '../api/schemas'

export type Point = { x: Date; y: number }
export type InstColor = 'blue' | 'red'

export interface Violation {
  display_name: string
  join_ts: Date
  instance: InstColor
  diff: number
  duration_seconds: number | null
}

export type SessionMap = Map<string, Array<{ join_ms: number; duration_seconds: number | null }>>

const REJOIN_MS = 3 * 60 * 1000

export function buildPoints(data: TimelinePoint[], closedAt: string | null | undefined): Point[] {
  const pts: Point[] = data.map((d) => ({ x: new Date(d.timestamp), y: d.count }))
  if (!closedAt && pts.length > 0) {
    pts.push({ x: new Date(), y: pts[pts.length - 1].y })
  }
  return pts
}

export function stepValue(pts: Point[], t: number): number {
  let val = 0
  for (const p of pts) {
    if (p.x.getTime() <= t) val = p.y
    else break
  }
  return val
}

export function buildSessionMap(sessions: SessionOut[]): SessionMap {
  const map: SessionMap = new Map()
  for (const s of sessions) {
    const ms = new Date(s.join_ts).getTime()
    if (!map.has(s.user_id)) map.set(s.user_id, [])
    map.get(s.user_id)!.push({
      join_ms: ms,
      duration_seconds: s.duration_seconds ?? null,
    })
  }
  return map
}

function lookupDuration(sessions: SessionMap, userId: string, joinMs: number): number | null {
  return sessions.get(userId)?.find((s) => s.join_ms === joinMs)?.duration_seconds ?? null
}

function isRejoin(sessions: SessionMap, userId: string, t: number): boolean {
  const arr = sessions.get(userId)
  if (!arr) return false
  return arr.some((s) => {
    if (s.join_ms >= t || s.duration_seconds == null) return false
    const leaveMs = s.join_ms + s.duration_seconds * 1000
    return leaveMs <= t && t - leaveMs <= REJOIN_MS
  })
}

// 「自インスタンス > 相手」が開始した時刻のスナップショットを構築し、
// 任意の時刻 t に対するその時点での streakStart を返すルックアップを生成
function buildDiffStartLookup(tl: TimelinePoint[], otherPts: Point[]) {
  const selfPts: Point[] = tl.map((d) => ({ x: new Date(d.timestamp), y: d.count }))
  const times = [
    ...new Set([
      ...selfPts.map((p) => p.x.getTime()),
      ...otherPts.map((p) => p.x.getTime()),
    ]),
  ].sort((a, b) => a - b)

  let streakStart: number | null = null
  const snapshots: { t: number; streakStart: number | null }[] = []
  for (const t of times) {
    const diff = stepValue(selfPts, t) - stepValue(otherPts, t)
    if (diff > 0) {
      if (streakStart === null) streakStart = t
    } else {
      streakStart = null
    }
    snapshots.push({ t, streakStart })
  }

  return (t: number): number | null => {
    if (snapshots.length === 0 || snapshots[0].t > t) return null
    let lo = 0
    let hi = snapshots.length - 1
    while (lo < hi) {
      const mid = (lo + hi + 1) >> 1
      if (snapshots[mid].t <= t) lo = mid
      else hi = mid - 1
    }
    return snapshots[lo].streakStart
  }
}

export function detectViolations(
  tl: TimelinePoint[],
  otherPts: Point[],
  sessions: SessionMap,
  color: InstColor,
  graceMs: number,
): Violation[] {
  const violations: Violation[] = []
  const getDiffStart = buildDiffStartLookup(tl, otherPts)
  for (let i = 1; i < tl.length; i++) {
    const pt = tl[i]
    if (!pt.user_id) continue
    const countBefore = tl[i - 1].count
    if (pt.count <= countBefore) continue // Leave

    const t = new Date(pt.timestamp).getTime()
    if (isRejoin(sessions, pt.user_id, t)) continue

    const otherCount = stepValue(otherPts, t)
    if (otherCount === 0) continue
    const diff = countBefore - otherCount
    if (diff <= 0) continue

    const diffStart = getDiffStart(t)
    if (diffStart === null || t - diffStart <= graceMs) continue

    const duration_seconds = lookupDuration(sessions, pt.user_id, t)
    if (duration_seconds != null && duration_seconds <= 180) continue

    violations.push({
      display_name: pt.display_name ?? '',
      join_ts: new Date(pt.timestamp),
      instance: color,
      diff,
      duration_seconds,
    })
  }
  return violations
}

export function buildDiffPoints(pts1: Point[], pts2: Point[]): Point[] {
  const times = [
    ...new Set([
      ...pts1.map((p) => p.x.getTime()),
      ...pts2.map((p) => p.x.getTime()),
    ]),
  ].sort((a, b) => a - b)

  return times
    .filter((t) => stepValue(pts1, t) > 0 && stepValue(pts2, t) > 0)
    .map((t) => ({ x: new Date(t), y: stepValue(pts1, t) - stepValue(pts2, t) }))
}
