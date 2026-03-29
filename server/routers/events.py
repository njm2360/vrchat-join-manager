import aiosqlite
from fastapi import APIRouter, Depends

from db import get_db
from models import PlayerEvent
from repositories import events as repo
from repositories import instances as instances_repo
from utils import parse_location_id, to_utc_str

router = APIRouter(prefix="/api", tags=["events"])


@router.post("/events")
async def receive_event(
    body: PlayerEvent,
    db: aiosqlite.Connection = Depends(get_db),
) -> None:
    ts = to_utc_str(body.timestamp)
    loc = parse_location_id(body.location_id)

    await repo.upsert_player(db, body.user_id, body.name, ts)

    if body.event == "join":
        instance_id = await instances_repo.get_or_create_instance(
            db, body.location_id, loc["world_id"], ts
        )
        event_id = await repo.insert_event(db, body, instance_id, ts)
        if event_id is not None:
            await repo.open_session(db, body, instance_id, event_id, ts)

    elif body.event == "leave":
        instance_id = await instances_repo.get_open_instance_id(db, body.location_id)
        if instance_id is not None:
            event_id = await repo.insert_event(db, body, instance_id, ts)
            if event_id is not None:
                await repo.close_session(db, body.user_id, instance_id, event_id, ts)

    await db.commit()
