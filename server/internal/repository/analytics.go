package repository

import (
	"context"
	"strings"

	"github.com/jmoiron/sqlx"
)

type AnalyticsRepo struct {
	DB *sqlx.DB
}

func NewAnalyticsRepo(db *sqlx.DB) *AnalyticsRepo { return &AnalyticsRepo{DB: db} }

func (r *AnalyticsRepo) DailyActiveUsers(ctx context.Context, worldID, groupID, start, end *string) ([]DailyActiveUsersRow, error) {
	conditions := []string{}
	args := map[string]interface{}{}
	if worldID != nil {
		conditions = append(conditions, "s.world_id = :world_id")
		args["world_id"] = *worldID
	}
	if groupID != nil {
		conditions = append(conditions, "i.group_id = :group_id")
		args["group_id"] = *groupID
	}
	if start != nil {
		conditions = append(conditions, "(s.leave_ts IS NULL OR s.leave_ts >= :start)")
		args["start"] = *start
	}
	if end != nil {
		conditions = append(conditions, "s.join_ts <= :end")
		args["end"] = *end
	}
	joinInstances := ""
	if groupID != nil {
		joinInstances = "JOIN instances i ON i.id = s.instance_id"
	}
	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	dayFilterParts := []string{}
	if start != nil {
		dayFilterParts = append(dayFilterParts, "day >= DATE(:start)")
	}
	if end != nil {
		dayFilterParts = append(dayFilterParts, "day <= DATE(:end)")
	}
	dayFilter := ""
	if len(dayFilterParts) > 0 {
		dayFilter = "WHERE " + strings.Join(dayFilterParts, " AND ")
	}

	q := `
		WITH RECURSIVE expanded AS (
		    SELECT s.user_id,
		           DATE(s.join_ts)                                     AS day,
		           DATE(COALESCE(s.leave_ts, datetime('now')))         AS last_day
		    FROM sessions s
		    ` + joinInstances + `
		    ` + where + `

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
		` + dayFilter + `
		GROUP BY day
		ORDER BY day DESC`

	stmt, err := r.DB.PrepareNamedContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows := []DailyActiveUsersRow{}
	if err := stmt.SelectContext(ctx, &rows, args); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *AnalyticsRepo) HourlyActiveUsers(ctx context.Context, worldID, groupID, start, end *string) ([]HourlyActiveUsersRow, error) {
	// Positional ? placeholders are used here because the SQL contains
	// literal ':00' inside strftime() format strings, which sqlx's named
	// parameter parser would otherwise misinterpret as bind variables.
	conditions := []string{}
	args := []interface{}{}
	if worldID != nil {
		conditions = append(conditions, "s.world_id = ?")
		args = append(args, *worldID)
	}
	if groupID != nil {
		conditions = append(conditions, "i.group_id = ?")
		args = append(args, *groupID)
	}
	if start != nil {
		conditions = append(conditions, "(s.leave_ts IS NULL OR s.leave_ts >= ?)")
		args = append(args, *start)
	}
	if end != nil {
		conditions = append(conditions, "s.join_ts <= ?")
		args = append(args, *end)
	}
	joinInstances := ""
	if groupID != nil {
		joinInstances = "JOIN instances i ON i.id = s.instance_id"
	}
	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	hourFilterParts := []string{}
	if start != nil {
		hourFilterParts = append(hourFilterParts, "hour >= strftime('%Y-%m-%dT%H:00:00', ?)")
		args = append(args, *start)
	}
	if end != nil {
		hourFilterParts = append(hourFilterParts, "hour <= strftime('%Y-%m-%dT%H:00:00', ?)")
		args = append(args, *end)
	}
	hourFilter := ""
	if len(hourFilterParts) > 0 {
		hourFilter = "WHERE " + strings.Join(hourFilterParts, " AND ")
	}

	q := `
		WITH RECURSIVE expanded AS (
		    SELECT s.user_id,
		           strftime('%Y-%m-%dT%H:00:00', s.join_ts)                          AS hour,
		           strftime('%Y-%m-%dT%H:00:00', COALESCE(s.leave_ts, datetime('now'))) AS last_hour
		    FROM sessions s
		    ` + joinInstances + `
		    ` + where + `

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
		` + hourFilter + `
		GROUP BY hour
		ORDER BY hour DESC`

	rows := []HourlyActiveUsersRow{}
	if err := r.DB.SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *AnalyticsRepo) PlayerRankings(ctx context.Context, worldID, groupID *string, order string, limit *int, offset int) ([]PlayerRankRow, error) {
	conditions := []string{}
	args := map[string]interface{}{}
	if worldID != nil {
		conditions = append(conditions, "s.world_id = :world_id")
		args["world_id"] = *worldID
	}
	if groupID != nil {
		conditions = append(conditions, "i.group_id = :group_id")
		args["group_id"] = *groupID
	}
	joinInstances := ""
	if groupID != nil {
		joinInstances = "JOIN instances i ON i.id = s.instance_id"
	}
	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	o := orderUpper(order)
	q := `
		SELECT RANK() OVER (ORDER BY total_duration_seconds ` + o + `) AS rank,
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
		    ` + joinInstances + `
		    ` + where + `
		    GROUP BY s.user_id
		)
		ORDER BY total_duration_seconds ` + o + `, user_id ` + o + `
		` + limitClause(limit, offset)

	stmt, err := r.DB.PrepareNamedContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows := []PlayerRankRow{}
	if err := stmt.SelectContext(ctx, &rows, args); err != nil {
		return nil, err
	}
	return rows, nil
}

