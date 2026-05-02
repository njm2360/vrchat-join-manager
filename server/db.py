from collections.abc import AsyncGenerator
from pathlib import Path

import aiosqlite

DB_PATH = Path(__file__).parent / "data" / "vrchat.db"

_DDL = """
CREATE TABLE IF NOT EXISTS groups (
    group_id   TEXT PRIMARY KEY,
    name       TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS worlds (
    world_id   TEXT PRIMARY KEY,
    name       TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS players (
    user_id      TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS instances (
    id                INTEGER PRIMARY KEY,
    location_id       TEXT    NOT NULL,
    world_id          TEXT    NOT NULL,
    instance_id       TEXT,
    group_id          TEXT,
    group_access_type TEXT,
    region            TEXT,
    friends           TEXT,
    hidden            TEXT,
    private           TEXT,
    opened_at         TEXT    NOT NULL,
    closed_at         TEXT
);
CREATE INDEX IF NOT EXISTS idx_instances_location ON instances(location_id);
CREATE INDEX IF NOT EXISTS idx_instances_group    ON instances(group_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_instances_open ON instances(location_id) WHERE closed_at IS NULL;

CREATE TABLE IF NOT EXISTS events (
    id           INTEGER PRIMARY KEY,
    event_type   TEXT    NOT NULL CHECK(event_type IN ('join', 'leave')),
    instance_id  INTEGER NOT NULL REFERENCES instances(id),
    world_id     TEXT    NOT NULL,
    user_id      TEXT    NOT NULL REFERENCES players(user_id),
    location_id  TEXT    NOT NULL,
    timestamp    TEXT    NOT NULL,
    UNIQUE(user_id, event_type, location_id, timestamp)
);
CREATE INDEX IF NOT EXISTS idx_events_instance_time ON events(instance_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_events_user_time     ON events(user_id, timestamp);

CREATE TABLE IF NOT EXISTS player_discord (
    user_id    TEXT PRIMARY KEY,
    discord_id TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER IF NOT EXISTS trg_player_discord_updated
AFTER UPDATE OF discord_id ON player_discord
BEGIN
    UPDATE player_discord SET updated_at = CURRENT_TIMESTAMP WHERE user_id = NEW.user_id;
END;

CREATE TABLE IF NOT EXISTS sessions (
    id                    INTEGER PRIMARY KEY,
    instance_id           INTEGER NOT NULL REFERENCES instances(id),
    world_id              TEXT    NOT NULL,
    user_id               TEXT    NOT NULL REFERENCES players(user_id),
    internal_id           INTEGER,
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
CREATE UNIQUE INDEX IF NOT EXISTS uq_sessions_open    ON sessions(user_id, instance_id) WHERE leave_ts IS NULL;
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
