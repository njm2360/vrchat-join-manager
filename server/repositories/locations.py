from datetime import datetime

import aiosqlite

from models import EventOut, LocationOut, PlayerOut, SessionOut, TimelinePoint
from utils import to_utc_str


async def get_locations(
    db: aiosqlite.Connection,
    start: datetime | None,
    end: datetime | None,
    order: str = "desc",
) -> list[LocationOut]:
    having: list[str] = []
    params: dict = {}
    if start is not None:
        having.append("first_seen >= :start")
        params["start"] = to_utc_str(start)
    if end is not None:
        having.append("last_seen <= :end")
        params["end"] = to_utc_str(end)
    having_clause = ("HAVING " + " AND ".join(having)) if having else ""
    cursor = await db.execute(
        f"""
        SELECT location_id, world_id, MIN(join_ts) AS first_seen, MAX(join_ts) AS last_seen
        FROM sessions
        GROUP BY location_id
        {having_clause}
        ORDER BY last_seen {order.upper()}
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [LocationOut(**dict(row)) for row in rows]


async def get_presence(
    db: aiosqlite.Connection,
    location_id: str,
    at: datetime,
) -> list[SessionOut]:
    cursor = await db.execute(
        """
        SELECT id, location_id, user_id, display_name, join_ts, leave_ts,
               duration_seconds
        FROM sessions
        WHERE location_id = :location_id
          AND join_ts  <= :at
          AND (leave_ts IS NULL OR leave_ts >= :at)
        ORDER BY join_ts
        """,
        {"location_id": location_id, "at": to_utc_str(at)},
    )
    rows = await cursor.fetchall()
    return [SessionOut(**dict(row)) for row in rows]


async def get_location_players(
    db: aiosqlite.Connection,
    location_id: str,
) -> list[PlayerOut]:
    cursor = await db.execute(
        """
        SELECT s.user_id, s.display_name, e.internal_id, s.join_ts,
               (SELECT COUNT(*) FROM sessions s2
                WHERE s2.user_id = s.user_id AND s2.location_id = s.location_id) AS join_count
        FROM sessions s
        JOIN events e ON e.id = s.join_event_id
        WHERE s.location_id = :location_id
          AND s.leave_ts IS NULL
        ORDER BY e.internal_id ASC
        """,
        {"location_id": location_id},
    )
    rows = await cursor.fetchall()
    return [PlayerOut(**dict(row)) for row in rows]


async def get_presence_timeline(
    db: aiosqlite.Connection,
    location_id: str,
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
            WHERE location_id = :location_id
              AND join_ts <= :start
              AND (leave_ts IS NULL OR leave_ts > :start)
            """,
            {"location_id": location_id, "start": start_str},
        )
        row = await cursor.fetchone()
        initial_count: int = row[0]
    else:
        initial_count = 0

    # 範囲内の join/leave イベントを時系列順に取得
    conditions = ["location_id = :location_id"]
    params: dict = {"location_id": location_id}
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
    location_id: str,
    start: datetime | None,
    end: datetime | None,
    order: str = "desc",
) -> list[EventOut]:
    conditions = ["location_id = :location_id"]
    params: dict = {"location_id": location_id}
    if start is not None:
        conditions.append("timestamp >= :start")
        params["start"] = to_utc_str(start)
    if end is not None:
        conditions.append("timestamp <= :end")
        params["end"] = to_utc_str(end)
    where = " AND ".join(conditions)
    cursor = await db.execute(
        f"""
        SELECT id, event_type, location_id, world_id, user_id, display_name, internal_id, timestamp
        FROM events
        WHERE {where}
        ORDER BY timestamp {order.upper()}
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [EventOut(**dict(row)) for row in rows]


async def get_location_sessions(
    db: aiosqlite.Connection,
    location_id: str,
    start: datetime | None,
    end: datetime | None,
    order: str = "asc",
) -> list[SessionOut]:
    conditions = ["location_id = :location_id"]
    params: dict = {"location_id": location_id}
    if start is not None:
        conditions.append("join_ts >= :start")
        params["start"] = to_utc_str(start)
    if end is not None:
        conditions.append("join_ts <= :end")
        params["end"] = to_utc_str(end)
    where = " AND ".join(conditions)
    cursor = await db.execute(
        f"""
        SELECT id, location_id, user_id, display_name, join_ts, leave_ts, duration_seconds
        FROM sessions
        WHERE {where}
        ORDER BY join_ts {order.upper()}
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [SessionOut(**dict(row)) for row in rows]
