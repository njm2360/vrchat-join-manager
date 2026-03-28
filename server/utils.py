def parse_location_id(location_id: str) -> dict:
    """'wrld_xxx:07695~...' -> world_id"""
    world_part, _, _ = location_id.partition(":")
    return {"world_id": world_part}
