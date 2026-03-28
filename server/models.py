from datetime import datetime

from pydantic import BaseModel


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
    timestamp: str


class SessionOut(BaseModel):
    id: int
    location_id: str
    user_id: str
    display_name: str
    join_ts: str
    leave_ts: str | None
    duration_seconds: int | None


class PlayerSessionOut(BaseModel):
    id: int
    location_id: str
    join_ts: str
    leave_ts: str | None
    duration_seconds: int | None


class LocationOut(BaseModel):
    location_id: str
    world_id: str
    first_seen: str
    last_seen: str


class PlayerOut(BaseModel):
    user_id: str
    display_name: str
    internal_id: int | None
    join_ts: str


class TimelinePoint(BaseModel):
    timestamp: str
    count: int
