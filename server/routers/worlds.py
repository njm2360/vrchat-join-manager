from datetime import datetime

import aiosqlite
from fastapi import APIRouter, Depends, Query

from db import get_db
from models import SessionOut
from repositories import worlds as repo

router = APIRouter()


@router.get("/api/worlds/{world_id}/sessions", response_model=list[SessionOut])
async def get_world_sessions(
    world_id: str,
    start: datetime | None = Query(None),
    end: datetime | None = Query(None),
    order: str = Query(default="asc", pattern="^(asc|desc)$"),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[SessionOut]:
    return await repo.get_world_sessions(db, world_id, start, end, order)
