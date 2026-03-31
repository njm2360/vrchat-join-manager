import aiosqlite

from models.groups import GroupOut


async def get_groups(
    db: aiosqlite.Connection,
    order: str = "desc",
    limit: int | None = None,
    offset: int = 0,
) -> list[GroupOut]:
    limit_clause = (
        f"LIMIT {limit} OFFSET {offset}"
        if limit is not None
        else f"LIMIT -1 OFFSET {offset}"
    )
    cursor = await db.execute(
        f"""
        SELECT group_id, name, created_at, updated_at
        FROM groups
        ORDER BY created_at {order.upper()}
        {limit_clause}
        """,
    )
    rows = await cursor.fetchall()
    return [GroupOut(**dict(row)) for row in rows]


async def rename_group(
    db: aiosqlite.Connection, group_id: str, name: str, ts: str
) -> bool:
    cursor = await db.execute(
        "UPDATE groups SET name = :name, updated_at = :ts WHERE group_id = :group_id",
        {"name": name, "ts": ts, "group_id": group_id},
    )
    await db.commit()
    return cursor.rowcount > 0


async def delete_group(
    db: aiosqlite.Connection, group_id: str
) -> bool:
    cursor = await db.execute(
        "DELETE FROM groups WHERE group_id = :group_id",
        {"group_id": group_id},
    )
    await db.commit()
    return cursor.rowcount > 0
