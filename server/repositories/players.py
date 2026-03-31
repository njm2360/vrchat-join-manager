from datetime import datetime

import aiosqlite

from models.common import EventOut
from models.players import PlayerOut, PlayerSessionOut
from utils import to_utc_str


async def get_players(
    db: aiosqlite.Connection,
    name: str | None,
    order: str = "asc",
    limit: int | None = None,
    offset: int = 0,
) -> list[PlayerOut]:
    conditions: list[str] = []
    params: dict = {}
    if name is not None:
        conditions.append("display_name LIKE :name")
        params["name"] = f"%{name}%"
    where = ("WHERE " + " AND ".join(conditions)) if conditions else ""
    limit_clause = (
        f"LIMIT {limit} OFFSET {offset}"
        if limit is not None
        else f"LIMIT -1 OFFSET {offset}"
    )
    cursor = await db.execute(
        f"""
        SELECT user_id, display_name, created_at, updated_at
        FROM players
        {where}
        ORDER BY created_at {order.upper()}
        {limit_clause}
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [PlayerOut(**dict(row)) for row in rows]


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
    conditions = ["e.user_id = :user_id"]
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
