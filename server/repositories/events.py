import aiosqlite

from models import PlayerEvent
from utils import parse_location_id


async def upsert_player(db: aiosqlite.Connection, user_id: str, name: str, ts: str) -> None:
    await db.execute(
        """
        INSERT INTO players(user_id, display_name, updated_at)
        VALUES(:user_id, :name, :ts)
        ON CONFLICT(user_id) DO UPDATE
            SET display_name = excluded.display_name,
                updated_at   = excluded.updated_at
        """,
        {"user_id": user_id, "name": name, "ts": ts},
    )


async def insert_event(db: aiosqlite.Connection, body: PlayerEvent, ts: str) -> int:
    loc = parse_location_id(body.location_id)
    cursor = await db.execute(
        """
        INSERT INTO events(
            event_type, location_id, world_id,
            user_id, display_name, internal_id, timestamp
        )
        VALUES(
            :event_type, :location_id, :world_id,
            :user_id, :name, :internal_id, :ts
        )
        """,
        {
            "event_type": body.event,
            "location_id": body.location_id,
            **loc,
            "user_id": body.user_id,
            "name": body.name,
            "internal_id": body.internal_id,
            "ts": ts,
        },
    )
    return cursor.lastrowid


async def open_session(
    db: aiosqlite.Connection, body: PlayerEvent, event_id: int, ts: str
) -> None:
    loc = parse_location_id(body.location_id)
    # 同ロケーションにオープンセッションが既にあれば重複とみなしてスキップ
    cur = await db.execute(
        """
        SELECT 1 FROM sessions
        WHERE user_id = :user_id AND location_id = :location_id AND leave_ts IS NULL
        LIMIT 1
        """,
        {"user_id": body.user_id, "location_id": body.location_id},
    )
    if await cur.fetchone() is None:
        await db.execute(
            """
            INSERT INTO sessions(
                location_id, world_id,
                user_id, display_name, join_event_id, join_ts
            )
            VALUES(
                :location_id, :world_id,
                :user_id, :name, :join_event_id, :join_ts
            )
            """,
            {
                "location_id": body.location_id,
                **loc,
                "user_id": body.user_id,
                "name": body.name,
                "join_event_id": event_id,
                "join_ts": ts,
            },
        )


async def close_session(
    db: aiosqlite.Connection, user_id: str, location_id: str, event_id: int, ts: str
) -> None:
    # 同ロケーションの最新オープンセッションを閉じる
    await db.execute(
        """
        UPDATE sessions
        SET leave_event_id   = :event_id,
            leave_ts         = :ts,
            duration_seconds = CAST(ROUND((julianday(:ts) - julianday(join_ts)) * 86400) AS INTEGER)
        WHERE id = (
            SELECT id FROM sessions
            WHERE user_id     = :user_id
              AND location_id = :location_id
              AND leave_ts IS NULL
            ORDER BY join_ts DESC
            LIMIT 1
        )
        """,
        {
            "event_id": event_id,
            "ts": ts,
            "user_id": user_id,
            "location_id": location_id,
        },
    )
