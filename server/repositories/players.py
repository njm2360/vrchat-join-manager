from datetime import datetime

import aiosqlite

from models import EventOut, PlayerSessionOut
from utils import to_utc_str


async def get_player_events(
    db: aiosqlite.Connection,
    user_id: str,
    location_id: str | None,
    start: datetime | None,
    end: datetime | None,
    order: str = "asc",
    limit: int | None = None,
    offset: int = 0,
) -> list[EventOut]:
    conditions = ["user_id = :user_id"]
    params: dict = {"user_id": user_id}

    if location_id is not None:
        conditions.append("location_id = :location_id")
        params["location_id"] = location_id
    if start is not None:
        conditions.append("timestamp >= :start")
        params["start"] = to_utc_str(start)
    if end is not None:
        conditions.append("timestamp <= :end")
        params["end"] = to_utc_str(end)

    where = " AND ".join(conditions)
    limit_clause = f"LIMIT {limit} OFFSET {offset}" if limit is not None else f"LIMIT -1 OFFSET {offset}"
    cursor = await db.execute(
        f"""
        SELECT id, event_type, location_id, world_id, user_id, display_name, internal_id, timestamp
        FROM events
        WHERE {where}
        ORDER BY timestamp {order.upper()}
        {limit_clause}
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
    order: str = "asc",
    limit: int | None = None,
    offset: int = 0,
) -> list[PlayerSessionOut]:
    conditions = ["user_id = :user_id"]
    params: dict = {"user_id": user_id}

    if location_id is not None:
        conditions.append("location_id = :location_id")
        params["location_id"] = location_id
    if start is not None:
        conditions.append("join_ts >= :start")
        params["start"] = to_utc_str(start)
    if end is not None:
        conditions.append("join_ts <= :end")
        params["end"] = to_utc_str(end)

    where = " AND ".join(conditions)
    limit_clause = f"LIMIT {limit} OFFSET {offset}" if limit is not None else f"LIMIT -1 OFFSET {offset}"
    cursor = await db.execute(
        f"""
        SELECT id, location_id, join_ts, leave_ts,
               duration_seconds
        FROM sessions
        WHERE {where}
        ORDER BY join_ts {order.upper()}
        {limit_clause}
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [PlayerSessionOut(**dict(row)) for row in rows]
