from datetime import datetime

import aiosqlite

from models import EventOut, PlayerSessionOut


async def get_player_events(
    db: aiosqlite.Connection,
    user_id: str,
    location_id: str | None,
    start: datetime | None,
    end: datetime | None,
) -> list[EventOut]:
    conditions = ["user_id = :user_id"]
    params: dict = {"user_id": user_id}

    if location_id is not None:
        conditions.append("location_id = :location_id")
        params["location_id"] = location_id
    if start is not None:
        conditions.append("timestamp >= :start")
        params["start"] = start.isoformat()
    if end is not None:
        conditions.append("timestamp <= :end")
        params["end"] = end.isoformat()

    where = " AND ".join(conditions)
    cursor = await db.execute(
        f"""
        SELECT id, event_type, location_id, world_id, user_id, display_name, internal_id, timestamp
        FROM events
        WHERE {where}
        ORDER BY timestamp
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [EventOut(**dict(row)) for row in rows]


async def get_player_sessions(
    db: aiosqlite.Connection,
    user_id: str,
    location_id: str | None,
    start: datetime | None,
    end: datetime | None,
) -> list[PlayerSessionOut]:
    conditions = ["user_id = :user_id"]
    params: dict = {"user_id": user_id}

    if location_id is not None:
        conditions.append("location_id = :location_id")
        params["location_id"] = location_id
    if start is not None:
        conditions.append("join_ts >= :start")
        params["start"] = start.isoformat()
    if end is not None:
        conditions.append("join_ts <= :end")
        params["end"] = end.isoformat()

    where = " AND ".join(conditions)
    cursor = await db.execute(
        f"""
        SELECT id, location_id, join_ts, leave_ts,
               duration_seconds
        FROM sessions
        WHERE {where}
        ORDER BY join_ts
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [PlayerSessionOut(**dict(row)) for row in rows]
