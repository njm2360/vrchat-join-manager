from datetime import datetime

from pydantic import BaseModel


class PlayerEvent(BaseModel):
    event: str
    location_id: str
    name: str
    user_id: str
    internal_id: int | None
    timestamp: datetime
