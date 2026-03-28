from datetime import datetime, timezone
from typing import Annotated

from pydantic import BaseModel, PlainSerializer


def _fmt_utc(v: datetime) -> str:
    return v.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


UtcDatetime = Annotated[datetime, PlainSerializer(_fmt_utc, return_type=str)]


class PlayerEvent(BaseModel):
    event: str
    location_id: str
    name: str
    user_id: str
    internal_id: int | None
    timestamp: datetime


class EventOut(BaseModel):
    id: int
    event_type: str
    location_id: str
    world_id: str
    user_id: str
    display_name: str
    internal_id: int | None
    timestamp: UtcDatetime


class SessionOut(BaseModel):
    id: int
    location_id: str
    user_id: str
    display_name: str
    join_ts: UtcDatetime
    leave_ts: UtcDatetime | None
    duration_seconds: int | None


class PlayerSessionOut(BaseModel):
    id: int
    location_id: str
    join_ts: UtcDatetime
    leave_ts: UtcDatetime | None
    duration_seconds: int | None


class LocationOut(BaseModel):
    location_id: str
    world_id: str
    first_seen: UtcDatetime
    last_seen: UtcDatetime


class PlayerOut(BaseModel):
    user_id: str
    display_name: str
    internal_id: int | None
    join_ts: UtcDatetime
    join_count: int


class TimelinePoint(BaseModel):
    timestamp: UtcDatetime
    count: int
