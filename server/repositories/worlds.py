from datetime import datetime

import aiosqlite

from models import SessionOut, WorldOut
from utils import to_utc_str


async def get_worlds(
    db: aiosqlite.Connection,
    start: datetime | None,
    end: datetime | None,
    order: str = "desc",
    limit: int | None = None,
    offset: int = 0,
) -> list[WorldOut]:
    having: list[str] = []
    params: dict = {}
    if start is not None:
        having.append("first_seen >= :start")
        params["start"] = to_utc_str(start)
    if end is not None:
        having.append("last_seen <= :end")
        params["end"] = to_utc_str(end)
    having_clause = ("HAVING " + " AND ".join(having)) if having else ""
    limit_clause = (
        f"LIMIT {limit} OFFSET {offset}"
        if limit is not None
        else f"LIMIT -1 OFFSET {offset}"
    )
    cursor = await db.execute(
        f"""
        SELECT world_id,
               MIN(join_ts) AS first_seen,
               MAX(join_ts) AS last_seen,
               COUNT(*)     AS session_count
        FROM sessions
        GROUP BY world_id
        {having_clause}
        ORDER BY last_seen {order.upper()}
        {limit_clause}
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [WorldOut(**dict(row)) for row in rows]


async def get_world_sessions(
    db: aiosqlite.Connection,
    world_id: str,
    start: datetime | None,
    end: datetime | None,
    order: str = "asc",
    limit: int | None = None,
    offset: int = 0,
) -> list[SessionOut]:
    conditions = ["world_id = :world_id"]
    params: dict = {"world_id": world_id}

    if start is not None:
        conditions.append("join_ts >= :start")
        params["start"] = start.isoformat()
    if end is not None:
        conditions.append("join_ts <= :end")
        params["end"] = end.isoformat()

    where = " AND ".join(conditions)
    limit_clause = (
        f"LIMIT {limit} OFFSET {offset}"
        if limit is not None
        else f"LIMIT -1 OFFSET {offset}"
    )
    cursor = await db.execute(
        f"""
        SELECT id, location_id, user_id, display_name, join_ts, leave_ts,
               duration_seconds
        FROM sessions
        WHERE {where}
        ORDER BY join_ts {order.upper()}
        {limit_clause}
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [SessionOut(**dict(row)) for row in rows]
