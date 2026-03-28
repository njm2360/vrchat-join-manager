from datetime import datetime

import aiosqlite
from fastapi import APIRouter, Depends, Query

from db import get_db
from models import EventOut, PlayerSessionOut
from repositories import players as repo

router = APIRouter()


@router.get("/api/players/{user_id}/events", response_model=list[EventOut])
async def get_player_events(
    user_id: str,
    location_id: str | None = Query(None),
    start: datetime | None = Query(None),
    end: datetime | None = Query(None),
    order: str = Query(default="asc", pattern="^(asc|desc)$"),
    limit: int | None = Query(default=None, ge=1),
    offset: int = Query(default=0, ge=0),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[EventOut]:
    return await repo.get_player_events(db, user_id, location_id, start, end, order, limit, offset)


@router.get("/api/players/{user_id}/sessions", response_model=list[PlayerSessionOut])
async def get_player_sessions(
    user_id: str,
    location_id: str | None = Query(None),
    start: datetime | None = Query(None),
    end: datetime | None = Query(None),
    order: str = Query(default="asc", pattern="^(asc|desc)$"),
    limit: int | None = Query(default=None, ge=1),
    offset: int = Query(default=0, ge=0),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[PlayerSessionOut]:
    return await repo.get_player_sessions(db, user_id, location_id, start, end, order, limit, offset)
