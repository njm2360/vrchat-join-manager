from datetime import datetime

import aiosqlite
from fastapi import APIRouter, Depends, HTTPException, Query

from db import get_db
from models.groups import GroupOut, GroupRenameIn
from repositories import groups as repo
from utils import to_utc_str

router = APIRouter(prefix="/api", tags=["groups"])


@router.get("/groups", response_model=list[GroupOut])
async def get_groups(
    order: str = Query(default="desc", pattern="^(asc|desc)$"),
    limit: int | None = Query(default=None, ge=1),
    offset: int = Query(default=0, ge=0),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[GroupOut]:
    return await repo.get_groups(db, order, limit, offset)


@router.patch("/groups/{group_id}", status_code=204)
async def rename_group(
    group_id: str,
    body: GroupRenameIn,
    db: aiosqlite.Connection = Depends(get_db),
) -> None:
    ts = to_utc_str(datetime.now())
    if not await repo.rename_group(db, group_id, body.name, ts):
        raise HTTPException(status_code=404)


@router.delete("/groups/{group_id}", status_code=204)
async def delete_group(
    group_id: str,
    db: aiosqlite.Connection = Depends(get_db),
) -> None:
    if not await repo.delete_group(db, group_id):
        raise HTTPException(status_code=404)
