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
		ORDER BY ` + sortCol + ` ` + orderUpper(order) + `
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
		       s.is_estimated_leave
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
		       MAX(s.join_ts)          AS last_seen,
		       COUNT(*)                AS join_count,
		       COALESCE(SUM(COALESCE(s.duration_seconds,
		           CAST(ROUND((julianday('now') - julianday(s.join_ts)) * 86400) AS INTEGER)
		       )), 0)                  AS total_duration_seconds
		FROM sessions s
		JOIN players p ON p.user_id = s.user_id
		WHERE s.instance_id = ?
		GROUP BY s.user_id
		ORDER BY ` + sortCol + ` ` + orderUpper(order) + `
		` + limitClause(limit, offset)

	rows := []VisitorRow{}
	if err := r.DB.SelectContext(ctx, &rows, q, instanceID); err != nil {
		return nil, err
	}
	return rows, nil
}

type timelineEventRow struct {
	EventType   string `db:"event_type"`
	Timestamp   string `db:"timestamp"`
	UserID      string `db:"user_id"`
	DisplayName string `db:"display_name"`
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

	conditions := []string{"instance_id = :instance_id"}
	args := map[string]interface{}{"instance_id": instanceID}
	if start != nil {
		conditions = append(conditions, "timestamp > :start")
		args["start"] = *start
	}
	if end != nil {
		conditions = append(conditions, "timestamp <= :end")
		args["end"] = *end
	}
	where := strings.Join(conditions, " AND ")

	q := `
		SELECT e.event_type, e.timestamp, e.user_id, p.display_name
		FROM events e
		JOIN players p ON p.user_id = e.user_id
		WHERE ` + where + `
		ORDER BY e.timestamp`

	stmt, err := r.DB.PrepareNamedContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	events := []timelineEventRow{}
	if err := stmt.SelectContext(ctx, &events, args); err != nil {
		return nil, err
	}

	var anchor string
	if start != nil {
		anchor = *start
	} else if len(events) > 0 {
		anchor = events[0].Timestamp
	} else {
		return []TimelinePointRow{}, nil
	}

	points := []TimelinePointRow{{
		Timestamp: anchor,
		Count:     initial,
	}}
	count := initial
	for _, ev := range events {
		if ev.EventType == "join" {
			count++
		} else {
			count--
		}
		uid := ev.UserID
		dn := ev.DisplayName
		points = append(points, TimelinePointRow{
			Timestamp:   ev.Timestamp,
			Count:       count,
			UserID:      sql.NullString{String: uid, Valid: true},
			DisplayName: sql.NullString{String: dn, Valid: true},
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
		ORDER BY e.timestamp ` + orderUpper(order) + `
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
	args := map[string]interface{}{"instance_id": instanceID}
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
		       s.is_estimated_leave
		FROM sessions s
		JOIN players p ON p.user_id = s.user_id
		LEFT JOIN player_discord pd ON pd.user_id = s.user_id
		WHERE ` + where + `
		ORDER BY ` + sortCol + ` ` + orderUpper(order) + `
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

// GetInstanceStats returns instance-level aggregate counters in a single round
// trip. Every sub-aggregate is an index-bounded scan over one instance's rows
// (events / sessions are indexed by instance_id), so it stays light. The
// heaviest piece is peak_concurrent, computed as the running max of a join(+1)
// / leave(-1) delta stream over that instance's sessions.
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

// GetInstanceDiscordMentions returns the Discord IDs of currently-present
// players (open sessions) that have a registered Discord ID. One open session
// per user is guaranteed by uq_sessions_open, so the result has no duplicates.
func (r *LocationsRepo) GetInstanceDiscordMentions(ctx context.Context, instanceID int) ([]string, error) {
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
