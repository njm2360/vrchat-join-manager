import aiosqlite

from models.events import PlayerEvent
from utils import parse_location_id


async def upsert_group(db: aiosqlite.Connection, group_id: str, ts: str) -> None:
    await db.execute(
        """
        INSERT INTO groups(group_id, created_at, updated_at)
        VALUES(:group_id, :ts, :ts)
        ON CONFLICT(group_id) DO NOTHING
        """,
        {"group_id": group_id, "ts": ts},
    )


async def upsert_world(db: aiosqlite.Connection, world_id: str, ts: str) -> None:
    await db.execute(
        """
        INSERT INTO worlds(world_id, created_at, updated_at)
        VALUES(:world_id, :ts, :ts)
        ON CONFLICT(world_id) DO NOTHING
        """,
        {"world_id": world_id, "ts": ts},
    )


async def upsert_player(
    db: aiosqlite.Connection, user_id: str, name: str, ts: str
) -> None:
    await db.execute(
        """
        INSERT INTO players(user_id, display_name, created_at, updated_at)
        VALUES(:user_id, :name, :ts, :ts)
        ON CONFLICT(user_id) DO UPDATE
            SET display_name = excluded.display_name,
                updated_at   = excluded.updated_at
        """,
        {"user_id": user_id, "name": name, "ts": ts},
    )


async def insert_event(
    db: aiosqlite.Connection, body: PlayerEvent, instance_id: int, ts: str
) -> int | None:
    loc = parse_location_id(body.location_id)
    cursor = await db.execute(
        """
        INSERT OR IGNORE INTO events(
            event_type, instance_id, world_id,
            user_id, timestamp
        )
        VALUES(
            :event_type, :instance_id, :world_id,
            :user_id, :ts
        )
        """,
        {
            "event_type": body.event,
            "instance_id": instance_id,
            "world_id": loc.world_id,
            "user_id": body.user_id,
            "ts": ts,
        },
    )
    return cursor.lastrowid if cursor.rowcount else None


async def open_session(
    db: aiosqlite.Connection,
    body: PlayerEvent,
    instance_id: int,
    event_id: int,
    ts: str,
) -> None:
    loc = parse_location_id(body.location_id)
    # 同インスタンスにオープンセッションが既にあれば重複とみなしてスキップ
    cur = await db.execute(
        """
        SELECT 1 FROM sessions
        WHERE user_id = :user_id AND instance_id = :instance_id AND leave_ts IS NULL
        LIMIT 1
        """,
        {"user_id": body.user_id, "instance_id": instance_id},
    )
    if await cur.fetchone() is None:
        await db.execute(
            """
            INSERT INTO sessions(
                instance_id, world_id,
                user_id, internal_id, join_event_id, join_ts
            )
            VALUES(
                :instance_id, :world_id,
                :user_id, :internal_id, :join_event_id, :join_ts
            )
            """,
            {
                "instance_id": instance_id,
                "world_id": loc.world_id,
                "user_id": body.user_id,
                "internal_id": body.internal_id,
                "join_event_id": event_id,
                "join_ts": ts,
            },
        )


async def close_session(
    db: aiosqlite.Connection, user_id: str, instance_id: int, event_id: int, ts: str
) -> None:
    # 同インスタンスの最新オープンセッションを閉じる
    await db.execute(
        """
        UPDATE sessions
        SET leave_event_id   = :event_id,
            leave_ts         = :ts,
            duration_seconds = CAST(ROUND((julianday(:ts) - julianday(join_ts)) * 86400) AS INTEGER)
        WHERE id = (
            SELECT id FROM sessions
            WHERE user_id     = :user_id
              AND instance_id = :instance_id
              AND leave_ts IS NULL
            ORDER BY join_ts DESC
            LIMIT 1
        )
        """,
        {
            "event_id": event_id,
            "ts": ts,
            "user_id": user_id,
            "instance_id": instance_id,
        },
    )
