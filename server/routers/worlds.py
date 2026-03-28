from datetime import datetime

import aiosqlite
from fastapi import APIRouter, Depends, Query

from db import get_db
from models import SessionOut, WorldOut
from repositories import worlds as repo

router = APIRouter()


@router.get("/api/worlds", response_model=list[WorldOut])
async def get_worlds(
    start: datetime | None = Query(None, description="first_seen がこの時刻以降"),
    end: datetime | None = Query(None, description="last_seen がこの時刻以前"),
    order: str = Query(default="desc", pattern="^(asc|desc)$"),
    limit: int | None = Query(default=None, ge=1),
    offset: int = Query(default=0, ge=0),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[WorldOut]:
    return await repo.get_worlds(db, start, end, order, limit, offset)


@router.get("/api/worlds/{world_id}/sessions", response_model=list[SessionOut])
async def get_world_sessions(
    world_id: str,
    start: datetime | None = Query(None),
    end: datetime | None = Query(None),
    order: str = Query(default="asc", pattern="^(asc|desc)$"),
    limit: int | None = Query(default=None, ge=1),
    offset: int = Query(default=0, ge=0),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[SessionOut]:
    return await repo.get_world_sessions(db, world_id, start, end, order, limit, offset)
