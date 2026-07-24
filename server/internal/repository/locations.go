package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"
)

type LocationsRepo struct {
	DB *sqlx.DB
}

func NewLocationsRepo(db *sqlx.DB) *LocationsRepo { return &LocationsRepo{DB: db} }

const instanceSelect = `
	SELECT i.id, i.location_id, i.world_id, w.name AS world_name,
	       i.instance_id, i.group_id, g.name AS group_name,
	       i.group_access_type, i.region, i.friends, i.hidden, i.private, i.opened_at, i.closed_at,
	       (SELECT COUNT(*) FROM sessions s WHERE s.instance_id = i.id AND s.leave_ts IS NULL) AS user_count
	FROM instances i
	JOIN worlds w ON w.world_id = i.world_id
	LEFT JOIN groups g ON g.group_id = i.group_id
`

func (r *LocationsRepo) ListInstances(
	ctx context.Context,
	start, end *string,
	isOpen *bool,
	worldID, groupID, region *string,
	sortBy, order string,
	limit *int,
	offset int,
) ([]InstanceRow, error) {
	conditions := []string{}
	args := map[string]interface{}{}
	if start != nil {
		conditions = append(conditions, "(i.closed_at IS NULL OR i.closed_at >= :start)")
		args["start"] = *start
	}
	if end != nil {
		conditions = append(conditions, "i.opened_at <= :end")
		args["end"] = *end
	}
	if isOpen != nil {
		if *isOpen {
			conditions = append(conditions, "i.closed_at IS NULL")
		} else {
			conditions = append(conditions, "i.closed_at IS NOT NULL")
		}
	}
	if worldID != nil {
		conditions = append(conditions, "i.world_id = :world_id")
		args["world_id"] = *worldID
	}
	if groupID != nil {
		conditions = append(conditions, "i.group_id = :group_id")
		args["group_id"] = *groupID
	}
	if region != nil {
		conditions = append(conditions, "i.region = :region")
		args["region"] = *region
	}
	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	sortCol := pickSortColumn(sortBy, map[string]string{
		"opened_at": "i.opened_at",
		"closed_at": "i.closed_at",
	}, "i.opened_at")

	q := instanceSelect + where + `
		ORDER BY ` + sortCol + ` ` + orderUpper(order) + `, i.id ` + orderUpper(order) + `
		` + limitClause(limit, offset)

	stmt, err := r.DB.PrepareNamedContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows := []InstanceRow{}
	if err := stmt.SelectContext(ctx, &rows, args); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *LocationsRepo) GetInstance(ctx context.Context, instanceID int) (*InstanceRow, error) {
	q := instanceSelect + `WHERE i.id = ?`
	var row InstanceRow
	if err := r.DB.GetContext(ctx, &row, q, instanceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *LocationsRepo) GetPresence(ctx context.Context, instanceID int, at string) ([]SessionRow, error) {
	q := `
		SELECT s.id, s.instance_id, s.internal_id, s.user_id, p.display_name, pd.discord_id, s.join_ts, s.leave_ts,
		       COALESCE(s.duration_seconds,
		           CAST(ROUND((julianday('now') - julianday(s.join_ts)) * 86400) AS INTEGER)
		       ) AS duration_seconds,
		       s.is_estimated_join, s.is_estimated_leave
		FROM sessions s
		JOIN players p ON p.user_id = s.user_id
		LEFT JOIN player_discord pd ON pd.user_id = s.user_id
		WHERE s.instance_id = ?
		  AND s.join_ts  <= ?
		  AND (s.leave_ts IS NULL OR s.leave_ts >= ?)
		ORDER BY s.join_ts`

	rows := []SessionRow{}
	if err := r.DB.SelectContext(ctx, &rows, q, instanceID, at, at); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *LocationsRepo) GetLocationPlayers(ctx context.Context, instanceID int, sortBy, order string) ([]LocationPlayerRow, error) {
	col := pickSortColumn(sortBy, map[string]string{
		"internal_id":  "s.internal_id",
		"display_name": "p.display_name",
		"join_ts":      "s.join_ts",
	}, "s.internal_id")
	q := `
		SELECT s.user_id, p.display_name, pd.discord_id, s.internal_id, s.join_ts,
		       (SELECT COUNT(*) FROM sessions s2
		        WHERE s2.user_id = s.user_id AND s2.instance_id = s.instance_id) AS join_count
		FROM sessions s
		JOIN players p ON p.user_id = s.user_id
		LEFT JOIN player_discord pd ON pd.user_id = s.user_id
		WHERE s.instance_id = ?
		  AND s.leave_ts IS NULL
		ORDER BY ` + col + ` ` + orderUpper(order)

	rows := []LocationPlayerRow{}
	if err := r.DB.SelectContext(ctx, &rows, q, instanceID); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *LocationsRepo) GetLocationVisitors(ctx context.Context, instanceID int, sortBy, order string, limit *int, offset int) ([]VisitorRow, error) {
	sortCol := pickSortColumn(sortBy, map[string]string{
		"display_name":           "display_name",
		"first_seen":             "first_seen",
		"last_seen":              "last_seen",
		"join_count":             "join_count",
		"total_duration_seconds": "total_duration_seconds",
	}, "last_seen")
	q := `
		SELECT s.user_id, p.display_name,
		       MIN(s.join_ts)          AS first_seen,
		       MAX(COALESCE(s.leave_ts, strftime('%Y-%m-%dT%H:%M:%SZ','now'))) AS last_seen,
		       COUNT(*)                AS join_count,
		       COALESCE(SUM(COALESCE(s.duration_seconds,
		           CAST(ROUND((julianday('now') - julianday(s.join_ts)) * 86400) AS INTEGER)
		       )), 0)                  AS total_duration_seconds
		FROM sessions s
		JOIN players p ON p.user_id = s.user_id
		WHERE s.instance_id = ?
		GROUP BY s.user_id
		ORDER BY ` + sortCol + ` ` + orderUpper(order) + `, s.user_id ` + orderUpper(order) + `
		` + limitClause(limit, offset)

	rows := []VisitorRow{}
	if err := r.DB.SelectContext(ctx, &rows, q, instanceID); err != nil {
		return nil, err
	}
	return rows, nil
}

type timelinePointRow struct {
	UserID      string `db:"user_id"`
	DisplayName string `db:"display_name"`
	Timestamp   string `db:"timestamp"`
	Delta       int    `db:"delta"`
}

func (r *LocationsRepo) GetPresenceTimeline(ctx context.Context, instanceID int, start, end *string) ([]TimelinePointRow, error) {
	initial := 0
	if start != nil {
		if err := r.DB.GetContext(ctx, &initial,
			`SELECT COUNT(*) FROM sessions
			  WHERE instance_id = ?
			    AND join_ts <= ?
			    AND (leave_ts IS NULL OR leave_ts > ?)`,
			instanceID, *start, *start,
		); err != nil {
			return nil, err
		}
	}

	joinConds := []string{"s.instance_id = :instance_id"}
	leaveConds := []string{"s.instance_id = :instance_id", "s.leave_ts IS NOT NULL"}
	args := map[string]interface{}{"instance_id": instanceID}
	if start != nil {
		joinConds = append(joinConds, "s.join_ts > :start")
		leaveConds = append(leaveConds, "s.leave_ts > :start")
		args["start"] = *start
	}
	if end != nil {
		joinConds = append(joinConds, "s.join_ts <= :end")
		leaveConds = append(leaveConds, "s.leave_ts <= :end")
		args["end"] = *end
	}

	q := `
		WITH pts AS (
			SELECT s.user_id, p.display_name, s.join_ts AS timestamp, 1 AS delta, 1 AS ord
			FROM sessions s
			JOIN players p ON p.user_id = s.user_id
			WHERE ` + strings.Join(joinConds, " AND ") + `
			UNION ALL
			SELECT s.user_id, p.display_name, s.leave_ts AS timestamp, -1 AS delta, 0 AS ord
			FROM sessions s
			JOIN players p ON p.user_id = s.user_id
			WHERE ` + strings.Join(leaveConds, " AND ") + `
		)
		SELECT user_id, display_name, timestamp, delta
		FROM pts
		ORDER BY timestamp, ord, user_id`

	stmt, err := r.DB.PrepareNamedContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	pts := []timelinePointRow{}
	if err := stmt.SelectContext(ctx, &pts, args); err != nil {
		return nil, err
	}

	var anchor string
	if start != nil {
		anchor = *start
	} else if len(pts) > 0 {
		anchor = pts[0].Timestamp
	} else {
		return []TimelinePointRow{}, nil
	}

	points := []TimelinePointRow{{
		Timestamp: anchor,
		Count:     initial,
	}}
	count := initial
	for _, pt := range pts {
		count += pt.Delta
		points = append(points, TimelinePointRow{
			Timestamp:   pt.Timestamp,
			Count:       count,
			UserID:      sql.NullString{String: pt.UserID, Valid: true},
			DisplayName: sql.NullString{String: pt.DisplayName, Valid: true},
		})
	}
	return points, nil
}

func (r *LocationsRepo) GetLocationEvents(ctx context.Context, instanceID int, start, end *string, order string, limit *int, offset int) ([]EventRow, error) {
	conditions := []string{"instance_id = :instance_id"}
	args := map[string]interface{}{"instance_id": instanceID}
	if start != nil {
		conditions = append(conditions, "timestamp >= :start")
		args["start"] = *start
	}
	if end != nil {
		conditions = append(conditions, "timestamp <= :end")
		args["end"] = *end
	}
	where := strings.Join(conditions, " AND ")

	q := `
		SELECT e.id, e.event_type, e.instance_id, e.world_id, e.user_id, p.display_name, e.timestamp
		FROM events e
		JOIN players p ON p.user_id = e.user_id
		WHERE ` + where + `
		ORDER BY e.timestamp ` + orderUpper(order) + `, e.id ` + orderUpper(order) + `
		` + limitClause(limit, offset)

	stmt, err := r.DB.PrepareNamedContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows := []EventRow{}
	if err := stmt.SelectContext(ctx, &rows, args); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *LocationsRepo) GetLocationSessions(ctx context.Context, instanceID int, start, end *string, sortBy, order string, limit *int, offset int) ([]SessionRow, error) {
	sortCol := pickSortColumn(sortBy, map[string]string{
		"internal_id":      "s.internal_id",
		"display_name":     "p.display_name",
		"join_ts":          "s.join_ts",
		"leave_ts":         "s.leave_ts",
		"duration_seconds": "duration_seconds",
	}, "s.join_ts")
	conditions := []string{"s.instance_id = :instance_id"}
	args := map[string]any{"instance_id": instanceID}
	if start != nil {
		conditions = append(conditions, "(s.leave_ts IS NULL OR s.leave_ts >= :start)")
		args["start"] = *start
	}
	if end != nil {
		conditions = append(conditions, "s.join_ts <= :end")
		args["end"] = *end
	}
	where := strings.Join(conditions, " AND ")

	q := `
		SELECT s.id, s.instance_id, s.internal_id, s.user_id, p.display_name, pd.discord_id, s.join_ts, s.leave_ts,
		       COALESCE(s.duration_seconds,
		           CAST(ROUND((julianday('now') - julianday(s.join_ts)) * 86400) AS INTEGER)
		       ) AS duration_seconds,
		       s.is_estimated_join, s.is_estimated_leave
		FROM sessions s
		JOIN players p ON p.user_id = s.user_id
		LEFT JOIN player_discord pd ON pd.user_id = s.user_id
		WHERE ` + where + `
		ORDER BY ` + sortCol + ` ` + orderUpper(order) + `, s.id ` + orderUpper(order) + `
		` + limitClause(limit, offset)

	stmt, err := r.DB.PrepareNamedContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows := []SessionRow{}
	if err := stmt.SelectContext(ctx, &rows, args); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *LocationsRepo) GetInstanceStats(ctx context.Context, instanceID int) (*InstanceStatsRow, error) {
	const dur = `COALESCE(duration_seconds, CAST(ROUND((julianday('now')-julianday(join_ts))*86400) AS INTEGER))`
	q := `
		SELECT
		  (SELECT COUNT(*) FROM events   WHERE instance_id = :id)                       AS event_count,
		  (SELECT MIN(timestamp) FROM events WHERE instance_id = :id)                   AS first_event_at,
		  (SELECT MAX(timestamp) FROM events WHERE instance_id = :id)                   AS last_event_at,
		  (SELECT COUNT(*) FROM sessions WHERE instance_id = :id)                       AS session_count,
		  (SELECT COUNT(DISTINCT user_id) FROM sessions WHERE instance_id = :id)        AS visitor_count,
		  (SELECT COUNT(*) FROM sessions WHERE instance_id = :id AND leave_ts IS NULL)  AS present_count,
		  (SELECT COUNT(*) FROM (
		      SELECT 1 FROM sessions WHERE instance_id = :id GROUP BY user_id HAVING COUNT(*) > 1
		   ))                                                                           AS repeat_visitor_count,
		  (SELECT COALESCE(SUM(d),0)                FROM (SELECT ` + dur + ` AS d FROM sessions WHERE instance_id = :id)) AS total_duration_seconds,
		  (SELECT COALESCE(CAST(ROUND(AVG(d)) AS INTEGER),0) FROM (SELECT ` + dur + ` AS d FROM sessions WHERE instance_id = :id)) AS avg_session_seconds,
		  (SELECT COALESCE(MAX(run),0) FROM (
		      SELECT SUM(delta) OVER (ORDER BY ts, delta DESC) AS run FROM (
		          SELECT join_ts  AS ts, 1  AS delta FROM sessions WHERE instance_id = :id
		          UNION ALL
		          SELECT leave_ts AS ts, -1 AS delta FROM sessions WHERE instance_id = :id AND leave_ts IS NOT NULL
		      )
		   ))                                                                           AS peak_concurrent`

	stmt, err := r.DB.PrepareNamedContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var row InstanceStatsRow
	if err := stmt.GetContext(ctx, &row, map[string]interface{}{"id": instanceID}); err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *LocationsRepo) DiscordIDsPresent(ctx context.Context, instanceID int) ([]string, error) {
	q := `
		SELECT pd.discord_id
		FROM sessions s
		JOIN player_discord pd ON pd.user_id = s.user_id
		WHERE s.instance_id = ?
		  AND s.leave_ts IS NULL
		  AND pd.discord_id IS NOT NULL
		  AND pd.discord_id <> ''
		ORDER BY s.internal_id`
	ids := []string{}
	if err := r.DB.SelectContext(ctx, &ids, q, instanceID); err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *LocationsRepo) DiscordIDsAtClose(ctx context.Context, instanceID int) ([]string, error) {
	q := `
		SELECT pd.discord_id
		FROM sessions s
		JOIN instances i ON i.id = s.instance_id
		JOIN player_discord pd ON pd.user_id = s.user_id
		WHERE s.instance_id = ?
		  AND i.closed_at IS NOT NULL
		  AND s.leave_ts = i.closed_at
		  AND s.is_estimated_leave = 1
		  AND pd.discord_id IS NOT NULL
		  AND pd.discord_id <> ''
		ORDER BY s.internal_id`
	ids := []string{}
	if err := r.DB.SelectContext(ctx, &ids, q, instanceID); err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *LocationsRepo) ListDiscordMentions(
	ctx context.Context,
	start, end *string,
	groupID, worldID, region *string,
	instanceID *int,
	present *bool,
) ([]string, error) {
	conditions := []string{
		"pd.discord_id IS NOT NULL",
		"pd.discord_id <> ''",
	}
	args := map[string]any{}
	if start != nil {
		conditions = append(conditions, "s.join_ts >= :start")
		args["start"] = *start
	}
	if end != nil {
		conditions = append(conditions, "s.join_ts <= :end")
		args["end"] = *end
	}
	if groupID != nil {
		conditions = append(conditions, "i.group_id = :group_id")
		args["group_id"] = *groupID
	}
	if worldID != nil {
		conditions = append(conditions, "i.world_id = :world_id")
		args["world_id"] = *worldID
	}
	if region != nil {
		conditions = append(conditions, "i.region = :region")
		args["region"] = *region
	}
	if instanceID != nil {
		conditions = append(conditions, "s.instance_id = :instance_id")
		args["instance_id"] = *instanceID
	}
	if present != nil && *present {
		conditions = append(conditions, "s.leave_ts IS NULL")
	}

	q := `
		SELECT DISTINCT pd.discord_id
		FROM sessions s
		JOIN instances i ON i.id = s.instance_id
		JOIN player_discord pd ON pd.user_id = s.user_id
		WHERE ` + strings.Join(conditions, "\n\t\t  AND ") + `
		ORDER BY pd.discord_id`

	stmt, err := r.DB.PrepareNamedContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	ids := []string{}
	if err := stmt.SelectContext(ctx, &ids, args); err != nil {
		return nil, err
	}
	return ids, nil
}
