from datetime import datetime

import aiosqlite

from models import (
    EventOut,
    InstanceOut,
    PlayerListOut,
    PlayerOut,
    SessionOut,
    TimelinePoint,
)
from utils import to_utc_str


async def get_instances(
    db: aiosqlite.Connection,
    start: datetime | None,
    end: datetime | None,
    order: str = "desc",
    limit: int | None = None,
    offset: int = 0,
) -> list[InstanceOut]:
    conditions: list[str] = []
    params: dict = {}
    if start is not None:
        conditions.append("opened_at >= :start")
        params["start"] = to_utc_str(start)
    if end is not None:
        conditions.append("opened_at <= :end")
        params["end"] = to_utc_str(end)
    where_clause = ("WHERE " + " AND ".join(conditions)) if conditions else ""
    limit_clause = (
        f"LIMIT {limit} OFFSET {offset}"
        if limit is not None
        else f"LIMIT -1 OFFSET {offset}"
    )
    cursor = await db.execute(
        f"""
        SELECT id, location_id, world_id, instance_id, group_id, group_access_type, region, friends, hidden, opened_at, closed_at
        FROM instances
        {where_clause}
        ORDER BY opened_at {order.upper()}
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
        SELECT id, instance_id, user_id, display_name, join_ts, leave_ts,
               COALESCE(duration_seconds,
                   CAST(ROUND((julianday('now') - julianday(join_ts)) * 86400) AS INTEGER)
               ) AS duration_seconds,
               is_estimated_leave
        FROM sessions
        WHERE instance_id = :instance_id
          AND join_ts  <= :at
          AND (leave_ts IS NULL OR leave_ts >= :at)
        ORDER BY join_ts
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
) -> list[PlayerOut]:
    cursor = await db.execute(
        f"""
        SELECT s.user_id, s.display_name, e.internal_id, s.join_ts,
               (SELECT COUNT(*) FROM sessions s2
                WHERE s2.user_id = s.user_id AND s2.instance_id = s.instance_id) AS join_count
        FROM sessions s
        JOIN events e ON e.id = s.join_event_id
        WHERE s.instance_id = :instance_id
          AND s.leave_ts IS NULL
        ORDER BY {"s." + sort_by if sort_by in ("display_name", "join_ts") else sort_by} {order.upper()}
        """,
        {"instance_id": instance_id},
    )
    rows = await cursor.fetchall()
    return [PlayerOut(**dict(row)) for row in rows]


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
        SELECT user_id, display_name,
               MIN(join_ts)          AS first_seen,
               MAX(join_ts)          AS last_seen,
               COUNT(*)              AS join_count,
               SUM(COALESCE(duration_seconds,
                   CAST(ROUND((julianday('now') - julianday(join_ts)) * 86400) AS INTEGER)
               ))                    AS total_duration_seconds
        FROM sessions
        WHERE instance_id = :instance_id
        GROUP BY user_id
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
        SELECT id, event_type, instance_id, world_id, user_id, display_name, internal_id, timestamp
        FROM events
        WHERE {where}
        ORDER BY timestamp {order.upper()}
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
        SELECT id, instance_id, user_id, display_name, join_ts, leave_ts,
               COALESCE(duration_seconds,
                   CAST(ROUND((julianday('now') - julianday(join_ts)) * 86400) AS INTEGER)
               ) AS duration_seconds,
               is_estimated_leave
        FROM sessions
        WHERE {where}
        ORDER BY {sort_by} {order.upper()}
        {limit_clause}
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [SessionOut(**dict(row)) for row in rows]
