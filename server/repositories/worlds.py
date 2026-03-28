from datetime import datetime

import aiosqlite

from models import SessionOut


async def get_world_sessions(
    db: aiosqlite.Connection,
    world_id: str,
    start: datetime | None,
    end: datetime | None,
    order: str = "asc",
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
    cursor = await db.execute(
        f"""
        SELECT id, location_id, user_id, display_name, join_ts, leave_ts,
               duration_seconds
        FROM sessions
        WHERE {where}
        ORDER BY join_ts {order.upper()}
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [SessionOut(**dict(row)) for row in rows]
