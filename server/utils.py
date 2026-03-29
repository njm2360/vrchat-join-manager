from dataclasses import dataclass
from datetime import datetime, timezone


def to_utc_str(dt: datetime) -> str:
    return dt.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


@dataclass(slots=True)
class LocationInfo:
    world_id: str
    instance_id: str | None
    group_id: str | None
    group_access_type: str | None
    region: str | None
    friends: str | None
    hidden: str | None


def parse_location_id(location_id: str) -> LocationInfo:
    world_part, _, rest = location_id.partition(":")
    parts = rest.split("~")

    info = LocationInfo(
        world_id=world_part,
        instance_id=parts[0] if parts else None,
        group_id=None,
        group_access_type=None,
        region=None,
        friends=None,
        hidden=None,
    )
    for part in parts[1:]:
        if "(" not in part or not part.endswith(")"):
            continue
        key, _, val = part.partition("(")
        val = val[:-1]  # 末尾の ) を除去
        if key == "group":
            info.group_id = val
        elif key == "groupAccessType":
            info.group_access_type = val
        elif key == "region":
            info.region = val
        elif key == "friends":
            info.friends = val
        elif key == "hidden":
            info.hidden = val
    return info
