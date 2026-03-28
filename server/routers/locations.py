from datetime import datetime

import aiosqlite
from fastapi import APIRouter, Depends, Query

from db import get_db
from models import EventOut, LocationOut, PlayerOut, SessionOut, TimelinePoint
from repositories import locations as repo

router = APIRouter()


@router.get("/api/locations", response_model=list[LocationOut])
async def get_locations(
    start: datetime | None = Query(None, description="first_seen がこの時刻以降"),
    end: datetime | None = Query(None, description="last_seen がこの時刻以前"),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[LocationOut]:
    return await repo.get_locations(db, start, end)


@router.get("/api/locations/{location_id:path}/presence", response_model=list[SessionOut])
async def get_presence(
    location_id: str,
    at: datetime = Query(..., description="この時刻に在席していたプレイヤーを返す"),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[SessionOut]:
    return await repo.get_presence(db, location_id, at)


@router.get("/api/locations/{location_id:path}/players", response_model=list[PlayerOut])
async def get_location_players(
    location_id: str,
    db: aiosqlite.Connection = Depends(get_db),
) -> list[PlayerOut]:
    return await repo.get_location_players(db, location_id)


@router.get(
    "/api/locations/{location_id:path}/presence-timeline",
    response_model=list[TimelinePoint],
)
async def get_presence_timeline(
    location_id: str,
    start: datetime | None = Query(None),
    end: datetime | None = Query(None),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[TimelinePoint]:
    return await repo.get_presence_timeline(db, location_id, start, end)


@router.get("/api/locations/{location_id:path}/events", response_model=list[EventOut])
async def get_location_events(
    location_id: str,
    start: datetime | None = Query(default=None),
    end: datetime | None = Query(default=None),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[EventOut]:
    return await repo.get_location_events(db, location_id, start, end)


@router.get("/api/locations/{location_id:path}/sessions", response_model=list[SessionOut])
async def get_location_sessions(
    location_id: str,
    start: datetime | None = Query(default=None),
    end: datetime | None = Query(default=None),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[SessionOut]:
    return await repo.get_location_sessions(db, location_id, start, end)
