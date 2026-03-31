from pydantic import BaseModel


class PlayerRankOut(BaseModel):
    rank: int
    user_id: str
    display_name: str
    total_duration_seconds: int
    session_count: int


class DailyActiveUsersPoint(BaseModel):
    day: str
    active_users: int


class HourlyActiveUsersPoint(BaseModel):
    hour: str
    active_users: int


class JoinViolationRankOut(BaseModel):
    rank: int
    user_id: str
    display_name: str
    violation_count: int
    total_joins: int
