import type { SessionOut, TimelinePoint } from '../api/schemas'

export type Point = { x: Date; y: number }
export type InstColor = 'blue' | 'red'

export interface Violation {
  user_id: string
  display_name: string
  join_ts: Date
  instance: InstColor
  diff: number
  duration_seconds: number | null
}

export type SessionMap = Map<string, Array<{ join_s: number; duration_seconds: number | null }>>

const toSec = (d: Date | string | number): number => {
  const ms = typeof d === 'number' ? d : new Date(d).getTime()
  return Math.floor(ms / 1000)
}

export function buildPoints(data: TimelinePoint[], closedAt: string | null | undefined): Point[] {
  const pts: Point[] = data.map((d) => ({ x: new Date(d.timestamp), y: d.count }))
  if (!closedAt && pts.length > 0) {
    pts.push({ x: new Date(), y: pts[pts.length - 1].y })
  }
  return pts
}

// ステップ補間: 秒 t における pts の値を返す。
export function stepValue(pts: Point[], t: number): number {
  let val = 0
  for (const p of pts) {
    if (toSec(p.x) <= t) val = p.y
    else break
  }
  return val
}

// セッションマップ: user_id -> [{join_s, duration_seconds}]
export function buildSessionMap(sessions: SessionOut[]): SessionMap {
  const map: SessionMap = new Map()
  for (const s of sessions) {
    if (!map.has(s.user_id)) map.set(s.user_id, [])
    map.get(s.user_id)!.push({
      join_s: toSec(s.join_ts),
      duration_seconds: s.duration_seconds ?? null,
    })
  }
  return map
}

function lookupDuration(sessions: SessionMap, userId: string, joinSec: number): number | null {
  return sessions.get(userId)?.find((s) => s.join_s === joinSec)?.duration_seconds ?? null
}

function isRejoin(sessions: SessionMap, userId: string, t: number, rejoinSeconds: number): boolean {
  const arr = sessions.get(userId)
  if (!arr) return false
  return arr.some((s) => {
    if (s.join_s >= t || s.duration_seconds == null) return false
    const leaveSec = s.join_s + s.duration_seconds
    const gapSec = t - leaveSec
    return gapSec >= 0 && gapSec <= rejoinSeconds
  })
}

// 両インスタンスの全イベント時刻を走査し、自インスタンスが相手より多い状態の開始時刻を
// 任意の時刻に対して返すルックアップ関数を構築する。
function buildDiffStartLookup(tl: TimelinePoint[], otherPts: Point[]) {
  const selfPts: Point[] = tl.map((d) => ({ x: new Date(d.timestamp), y: d.count }))
  const times = [
    ...new Set([
      ...selfPts.map((p) => toSec(p.x)),
      ...otherPts.map((p) => toSec(p.x)),
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

// color ('blue'|'red') のインスタンスへの違反Joinを検出する。
// 違反 = 自インスタンスの方が相手より人が多い状態でJoinした。
// graceSeconds: 差が発生してからこの秒数以内なら人数表示未更新として許容する。
export function detectViolations(
  tl: TimelinePoint[],
  otherPts: Point[],
  sessions: SessionMap,
  color: InstColor,
  graceSeconds: number,
): Violation[] {
  const REJOIN_S = 180     // 退室後のこの秒数以内に再入室した場合はリジョインとして除外
  const SHORT_STAY_S = 180 // 違反後この秒数以内に退室 → 違反に気付き離脱として許容

  const violations: Violation[] = []
  const getDiffStart = buildDiffStartLookup(tl, otherPts)
  for (let i = 1; i < tl.length; i++) {
    const pt = tl[i]
    if (!pt.user_id) continue
    const countBefore = tl[i - 1].count // このイベント直前の自インスタンス人数
    if (pt.count <= countBefore) continue // Joinでない (Leave)

    const t = toSec(pt.timestamp)
    if (isRejoin(sessions, pt.user_id, t, REJOIN_S)) continue // 直近のRejoin → 除外

    const otherCount = stepValue(otherPts, t)
    if (otherCount === 0) continue // 相手インスタンスがまだ存在しない期間は除外
    const diff = countBefore - otherCount
    if (diff <= 0) continue

    const diffStart = getDiffStart(t)
    if (diffStart === null || t - diffStart <= graceSeconds) continue // 差発生から猶予秒以内 → 除外

    const duration_seconds = lookupDuration(sessions, pt.user_id, t)
    if (duration_seconds != null && duration_seconds <= SHORT_STAY_S) continue // 短時間で退出 → 違反に気付き離脱として許容

    violations.push({
      user_id: pt.user_id,
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
      ...pts1.map((p) => toSec(p.x)),
      ...pts2.map((p) => toSec(p.x)),
    ]),
  ].sort((a, b) => a - b)

  return times
    .filter((t) => stepValue(pts1, t) > 0 && stepValue(pts2, t) > 0)
    .map((t) => ({ x: new Date(t * 1000), y: stepValue(pts1, t) - stepValue(pts2, t) }))
}
