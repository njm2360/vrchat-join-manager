from datetime import datetime
from typing import Literal

from pydantic import BaseModel, field_validator

from models.locations import LocationInfo


class PlayerEvent(BaseModel):
    event: Literal["join", "leave"]
    location_id: str
    name: str
    user_id: str
    internal_id: int | None
    timestamp: datetime

    @field_validator("location_id")
    @classmethod
    def validate_location_id(cls, v: str) -> str:
        LocationInfo.parse(v)
        return v
