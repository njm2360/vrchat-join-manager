import aiosqlite
from fastapi import APIRouter, Depends

from db import get_db
from models import PlayerEvent
from repositories import events as repo
from utils import to_utc_str

router = APIRouter()


@router.post("/api/events")
async def receive_event(
    body: PlayerEvent,
    db: aiosqlite.Connection = Depends(get_db),
) -> None:
    ts = to_utc_str(body.timestamp)

    await repo.upsert_player(db, body.user_id, body.name, ts)
    event_id = await repo.insert_event(db, body, ts)

    if body.event == "join":
        await repo.open_session(db, body, event_id, ts)
    elif body.event == "leave":
        await repo.close_session(db, body.user_id, body.location_id, event_id, ts)

    await db.commit()
