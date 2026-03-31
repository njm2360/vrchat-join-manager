from datetime import datetime

import aiosqlite
from fastapi import APIRouter, Depends, HTTPException, Query

from db import get_db
from models.worlds import WorldOut, WorldRenameIn
from repositories import worlds as repo
from utils import to_utc_str

router = APIRouter(prefix="/api", tags=["worlds"])


@router.get("/worlds", response_model=list[WorldOut])
async def get_worlds(
    start: datetime | None = Query(None, description="first_seen がこの時刻以降"),
    end: datetime | None = Query(None, description="last_seen がこの時刻以前"),
    order: str = Query(default="desc", pattern="^(asc|desc)$"),
    limit: int | None = Query(default=None, ge=1),
    offset: int = Query(default=0, ge=0),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[WorldOut]:
    return await repo.get_worlds(db, start, end, order, limit, offset)


@router.patch("/worlds/{world_id}", status_code=204)
async def rename_world(
    world_id: str,
    body: WorldRenameIn,
    db: aiosqlite.Connection = Depends(get_db),
) -> None:
    ts = to_utc_str(datetime.now())
    if not await repo.rename_world(db, world_id, body.name, ts):
        raise HTTPException(status_code=404)


@router.delete("/worlds/{world_id}", status_code=204)
async def delete_world(
    world_id: str,
    db: aiosqlite.Connection = Depends(get_db),
) -> None:
    if not await repo.delete_world(db, world_id):
        raise HTTPException(status_code=404)
