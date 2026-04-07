import aiosqlite


async def get_or_create_instance(
    db: aiosqlite.Connection,
    location_id: str,
    world_id: str,
    instance_id: str | None,
    group_id: str | None,
    group_access_type: str | None,
    region: str | None,
    friends: str | None,
    hidden: str | None,
    private: str | None,
    ts: str,
) -> int:
    """joinイベント用: オープン中のインスタンスがあればそれを返し、なければ新規作成する。"""
    cur = await db.execute(
        """INSERT OR IGNORE INTO instances
               (location_id, world_id, instance_id, group_id, group_access_type, region, friends, hidden, private, opened_at)
           VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
           RETURNING id""",
        (
            location_id,
            world_id,
            instance_id,
            group_id,
            group_access_type,
            region,
            friends,
            hidden,
            private,
            ts,
        ),
    )
    row = await cur.fetchone()
    if row:
        return row[0]
    cur = await db.execute(
        "SELECT id FROM instances WHERE location_id = ? AND closed_at IS NULL",
        (location_id,),
    )
    row = await cur.fetchone()
    return row[0]


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


async def get_potential_sessions(
    db: aiosqlite.Connection,
    location_id: str,
) -> list[tuple[str, int]]:
    """直近のclosedインスタンスのis_estimated_leave=1セッションを返す。"""
    cur = await db.execute(
        """SELECT id FROM instances
           WHERE location_id = ? AND closed_at IS NOT NULL
           ORDER BY closed_at DESC LIMIT 1""",
        (location_id,),
    )
    row = await cur.fetchone()
    if not row:
        return []
    cur = await db.execute(
        "SELECT user_id, internal_id FROM sessions WHERE instance_id = ? AND is_estimated_leave = 1",
        (row[0],),
    )
    return await cur.fetchall()


async def resume_instance(
    db: aiosqlite.Connection,
    location_id: str,
    user_ids: list[str],
) -> None:
    if not user_ids:
        return
    cur = await db.execute(
        "SELECT id FROM instances WHERE location_id = ? AND closed_at IS NULL",
        (location_id,),
    )
    if await cur.fetchone():
        return
    cur = await db.execute(
        """SELECT id FROM instances
           WHERE location_id = ? AND closed_at IS NOT NULL
           ORDER BY closed_at DESC LIMIT 1""",
        (location_id,),
    )
    row = await cur.fetchone()
    if not row:
        return
    instance_id = row[0]
    await db.execute(
        "UPDATE instances SET closed_at = NULL WHERE id = ?",
        (instance_id,),
    )
    for user_id in user_ids:
        await db.execute(
            """UPDATE sessions
               SET leave_ts           = NULL,
                   duration_seconds   = NULL,
                   is_estimated_leave = 0
               WHERE user_id     = ?
                 AND instance_id = ?
                 AND is_estimated_leave = 1""",
            (user_id, instance_id),
        )


async def close_location_sessions(
    db: aiosqlite.Connection, instance_id: int, ts: str, self_user_id: str | None = None
) -> int:
    cursor = await db.execute(
        """
        UPDATE sessions
        SET leave_ts           = :ts,
            duration_seconds   = CAST(ROUND((julianday(:ts) - julianday(join_ts)) * 86400) AS INTEGER),
            is_estimated_leave = CASE WHEN user_id = :self_user_id THEN 0 ELSE 1 END
        WHERE instance_id = :instance_id
          AND leave_ts IS NULL
        """,
        {"ts": ts, "instance_id": instance_id, "self_user_id": self_user_id},
    )
    await db.execute(
        "UPDATE instances SET closed_at = :ts WHERE id = :instance_id",
        {"ts": ts, "instance_id": instance_id},
    )
    return cursor.rowcount
