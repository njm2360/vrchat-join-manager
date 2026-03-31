from datetime import datetime

import aiosqlite

from models.common import EventOut
from models.locations import (
    InstanceOut,
    LocationPlayerOut,
    PlayerListOut,
    SessionOut,
    TimelinePoint,
)
from utils import to_utc_str


async def get_instances(
    db: aiosqlite.Connection,
    start: datetime | None,
    end: datetime | None,
    is_open: bool | None = None,
    world_id: str | None = None,
    group_id: str | None = None,
    region: str | None = None,
    sort_by: str = "opened_at",
    order: str = "desc",
    limit: int | None = None,
    offset: int = 0,
) -> list[InstanceOut]:
    conditions: list[str] = []
    params: dict = {}
    if start is not None:
        conditions.append("i.opened_at >= :start")
        params["start"] = to_utc_str(start)
    if end is not None:
        conditions.append("i.opened_at <= :end")
        params["end"] = to_utc_str(end)
    if is_open is True:
        conditions.append("i.closed_at IS NULL")
    elif is_open is False:
        conditions.append("i.closed_at IS NOT NULL")
    if world_id is not None:
        conditions.append("i.world_id = :world_id")
        params["world_id"] = world_id
    if group_id is not None:
        conditions.append("i.group_id = :group_id")
        params["group_id"] = group_id
    if region is not None:
        conditions.append("i.region = :region")
        params["region"] = region
    where_clause = ("WHERE " + " AND ".join(conditions)) if conditions else ""
    limit_clause = (
        f"LIMIT {limit} OFFSET {offset}"
        if limit is not None
        else f"LIMIT -1 OFFSET {offset}"
    )
    cursor = await db.execute(
        f"""
        SELECT i.id, i.location_id, i.world_id, w.name AS world_name,
               i.instance_id, i.group_id, g.name AS group_name,
               i.group_access_type, i.region, i.friends, i.hidden, i.opened_at, i.closed_at,
               (SELECT COUNT(*) FROM sessions s WHERE s.instance_id = i.id AND s.leave_ts IS NULL) AS user_count
        FROM instances i
        JOIN worlds w ON w.world_id = i.world_id
        LEFT JOIN groups g ON g.group_id = i.group_id
        {where_clause}
        ORDER BY {sort_by} {order.upper()}
        {limit_clause}
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [InstanceOut(**dict(row)) for row in rows]


async def get_presence(
    db: aiosqlite.Connection,
    instance_id: int,
    at: datetime,
) -> list[SessionOut]:
    cursor = await db.execute(
        """
        SELECT s.id, s.instance_id, s.user_id, p.display_name, s.join_ts, s.leave_ts,
               COALESCE(s.duration_seconds,
                   CAST(ROUND((julianday('now') - julianday(s.join_ts)) * 86400) AS INTEGER)
               ) AS duration_seconds,
               s.is_estimated_leave
        FROM sessions s
        JOIN players p ON p.user_id = s.user_id
        WHERE s.instance_id = :instance_id
          AND s.join_ts  <= :at
          AND (s.leave_ts IS NULL OR s.leave_ts >= :at)
        ORDER BY s.join_ts
        """,
        {"instance_id": instance_id, "at": to_utc_str(at)},
    )
    rows = await cursor.fetchall()
    return [SessionOut(**dict(row)) for row in rows]


async def get_location_players(
    db: aiosqlite.Connection,
    instance_id: int,
    sort_by: str = "internal_id",
    order: str = "asc",
) -> list[LocationPlayerOut]:
    cursor = await db.execute(
        f"""
        SELECT s.user_id, p.display_name, s.internal_id, s.join_ts,
               (SELECT COUNT(*) FROM sessions s2
                WHERE s2.user_id = s.user_id AND s2.instance_id = s.instance_id) AS join_count
        FROM sessions s
        JOIN players p ON p.user_id = s.user_id
        WHERE s.instance_id = :instance_id
          AND s.leave_ts IS NULL
        ORDER BY {"p." + sort_by if sort_by == "display_name" else "s." + sort_by} {order.upper()}
        """,
        {"instance_id": instance_id},
    )
    rows = await cursor.fetchall()
    return [LocationPlayerOut(**dict(row)) for row in rows]


async def get_location_visitors(
    db: aiosqlite.Connection,
    instance_id: int,
    sort_by: str = "last_seen",
    order: str = "desc",
    limit: int | None = None,
    offset: int = 0,
) -> list[PlayerListOut]:
    limit_clause = (
        f"LIMIT {limit} OFFSET {offset}"
        if limit is not None
        else f"LIMIT -1 OFFSET {offset}"
    )
    cursor = await db.execute(
        f"""
        SELECT s.user_id, p.display_name,
               MIN(s.join_ts)          AS first_seen,
               MAX(s.join_ts)          AS last_seen,
               COUNT(*)                AS join_count,
               SUM(COALESCE(s.duration_seconds,
                   CAST(ROUND((julianday('now') - julianday(s.join_ts)) * 86400) AS INTEGER)
               ))                      AS total_duration_seconds
        FROM sessions s
        JOIN players p ON p.user_id = s.user_id
        WHERE s.instance_id = :instance_id
        GROUP BY s.user_id
        ORDER BY {sort_by} {order.upper()}
        {limit_clause}
        """,
        {"instance_id": instance_id},
    )
    rows = await cursor.fetchall()
    return [PlayerListOut(**dict(row)) for row in rows]


async def get_presence_timeline(
    db: aiosqlite.Connection,
    instance_id: int,
    start: datetime | None,
    end: datetime | None,
) -> list[TimelinePoint]:
    start_str = to_utc_str(start) if start else None
    end_str = to_utc_str(end) if end else None

    # start 時点での在席数（start より前に join し、まだ leave していないセッション）
    if start_str:
        cursor = await db.execute(
            """
            SELECT COUNT(*) FROM sessions
            WHERE instance_id = :instance_id
              AND join_ts <= :start
              AND (leave_ts IS NULL OR leave_ts > :start)
            """,
            {"instance_id": instance_id, "start": start_str},
        )
        row = await cursor.fetchone()
        initial_count: int = row[0]
    else:
        initial_count = 0

    # 範囲内の join/leave イベントを時系列順に取得
    conditions = ["instance_id = :instance_id"]
    params: dict = {"instance_id": instance_id}
    if start_str:
        conditions.append("timestamp > :start")
        params["start"] = start_str
    if end_str:
        conditions.append("timestamp <= :end")
        params["end"] = end_str
    where = " AND ".join(conditions)

    cursor = await db.execute(
        f"SELECT event_type, timestamp FROM events WHERE {where} ORDER BY timestamp",
        params,
    )
    events = await cursor.fetchall()

    # 起点 + 各イベント時点のカウントを積み上げ
    anchor = start_str or (events[0]["timestamp"] if events else None)
    if anchor is None:
        return []
    points = [TimelinePoint(timestamp=anchor, count=initial_count)]
    count = initial_count
    for event in events:
        count += 1 if event["event_type"] == "join" else -1
        points.append(TimelinePoint(timestamp=event["timestamp"], count=count))

    return points


async def get_location_events(
    db: aiosqlite.Connection,
    instance_id: int,
    start: datetime | None,
    end: datetime | None,
    order: str = "desc",
    limit: int | None = None,
    offset: int = 0,
) -> list[EventOut]:
    conditions = ["instance_id = :instance_id"]
    params: dict = {"instance_id": instance_id}
    if start is not None:
        conditions.append("timestamp >= :start")
        params["start"] = to_utc_str(start)
    if end is not None:
        conditions.append("timestamp <= :end")
        params["end"] = to_utc_str(end)
    where = " AND ".join(conditions)
    limit_clause = (
        f"LIMIT {limit} OFFSET {offset}"
        if limit is not None
        else f"LIMIT -1 OFFSET {offset}"
    )
    cursor = await db.execute(
        f"""
        SELECT e.id, e.event_type, e.instance_id, e.world_id, e.user_id, p.display_name, e.timestamp
        FROM events e
        JOIN players p ON p.user_id = e.user_id
        WHERE {where}
        ORDER BY e.timestamp {order.upper()}
        {limit_clause}
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [EventOut(**dict(row)) for row in rows]


async def get_location_sessions(
    db: aiosqlite.Connection,
    instance_id: int,
    start: datetime | None,
    end: datetime | None,
    sort_by: str = "join_ts",
    order: str = "asc",
    limit: int | None = None,
    offset: int = 0,
) -> list[SessionOut]:
    conditions = ["instance_id = :instance_id"]
    params: dict = {"instance_id": instance_id}
    if start is not None:
        conditions.append("join_ts >= :start")
        params["start"] = to_utc_str(start)
    if end is not None:
        conditions.append("join_ts <= :end")
        params["end"] = to_utc_str(end)
    where = " AND ".join(conditions)
    limit_clause = (
        f"LIMIT {limit} OFFSET {offset}"
        if limit is not None
        else f"LIMIT -1 OFFSET {offset}"
    )
    cursor = await db.execute(
        f"""
        SELECT s.id, s.instance_id, s.user_id, p.display_name, s.join_ts, s.leave_ts,
               COALESCE(s.duration_seconds,
                   CAST(ROUND((julianday('now') - julianday(s.join_ts)) * 86400) AS INTEGER)
               ) AS duration_seconds,
               s.is_estimated_leave
        FROM sessions s
        JOIN players p ON p.user_id = s.user_id
        WHERE {where}
        ORDER BY {sort_by} {order.upper()}
        {limit_clause}
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [SessionOut(**dict(row)) for row in rows]
