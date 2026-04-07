from pydantic import BaseModel
from dataclasses import dataclass

from models.common import UtcDatetime


@dataclass(slots=True)
class LocationInfo:
    world_id: str
    instance_id: str
    region: str
    group_id: str | None = None
    group_access_type: str | None = None
    friends: str | None = None
    hidden: str | None = None
    private: str | None = None

    @classmethod
    def parse(cls, location_id: str) -> "LocationInfo":
        world_part, _, rest = location_id.partition(":")
        parts = rest.split("~")
        if not world_part or not parts[0]:
            raise ValueError(f"Invalid location_id: {location_id!r}")

        region: str | None = None
        group_id = group_access_type = friends = hidden = private = None
        for part in parts[1:]:
            if "(" not in part or not part.endswith(")"):
                continue
            key, _, val = part.partition("(")
            val = val[:-1]
            if key == "region":
                region = val
            elif key == "group":
                group_id = val
            elif key == "groupAccessType":
                group_access_type = val
            elif key == "friends":
                friends = val
            elif key == "hidden":
                hidden = val
            elif key == "private":
                private = val

        if region is None:
            raise ValueError(f"region missing in location_id: {location_id!r}")

        return cls(
            world_id=world_part,
            instance_id=parts[0],
            region=region,
            group_id=group_id,
            group_access_type=group_access_type,
            friends=friends,
            hidden=hidden,
            private=private,
        )


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
    private: str | None
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
