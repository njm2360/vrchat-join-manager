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

CREATE TABLE IF NOT EXISTS instances (
    id          INTEGER PRIMARY KEY,
    location_id TEXT    NOT NULL,
    world_id    TEXT    NOT NULL,
    opened_at   TEXT    NOT NULL,
    closed_at   TEXT
);
CREATE INDEX IF NOT EXISTS idx_instances_location ON instances(location_id);

CREATE TABLE IF NOT EXISTS events (
    id           INTEGER PRIMARY KEY,
    event_type   TEXT    NOT NULL CHECK(event_type IN ('join', 'leave')),
    instance_id  INTEGER NOT NULL REFERENCES instances(id),
    world_id     TEXT    NOT NULL,
    user_id      TEXT    NOT NULL REFERENCES players(user_id),
    display_name TEXT    NOT NULL,
    internal_id  INTEGER,
    timestamp    TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_instance_time ON events(instance_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_events_user_time     ON events(user_id, timestamp);

CREATE TABLE IF NOT EXISTS sessions (
    id                    INTEGER PRIMARY KEY,
    instance_id           INTEGER NOT NULL REFERENCES instances(id),
    world_id              TEXT    NOT NULL,
    user_id               TEXT    NOT NULL REFERENCES players(user_id),
    display_name          TEXT    NOT NULL,
    join_event_id         INTEGER NOT NULL REFERENCES events(id),
    leave_event_id        INTEGER          REFERENCES events(id),
    join_ts               TEXT    NOT NULL,
    leave_ts              TEXT,
    duration_seconds      INTEGER,
    is_estimated_leave    INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_sessions_instance_time ON sessions(instance_id, join_ts, leave_ts);
CREATE INDEX IF NOT EXISTS idx_sessions_user_time     ON sessions(user_id, join_ts);
CREATE INDEX IF NOT EXISTS idx_sessions_world_time    ON sessions(world_id, join_ts);
"""


_db: aiosqlite.Connection | None = None


async def init_db() -> None:
    global _db
    DB_PATH.parent.mkdir(parents=True, exist_ok=True)
    _db = await aiosqlite.connect(DB_PATH)
    _db.row_factory = aiosqlite.Row
    await _db.execute("PRAGMA journal_mode=WAL")
    await _db.execute("PRAGMA synchronous=NORMAL")
    await _db.executescript(_DDL)
    await _db.commit()


async def close_db() -> None:
    global _db
    if _db is not None:
        await _db.close()
        _db = None


async def get_db() -> AsyncGenerator[aiosqlite.Connection, None]:
    assert _db is not None, "DB not initialized"
    yield _db
