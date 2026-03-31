from datetime import datetime

import aiosqlite

from models.worlds import WorldOut
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
        having.append("w.created_at >= :start")
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
        SELECT w.world_id,
               w.name,
               w.created_at,
               w.updated_at,
               MAX(s.join_ts) AS last_seen,
               COUNT(s.id)    AS session_count
        FROM worlds w
        LEFT JOIN sessions s ON s.world_id = w.world_id
        GROUP BY w.world_id
        {having_clause}
        ORDER BY last_seen {order.upper()} NULLS LAST
        {limit_clause}
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [WorldOut(**dict(row)) for row in rows]


async def rename_world(
    db: aiosqlite.Connection, world_id: str, name: str, ts: str
) -> bool:
    cursor = await db.execute(
        "UPDATE worlds SET name = :name, updated_at = :ts WHERE world_id = :world_id",
        {"name": name, "ts": ts, "world_id": world_id},
    )
    await db.commit()
    return cursor.rowcount > 0


async def delete_world(db: aiosqlite.Connection, world_id: str) -> bool:
    cursor = await db.execute(
        "DELETE FROM worlds WHERE world_id = :world_id",
        {"world_id": world_id},
    )
    await db.commit()
    return cursor.rowcount > 0
