from datetime import datetime

import aiosqlite
from fastapi import APIRouter, Depends, Query

from db import get_db
from models.analytics import DailyActiveUsersPoint, HourlyActiveUsersPoint, JoinViolationRankOut, PlayerRankOut
from repositories import analytics as repo

router = APIRouter(prefix="/api/analytics", tags=["analytics"])


@router.get("/daily-active-users", response_model=list[DailyActiveUsersPoint])
async def get_daily_active_users(
    world_id: str | None = Query(None),
    group_id: str | None = Query(None),
    start: datetime | None = Query(None),
    end: datetime | None = Query(None),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[DailyActiveUsersPoint]:
    return await repo.get_daily_active_users(db, world_id, group_id, start, end)


@router.get("/hourly-active-users", response_model=list[HourlyActiveUsersPoint])
async def get_hourly_active_users(
    world_id: str | None = Query(None),
    group_id: str | None = Query(None),
    start: datetime | None = Query(None),
    end: datetime | None = Query(None),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[HourlyActiveUsersPoint]:
    return await repo.get_hourly_active_users(db, world_id, group_id, start, end)


@router.get("/join-violation-rankings", response_model=list[JoinViolationRankOut])
async def get_join_violation_rankings(
    group_id: str = Query(...),
    start: datetime | None = Query(None),
    end: datetime | None = Query(None),
    order: str = Query(default="desc", pattern="^(asc|desc)$"),
    limit: int | None = Query(default=None, ge=1),
    offset: int = Query(default=0, ge=0),
    allow_diff: int = Query(default=0, ge=0),
    min_duration: int | None = Query(default=None, ge=0),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[JoinViolationRankOut]:
    return await repo.get_join_violation_rankings(db, group_id, start, end, order, limit, offset, allow_diff, min_duration)


@router.get("/player-rankings", response_model=list[PlayerRankOut])
async def get_player_rankings(
    world_id: str | None = Query(None),
    group_id: str | None = Query(None),
    order: str = Query(default="desc", pattern="^(asc|desc)$"),
    limit: int | None = Query(default=None, ge=1),
    offset: int = Query(default=0, ge=0),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[PlayerRankOut]:
    return await repo.get_player_rankings(db, world_id, group_id, order, limit, offset)
