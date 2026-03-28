from datetime import datetime, timezone


def to_utc_str(dt: datetime) -> str:
    """timezone-aware datetimeをUTC ISO 8601 Z付き文字列に変換する。"""
    return dt.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def parse_location_id(location_id: str) -> dict:
    """'wrld_xxx:07695~...' -> world_id"""
    world_part, _, _ = location_id.partition(":")
    return {"world_id": world_part}
