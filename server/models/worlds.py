from pydantic import BaseModel

from models.common import UtcDatetime


class WorldOut(BaseModel):
    world_id: str
    name: str | None
    created_at: UtcDatetime
    updated_at: UtcDatetime
    last_seen: UtcDatetime | None
    session_count: int


class WorldRenameIn(BaseModel):
    name: str
