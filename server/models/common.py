from datetime import datetime, timezone
from typing import Annotated

from pydantic import BaseModel, PlainSerializer


def _fmt_utc(v: datetime) -> str:
    return v.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


UtcDatetime = Annotated[datetime, PlainSerializer(_fmt_utc, return_type=str)]


class EventOut(BaseModel):
    id: int
    event_type: str
    instance_id: int
    world_id: str
    user_id: str
    display_name: str
    timestamp: UtcDatetime
