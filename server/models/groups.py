from pydantic import BaseModel

from models.common import UtcDatetime


class GroupOut(BaseModel):
    group_id: str
    name: str | None
    created_at: UtcDatetime
    updated_at: UtcDatetime


class GroupRenameIn(BaseModel):
    name: str
