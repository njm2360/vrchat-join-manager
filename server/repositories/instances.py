import aiosqlite


async def get_or_create_instance(
    db: aiosqlite.Connection, location_id: str, world_id: str, ts: str
) -> int:
    """joinイベント用: オープン中のインスタンスがあればそれを返し、なければ新規作成する。"""
    cur = await db.execute(
        "SELECT id FROM instances WHERE location_id = ? AND closed_at IS NULL",
        (location_id,),
    )
    row = await cur.fetchone()
    if row:
        return row[0]
    cur = await db.execute(
        "INSERT INTO instances(location_id, world_id, opened_at) VALUES(?, ?, ?)",
        (location_id, world_id, ts),
    )
    return cur.lastrowid


async def get_open_instance_id(
    db: aiosqlite.Connection, location_id: str
) -> int | None:
    """leaveイベント用: オープン中のインスタンスIDを返す。なければNone。"""
    cur = await db.execute(
        "SELECT id FROM instances WHERE location_id = ? AND closed_at IS NULL",
        (location_id,),
    )
    row = await cur.fetchone()
    return row[0] if row else None


async def close_location_sessions(
    db: aiosqlite.Connection, instance_id: int, ts: str
) -> int:
    cursor = await db.execute(
        """
        UPDATE sessions
        SET leave_ts           = :ts,
            duration_seconds   = CAST(ROUND((julianday(:ts) - julianday(join_ts)) * 86400) AS INTEGER),
            is_estimated_leave = 1
        WHERE instance_id = :instance_id
          AND leave_ts IS NULL
        """,
        {"ts": ts, "instance_id": instance_id},
    )
    await db.execute(
        "UPDATE instances SET closed_at = :ts WHERE id = :instance_id",
        {"ts": ts, "instance_id": instance_id},
    )
    return cursor.rowcount
