from pydantic import BaseModel

from models.common import UtcDatetime


class PlayerOut(BaseModel):
    user_id: str
    display_name: str
    created_at: UtcDatetime
    updated_at: UtcDatetime


class PlayerSessionOut(BaseModel):
    id: int
    instance_id: int
    world_id: str
    join_ts: UtcDatetime
    leave_ts: UtcDatetime | None
    duration_seconds: int | None
    is_estimated_leave: bool
