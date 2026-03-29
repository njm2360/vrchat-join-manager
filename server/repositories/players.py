from datetime import datetime

import aiosqlite

from models import EventOut, PlayerListOut, PlayerSessionOut
from utils import to_utc_str


async def get_players(
    db: aiosqlite.Connection,
    name: str | None,
    order: str = "asc",
    limit: int | None = None,
    offset: int = 0,
) -> list[PlayerListOut]:
    conditions: list[str] = []
    params: dict = {}
    if name is not None:
        conditions.append("p.display_name LIKE :name")
        params["name"] = f"%{name}%"
    where = ("WHERE " + " AND ".join(conditions)) if conditions else ""
    limit_clause = (
        f"LIMIT {limit} OFFSET {offset}"
        if limit is not None
        else f"LIMIT -1 OFFSET {offset}"
    )
    cursor = await db.execute(
        f"""
        SELECT p.user_id, p.display_name,
               MIN(s.join_ts) AS first_seen,
               MAX(s.join_ts) AS last_seen,
               COUNT(s.id)    AS join_count
        FROM players p
        LEFT JOIN sessions s ON s.user_id = p.user_id
        {where}
        GROUP BY p.user_id
        ORDER BY last_seen {order.upper()}
        {limit_clause}
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [PlayerListOut(**dict(row)) for row in rows]


async def get_player_events(
    db: aiosqlite.Connection,
    user_id: str,
    instance_id: int | None,
    start: datetime | None,
    end: datetime | None,
    order: str = "asc",
    limit: int | None = None,
    offset: int = 0,
) -> list[EventOut]:
    conditions = ["user_id = :user_id"]
    params: dict = {"user_id": user_id}

    if instance_id is not None:
        conditions.append("instance_id = :instance_id")
        params["instance_id"] = instance_id
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


async def get_player_sessions(
    db: aiosqlite.Connection,
    user_id: str,
    instance_id: int | None,
    world_id: str | None,
    group_id: str | None,
    start: datetime | None,
    end: datetime | None,
    order: str = "asc",
    limit: int | None = None,
    offset: int = 0,
) -> list[PlayerSessionOut]:
    conditions = ["s.user_id = :user_id"]
    params: dict = {"user_id": user_id}

    if instance_id is not None:
        conditions.append("s.instance_id = :instance_id")
        params["instance_id"] = instance_id
    if world_id is not None:
        conditions.append("i.world_id = :world_id")
        params["world_id"] = world_id
    if group_id is not None:
        conditions.append("i.group_id = :group_id")
        params["group_id"] = group_id
    if start is not None:
        conditions.append("s.join_ts >= :start")
        params["start"] = to_utc_str(start)
    if end is not None:
        conditions.append("s.join_ts <= :end")
        params["end"] = to_utc_str(end)

    where = " AND ".join(conditions)
    limit_clause = (
        f"LIMIT {limit} OFFSET {offset}"
        if limit is not None
        else f"LIMIT -1 OFFSET {offset}"
    )
    cursor = await db.execute(
        f"""
        SELECT s.id, s.instance_id, i.world_id, s.join_ts, s.leave_ts,
               COALESCE(s.duration_seconds,
                   CAST(ROUND((julianday('now') - julianday(s.join_ts)) * 86400) AS INTEGER)
               ) AS duration_seconds,
               s.is_estimated_leave
        FROM sessions s
        JOIN instances i ON i.id = s.instance_id
        WHERE {where}
        ORDER BY s.join_ts {order.upper()}
        {limit_clause}
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [PlayerSessionOut(**dict(row)) for row in rows]
