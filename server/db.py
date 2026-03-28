from collections.abc import AsyncGenerator
from pathlib import Path

import aiosqlite

DB_PATH = Path(__file__).parent / "data" / "vrchat.db"

_DDL = """
CREATE TABLE IF NOT EXISTS players (
    user_id      TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
    id           INTEGER PRIMARY KEY,
    event_type   TEXT    NOT NULL CHECK(event_type IN ('join', 'leave')),
    location_id  TEXT    NOT NULL,
    world_id     TEXT    NOT NULL,
    user_id      TEXT    NOT NULL REFERENCES players(user_id),
    display_name TEXT    NOT NULL,
    internal_id  INTEGER,
    timestamp    TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_location_time ON events(location_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_events_user_time     ON events(user_id, timestamp);

CREATE TABLE IF NOT EXISTS sessions (
    id               INTEGER PRIMARY KEY,
    location_id      TEXT    NOT NULL,
    world_id         TEXT    NOT NULL,
    user_id          TEXT    NOT NULL REFERENCES players(user_id),
    display_name     TEXT    NOT NULL,
    join_event_id    INTEGER NOT NULL REFERENCES events(id),
    leave_event_id   INTEGER          REFERENCES events(id),
    join_ts          TEXT    NOT NULL,
    leave_ts         TEXT,
    duration_seconds INTEGER
);
CREATE INDEX IF NOT EXISTS idx_sessions_location_time ON sessions(location_id, join_ts, leave_ts);
CREATE INDEX IF NOT EXISTS idx_sessions_user_time     ON sessions(user_id, join_ts);
CREATE INDEX IF NOT EXISTS idx_sessions_world_time    ON sessions(world_id, join_ts);
"""


async def init_db() -> None:
    DB_PATH.parent.mkdir(parents=True, exist_ok=True)
    async with aiosqlite.connect(DB_PATH) as db:
        await db.executescript(_DDL)
        await db.commit()


async def get_db() -> AsyncGenerator[aiosqlite.Connection, None]:
    async with aiosqlite.connect(DB_PATH) as db:
        db.row_factory = aiosqlite.Row
        yield db
