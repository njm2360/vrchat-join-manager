from datetime import datetime

import aiosqlite

from models.analytics import (
    DailyActiveUsersPoint,
    HourlyActiveUsersPoint,
    JoinViolationRankOut,
    PlayerRankOut,
)
from utils import to_utc_str


async def get_daily_active_users(
    db: aiosqlite.Connection,
    world_id: str | None,
    group_id: str | None,
    start: datetime | None,
    end: datetime | None,
) -> list[DailyActiveUsersPoint]:
    conditions: list[str] = []
    params: dict = {}

    if world_id is not None:
        conditions.append("s.world_id = :world_id")
        params["world_id"] = world_id
    if group_id is not None:
        conditions.append("i.group_id = :group_id")
        params["group_id"] = group_id
    if start is not None:
        # 開始日より前に終了したセッションは除外
        conditions.append("(s.leave_ts IS NULL OR s.leave_ts >= :start)")
        params["start"] = to_utc_str(start)
    if end is not None:
        # 終了日より後に開始したセッションは除外
        conditions.append("s.join_ts <= :end")
        params["end"] = to_utc_str(end)

    join_instances = (
        "JOIN instances i ON i.id = s.instance_id" if group_id is not None else ""
    )
    where = ("WHERE " + " AND ".join(conditions)) if conditions else ""

    # 結果を指定範囲の日付のみに絞るフィルタ
    day_filter_parts: list[str] = []
    if start is not None:
        day_filter_parts.append("day >= DATE(:start)")
    if end is not None:
        day_filter_parts.append("day <= DATE(:end)")
    day_filter = ("WHERE " + " AND ".join(day_filter_parts)) if day_filter_parts else ""

    cursor = await db.execute(
        f"""
        WITH RECURSIVE expanded AS (
            SELECT s.user_id,
                   DATE(s.join_ts)                                     AS day,
                   DATE(COALESCE(s.leave_ts, datetime('now')))         AS last_day
            FROM sessions s
            {join_instances}
            {where}

            UNION ALL

            SELECT user_id,
                   DATE(day, '+1 day'),
                   last_day
            FROM expanded
            WHERE day < last_day
        )
        SELECT day,
               COUNT(DISTINCT user_id) AS active_users
        FROM expanded
        {day_filter}
        GROUP BY day
        ORDER BY day DESC
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [DailyActiveUsersPoint(**dict(row)) for row in rows]


async def get_hourly_active_users(
    db: aiosqlite.Connection,
    world_id: str | None,
    group_id: str | None,
    start: datetime | None,
    end: datetime | None,
) -> list[HourlyActiveUsersPoint]:
    conditions: list[str] = []
    params: dict = {}

    if world_id is not None:
        conditions.append("s.world_id = :world_id")
        params["world_id"] = world_id
    if group_id is not None:
        conditions.append("i.group_id = :group_id")
        params["group_id"] = group_id
    if start is not None:
        conditions.append("(s.leave_ts IS NULL OR s.leave_ts >= :start)")
        params["start"] = to_utc_str(start)
    if end is not None:
        conditions.append("s.join_ts <= :end")
        params["end"] = to_utc_str(end)

    join_instances = (
        "JOIN instances i ON i.id = s.instance_id" if group_id is not None else ""
    )
    where = ("WHERE " + " AND ".join(conditions)) if conditions else ""

    hour_filter_parts: list[str] = []
    if start is not None:
        hour_filter_parts.append("hour >= strftime('%Y-%m-%dT%H:00:00', :start)")
    if end is not None:
        hour_filter_parts.append("hour <= strftime('%Y-%m-%dT%H:00:00', :end)")
    hour_filter = (
        ("WHERE " + " AND ".join(hour_filter_parts)) if hour_filter_parts else ""
    )

    cursor = await db.execute(
        f"""
        WITH RECURSIVE expanded AS (
            SELECT s.user_id,
                   strftime('%Y-%m-%dT%H:00:00', s.join_ts)                          AS hour,
                   strftime('%Y-%m-%dT%H:00:00', COALESCE(s.leave_ts, datetime('now'))) AS last_hour
            FROM sessions s
            {join_instances}
            {where}

            UNION ALL

            SELECT user_id,
                   strftime('%Y-%m-%dT%H:00:00', datetime(hour, '+1 hour')),
                   last_hour
            FROM expanded
            WHERE hour < last_hour
        )
        SELECT hour,
               COUNT(DISTINCT user_id) AS active_users
        FROM expanded
        {hour_filter}
        GROUP BY hour
        ORDER BY hour DESC
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [HourlyActiveUsersPoint(**dict(row)) for row in rows]


async def get_player_rankings(
    db: aiosqlite.Connection,
    world_id: str | None,
    group_id: str | None,
    order: str = "desc",
    limit: int | None = None,
    offset: int = 0,
) -> list[PlayerRankOut]:
    conditions: list[str] = []
    params: dict = {}

    if world_id is not None:
        conditions.append("s.world_id = :world_id")
        params["world_id"] = world_id
    if group_id is not None:
        conditions.append("i.group_id = :group_id")
        params["group_id"] = group_id

    join_instances = (
        "JOIN instances i ON i.id = s.instance_id" if group_id is not None else ""
    )
    where = ("WHERE " + " AND ".join(conditions)) if conditions else ""
    limit_clause = (
        f"LIMIT {limit} OFFSET {offset}"
        if limit is not None
        else f"LIMIT -1 OFFSET {offset}"
    )
    cursor = await db.execute(
        f"""
        SELECT RANK() OVER (ORDER BY total_duration_seconds {order.upper()}) AS rank,
               user_id,
               display_name,
               total_duration_seconds,
               session_count
        FROM (
            SELECT s.user_id,
                   p.display_name,
                   SUM(COALESCE(s.duration_seconds,
                       CAST(ROUND((julianday('now') - julianday(s.join_ts)) * 86400) AS INTEGER)
                   )) AS total_duration_seconds,
                   COUNT(s.id) AS session_count
            FROM sessions s
            JOIN players p ON p.user_id = s.user_id
            {join_instances}
            {where}
            GROUP BY s.user_id
        )
        ORDER BY total_duration_seconds {order.upper()}
        {limit_clause}
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [PlayerRankOut(**dict(row)) for row in rows]


async def get_join_violation_rankings(
    db: aiosqlite.Connection,
    group_id: str,
    start: datetime | None = None,
    end: datetime | None = None,
    order: str = "desc",
    limit: int | None = None,
    offset: int = 0,
) -> list[JoinViolationRankOut]:
    params: dict = {"group_id": group_id}
    time_conditions: list[str] = []
    if start is not None:
        time_conditions.append("s.join_ts >= :start")
        params["start"] = to_utc_str(start)
    if end is not None:
        time_conditions.append("s.join_ts <= :end")
        params["end"] = to_utc_str(end)
    time_where = ("AND " + " AND ".join(time_conditions)) if time_conditions else ""

    limit_clause = (
        f"LIMIT {limit} OFFSET {offset}"
        if limit is not None
        else f"LIMIT -1 OFFSET {offset}"
    )
    cursor = await db.execute(
        f"""
        WITH
        session_counts AS (
            SELECT
                s.id          AS session_id,
                s.user_id,
                s.instance_id AS joined_iid,
                s.join_ts,
                (
                    SELECT COUNT(*) FROM sessions s2
                    WHERE s2.instance_id = s.instance_id
                      AND s2.join_ts < s.join_ts
                      AND (s2.leave_ts IS NULL OR s2.leave_ts > s.join_ts)
                ) AS my_count,
                (
                    SELECT i2.id FROM instances i2
                    WHERE i2.group_id = :group_id
                      AND i2.id != s.instance_id
                      AND i2.opened_at <= s.join_ts
                      AND (i2.closed_at IS NULL OR i2.closed_at > s.join_ts)
                    LIMIT 1
                ) AS other_iid
            FROM sessions s
            JOIN instances i ON i.id = s.instance_id
            WHERE i.group_id = :group_id
            {time_where}
        ),
        with_other AS (
            SELECT
                sc.*,
                (
                    SELECT COUNT(*) FROM sessions s3
                    WHERE s3.instance_id = sc.other_iid
                      AND s3.join_ts < sc.join_ts
                      AND (s3.leave_ts IS NULL OR s3.leave_ts > sc.join_ts)
                ) AS other_count
            FROM session_counts sc
            WHERE sc.other_iid IS NOT NULL
        ),
        user_stats AS (
            SELECT
                user_id,
                SUM(CASE WHEN my_count > other_count THEN 1 ELSE 0 END) AS violation_count,
                COUNT(*) AS total_joins
            FROM with_other
            GROUP BY user_id
        )
        SELECT
            RANK() OVER (ORDER BY violation_count {order.upper()}) AS rank,
            us.user_id,
            p.display_name,
            us.violation_count,
            us.total_joins
        FROM user_stats us
        JOIN players p ON p.user_id = us.user_id
        WHERE us.violation_count > 0
        ORDER BY violation_count {order.upper()}
        {limit_clause}
        """,
        params,
    )
    rows = await cursor.fetchall()
    return [JoinViolationRankOut(**dict(row)) for row in rows]