type JoinViolationRankingsParams struct {
	GroupID       string
	Start, End    *string
	Order         string
	Limit         *int
	Offset        int
	AllowDiff     int
	MinDuration   *int
	RejoinSeconds int
	GraceSeconds  int
}

func (r *AnalyticsRepo) JoinViolationRankings(ctx context.Context, p JoinViolationRankingsParams) ([]JoinViolationRankRow, error) {
	args := map[string]interface{}{
		"group_id":       p.GroupID,
		"allow_diff":     p.AllowDiff,
		"rejoin_seconds": p.RejoinSeconds,
		"grace_seconds":  p.GraceSeconds,
	}
	timeConditions := []string{}
	if p.Start != nil {
		timeConditions = append(timeConditions, "s.join_ts >= :start")
		args["start"] = *p.Start
	}
	if p.End != nil {
		timeConditions = append(timeConditions, "s.join_ts <= :end")
		args["end"] = *p.End
	}
	if p.MinDuration != nil {
		timeConditions = append(timeConditions,
			"CAST(ROUND((julianday(COALESCE(s.leave_ts, 'now')) - julianday(s.join_ts)) * 86400) AS INTEGER) >= :min_duration")
		args["min_duration"] = *p.MinDuration
	}
	timeWhere := ""
	if len(timeConditions) > 0 {
		timeWhere = "AND " + strings.Join(timeConditions, " AND ")
	}

	o := orderUpper(p.Order)
	q := `
		WITH
		all_joins AS (
		    SELECT
		        s.id          AS session_id,
		        s.user_id,
		        s.instance_id,
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
		    ` + timeWhere + `
		    AND NOT EXISTS (
		        SELECT 1 FROM sessions prev
		        WHERE prev.user_id = s.user_id
		          AND prev.instance_id = s.instance_id
		          AND prev.leave_ts IS NOT NULL
		          AND prev.leave_ts <= s.join_ts
		          AND CAST(ROUND((julianday(s.join_ts) - julianday(prev.leave_ts)) * 86400) AS INTEGER) <= :rejoin_seconds
		    )
		),
		with_other AS (
		    SELECT
		        aj.*,
		        (
		            SELECT COUNT(*) FROM sessions s3
		            WHERE s3.instance_id = aj.other_iid
		              AND s3.join_ts <= aj.join_ts
		              AND (s3.leave_ts IS NULL OR s3.leave_ts > aj.join_ts)
		        ) AS other_count
		    FROM all_joins aj
		    WHERE aj.other_iid IS NOT NULL
		),
		with_run AS (
		    SELECT
		        wo.*,
		        wo.my_count - wo.other_count AS diff,
		        SUM(CASE WHEN wo.my_count - wo.other_count <= :allow_diff THEN 1 ELSE 0 END)
		            OVER (PARTITION BY wo.instance_id ORDER BY wo.join_ts
		                  ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS run_grp
		    FROM with_other wo
		),
		with_run_start AS (
		    SELECT
		        wr.*,
		        MIN(wr.join_ts) OVER (
		            PARTITION BY wr.instance_id, wr.run_grp
		        ) AS diff_start_ts
		    FROM with_run wr
		    WHERE wr.diff > :allow_diff
		),
		user_totals AS (
		    SELECT user_id, COUNT(*) AS total_joins
		    FROM with_other
		    GROUP BY user_id
		),
		user_violations AS (
		    SELECT
		        user_id,
		        SUM(
		            CASE WHEN CAST(ROUND((julianday(join_ts) - julianday(diff_start_ts)) * 86400) AS INTEGER) > :grace_seconds
		            THEN 1 ELSE 0 END
		        ) AS violation_count
		    FROM with_run_start
		    GROUP BY user_id
		),
		user_stats AS (
		    SELECT
		        ut.user_id,
		        COALESCE(uv.violation_count, 0) AS violation_count,
		        ut.total_joins
		    FROM user_totals ut
		    LEFT JOIN user_violations uv ON uv.user_id = ut.user_id
		)
		SELECT
		    RANK() OVER (ORDER BY violation_count ` + o + `) AS rank,
		    us.user_id,
		    p.display_name,
		    us.violation_count,
		    us.total_joins
		FROM user_stats us
		JOIN players p ON p.user_id = us.user_id
		WHERE us.violation_count > 0
		ORDER BY violation_count ` + o + `, us.user_id ` + o + `
		` + limitClause(p.Limit, p.Offset)

	stmt, err := r.DB.PrepareNamedContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows := []JoinViolationRankRow{}
	if err := stmt.SelectContext(ctx, &rows, args); err != nil {
		return nil, err
	}
	return rows, nil
}
