from pydantic import BaseModel

from models.common import UtcDatetime


class InstanceOut(BaseModel):
    id: int
    location_id: str
    world_id: str
    world_name: str | None
    instance_id: str | None
    group_id: str | None
    group_name: str | None
    group_access_type: str | None
    region: str | None
    friends: str | None
    hidden: str | None
    opened_at: UtcDatetime
    closed_at: UtcDatetime | None
    user_count: int


class LocationPlayerOut(BaseModel):
    user_id: str
    display_name: str
    internal_id: int | None
    join_ts: UtcDatetime
    join_count: int


class PlayerListOut(BaseModel):
    user_id: str
    display_name: str
    first_seen: UtcDatetime
    last_seen: UtcDatetime
    join_count: int
    total_duration_seconds: int | None


class TimelinePoint(BaseModel):
    timestamp: UtcDatetime
    count: int


class SessionOut(BaseModel):
    id: int
    instance_id: int
    user_id: str
    display_name: str
    join_ts: UtcDatetime
    leave_ts: UtcDatetime | None
    duration_seconds: int | None
    is_estimated_leave: bool
