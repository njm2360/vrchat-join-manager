import aiosqlite
from datetime import datetime
from fastapi import APIRouter, Depends, HTTPException, Query

from db import get_db
from models.common import EventOut
from models.locations import SessionOut
from models.locations import (
    CloseLocationRequest,
    PotentialSession,
    InstanceOut,
    LocationPlayerOut,
    PlayerListOut,
    RestoreRequest,
    TimelinePoint,
)
from repositories import locations as repo
from repositories import instances as instances_repo
from utils import to_utc_str

router = APIRouter(prefix="/api", tags=["instances"])


@router.post("/locations/{location_id:path}/close", status_code=204)
async def close_location_by_location_id(
    location_id: str,
    body: CloseLocationRequest,
    db: aiosqlite.Connection = Depends(get_db),
) -> None:
    instance_id = await instances_repo.get_open_instance_id(db, location_id)
    if instance_id is not None:
        ts = to_utc_str(body.at)
        await instances_repo.close_location_sessions(db, instance_id, ts, body.user_id)
        await db.commit()


@router.get(
    "/locations/{location_id:path}/potential-sessions",
    response_model=list[PotentialSession],
)
async def get_potential_sessions(
    location_id: str,
    db: aiosqlite.Connection = Depends(get_db),
) -> list[PotentialSession]:
    rows = await instances_repo.get_potential_sessions(db, location_id)
    return [PotentialSession(user_id=r[0], internal_id=r[1]) for r in rows]


@router.post("/locations/{location_id:path}/resume", status_code=204)
async def resume_instance(
    location_id: str,
    body: RestoreRequest,
    db: aiosqlite.Connection = Depends(get_db),
) -> None:
    await instances_repo.resume_instance(db, location_id, body.user_ids)
    await db.commit()


_INSTANCE_SORT_COLS = {"opened_at", "closed_at"}


@router.get("/instances", response_model=list[InstanceOut])
async def get_instances(
    start: datetime | None = Query(None, description="opened_at がこの時刻以降"),
    end: datetime | None = Query(None, description="opened_at がこの時刻以前"),
    is_open: bool | None = Query(
        None, description="true=進行中のみ / false=終了済みのみ"
    ),
    world_id: str | None = Query(None, description="ワールドID完全一致"),
    group_id: str | None = Query(None, description="グループID完全一致"),
    region: str | None = Query(None, description="リージョン完全一致"),
    sort_by: str = Query(
        default="opened_at", description="ソートカラム: opened_at / closed_at"
    ),
    order: str = Query(default="desc", pattern="^(asc|desc)$"),
    limit: int | None = Query(default=None, ge=1),
    offset: int = Query(default=0, ge=0),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[InstanceOut]:
    if sort_by not in _INSTANCE_SORT_COLS:
        sort_by = "opened_at"
    return await repo.get_instances(
        db,
        start,
        end,
        is_open,
        world_id,
        group_id,
        region,
        sort_by,
        order,
        limit,
        offset,
    )


@router.delete("/instances/{instance_id}", status_code=204)
async def delete_instance(
    instance_id: int,
    db: aiosqlite.Connection = Depends(get_db),
) -> None:
    deleted = await instances_repo.delete_instance(db, instance_id)
    if not deleted:
        raise HTTPException(status_code=404, detail="instance not found")
    await db.commit()


@router.get("/instances/{instance_id}", response_model=InstanceOut)
async def get_instance(
    instance_id: int,
    db: aiosqlite.Connection = Depends(get_db),
) -> InstanceOut:
    inst = await repo.get_instance(db, instance_id)
    if inst is None:
        raise HTTPException(status_code=404, detail="instance not found")
    return inst


@router.get("/instances/{instance_id}/presence", response_model=list[SessionOut])
async def get_presence(
    instance_id: int,
    at: datetime = Query(..., description="この時刻に在席していたプレイヤーを返す"),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[SessionOut]:
    return await repo.get_presence(db, instance_id, at)


_PLAYER_SORT_COLS = {"internal_id", "display_name", "join_ts"}


@router.get("/instances/{instance_id}/players", response_model=list[LocationPlayerOut])
async def get_location_players(
    instance_id: int,
    sort_by: str = Query(default="internal_id"),
    order: str = Query(default="asc", pattern="^(asc|desc)$"),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[LocationPlayerOut]:
    if sort_by not in _PLAYER_SORT_COLS:
        sort_by = "internal_id"
    return await repo.get_location_players(db, instance_id, sort_by, order)


_VISITOR_SORT_COLS = {
    "display_name",
    "first_seen",
    "last_seen",
    "join_count",
    "total_duration_seconds",
}


@router.get("/instances/{instance_id}/visitors", response_model=list[PlayerListOut])
async def get_location_visitors(
    instance_id: int,
    sort_by: str = Query(default="last_seen"),
    order: str = Query(default="desc", pattern="^(asc|desc)$"),
    limit: int | None = Query(default=None, ge=1),
    offset: int = Query(default=0, ge=0),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[PlayerListOut]:
    if sort_by not in _VISITOR_SORT_COLS:
        sort_by = "last_seen"
    return await repo.get_location_visitors(
        db, instance_id, sort_by, order, limit, offset
    )


@router.get(
    "/instances/{instance_id}/presence-timeline",
    response_model=list[TimelinePoint],
)
async def get_presence_timeline(
    instance_id: int,
    start: datetime | None = Query(None),
    end: datetime | None = Query(None),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[TimelinePoint]:
    return await repo.get_presence_timeline(db, instance_id, start, end)


@router.get("/instances/{instance_id}/events", response_model=list[EventOut])
async def get_location_events(
    instance_id: int,
    start: datetime | None = Query(default=None),
    end: datetime | None = Query(default=None),
    order: str = Query(default="desc", pattern="^(asc|desc)$"),
    limit: int | None = Query(default=None, ge=1),
    offset: int = Query(default=0, ge=0),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[EventOut]:
    return await repo.get_location_events(
        db, instance_id, start, end, order, limit, offset
    )


_SESSION_SORT_COLS = {"display_name", "join_ts", "leave_ts", "duration_seconds"}


@router.get("/instances/{instance_id}/sessions", response_model=list[SessionOut])
async def get_location_sessions(
    instance_id: int,
    start: datetime | None = Query(default=None),
    end: datetime | None = Query(default=None),
    sort_by: str = Query(default="join_ts"),
    order: str = Query(default="asc", pattern="^(asc|desc)$"),
    limit: int | None = Query(default=None, ge=1),
    offset: int = Query(default=0, ge=0),
    db: aiosqlite.Connection = Depends(get_db),
) -> list[SessionOut]:
    if sort_by not in _SESSION_SORT_COLS:
        sort_by = "join_ts"
    return await repo.get_location_sessions(
        db, instance_id, start, end, sort_by, order, limit, offset
    )
