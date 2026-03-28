from datetime import datetime

import aiosqlite

from models import SessionOut


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
    limit_clause = f"LIMIT {limit} OFFSET {offset}" if limit is not None else f"LIMIT -1 OFFSET {offset}"
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
