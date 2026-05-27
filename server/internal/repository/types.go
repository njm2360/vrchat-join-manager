package repository

import (
	"database/sql"
	"strconv"
)

// Row types share these helpers across repositories.
// Nullable timestamps and strings are surfaced as sql.Null* so handlers can
// decide how to materialize them into the API response.

type GroupRow struct {
	GroupID   string         `db:"group_id"`
	Name      sql.NullString `db:"name"`
	CreatedAt string         `db:"created_at"`
	UpdatedAt string         `db:"updated_at"`
}

type WorldRow struct {
	WorldID      string         `db:"world_id"`
	Name         sql.NullString `db:"name"`
	CreatedAt    string         `db:"created_at"`
	UpdatedAt    string         `db:"updated_at"`
	LastSeen     sql.NullString `db:"last_seen"`
	SessionCount int            `db:"session_count"`
}

type PlayerRow struct {
	UserID      string `db:"user_id"`
	DisplayName string `db:"display_name"`
	CreatedAt   string `db:"created_at"`
	UpdatedAt   string `db:"updated_at"`
}

type PlayerDetailRow struct {
	UserID               string         `db:"user_id"`
	DisplayName          string         `db:"display_name"`
	DiscordID            sql.NullString `db:"discord_id"`
	CreatedAt            string         `db:"created_at"`
	UpdatedAt            string         `db:"updated_at"`
	TotalVisits          int            `db:"total_visits"`
	TotalDurationSeconds int            `db:"total_duration_seconds"`
	FirstSeen            sql.NullString `db:"first_seen"`
	LastSeen             sql.NullString `db:"last_seen"`
	InRoom               bool           `db:"in_room"`
}

type EventRow struct {
	ID          int    `db:"id"`
	EventType   string `db:"event_type"`
	InstanceID  int    `db:"instance_id"`
	WorldID     string `db:"world_id"`
	UserID      string `db:"user_id"`
	DisplayName string `db:"display_name"`
	Timestamp   string `db:"timestamp"`
}

type InstanceRow struct {
	ID              int            `db:"id"`
	LocationID      string         `db:"location_id"`
	WorldID         string         `db:"world_id"`
	WorldName       sql.NullString `db:"world_name"`
	InstanceID      sql.NullString `db:"instance_id"`
	GroupID         sql.NullString `db:"group_id"`
	GroupName       sql.NullString `db:"group_name"`
	GroupAccessType sql.NullString `db:"group_access_type"`
	Region          sql.NullString `db:"region"`
	Friends         sql.NullString `db:"friends"`
	Hidden          sql.NullString `db:"hidden"`
	Private         sql.NullString `db:"private"`
	OpenedAt        string         `db:"opened_at"`
	ClosedAt        sql.NullString `db:"closed_at"`
	UserCount       int            `db:"user_count"`
}

type SessionRow struct {
	ID               int            `db:"id"`
	InstanceID       int            `db:"instance_id"`
	UserID           string         `db:"user_id"`
	DisplayName      string         `db:"display_name"`
	DiscordID        sql.NullString `db:"discord_id"`
	JoinTs           string         `db:"join_ts"`
	LeaveTs          sql.NullString `db:"leave_ts"`
	DurationSeconds  sql.NullInt64  `db:"duration_seconds"`
	IsEstimatedLeave bool           `db:"is_estimated_leave"`
}

type LocationPlayerRow struct {
	UserID      string         `db:"user_id"`
	DisplayName string         `db:"display_name"`
	DiscordID   sql.NullString `db:"discord_id"`
	InternalID  int            `db:"internal_id"`
	JoinTs      string         `db:"join_ts"`
	JoinCount   int            `db:"join_count"`
}

type VisitorRow struct {
	UserID               string `db:"user_id"`
	DisplayName          string `db:"display_name"`
	FirstSeen            string `db:"first_seen"`
	LastSeen             string `db:"last_seen"`
	JoinCount            int    `db:"join_count"`
	TotalDurationSeconds int    `db:"total_duration_seconds"`
}

type PlayerSessionRow struct {
	ID               int            `db:"id"`
	InstanceID       int            `db:"instance_id"`
	WorldID          string         `db:"world_id"`
	JoinTs           string         `db:"join_ts"`
	LeaveTs          sql.NullString `db:"leave_ts"`
	DurationSeconds  sql.NullInt64  `db:"duration_seconds"`
	IsEstimatedLeave bool           `db:"is_estimated_leave"`
}

type TimelinePointRow struct {
	Timestamp   string         `db:"timestamp"`
	Count       int            `db:"count"`
	UserID      sql.NullString `db:"user_id"`
	DisplayName sql.NullString `db:"display_name"`
}

type DailyActiveUsersRow struct {
	Day         string `db:"day"`
	ActiveUsers int    `db:"active_users"`
}

type HourlyActiveUsersRow struct {
	Hour        string `db:"hour"`
	ActiveUsers int    `db:"active_users"`
}

type PlayerRankRow struct {
	Rank                 int    `db:"rank"`
	UserID               string `db:"user_id"`
	DisplayName          string `db:"display_name"`
	TotalDurationSeconds int    `db:"total_duration_seconds"`
	SessionCount         int    `db:"session_count"`
}

type JoinViolationRankRow struct {
	Rank           int    `db:"rank"`
	UserID         string `db:"user_id"`
	DisplayName    string `db:"display_name"`
	ViolationCount int    `db:"violation_count"`
	TotalJoins     int    `db:"total_joins"`
}

// orderUpper returns "ASC" or "DESC" given a case-insensitive order string.
func orderUpper(order string) string {
	if order == "desc" || order == "DESC" {
		return "DESC"
	}
	return "ASC"
}

// pickSortColumn maps an externally-supplied sort key to a SQL column via a
// whitelist. Defense-in-depth: prevents SQL injection if upstream enum
// validation (OpenAPI middleware) is ever bypassed or misconfigured.
func pickSortColumn(key string, cols map[string]string, defaultCol string) string {
	if c, ok := cols[key]; ok {
		return c
	}
	return defaultCol
}

// limitClause returns a "LIMIT n OFFSET m" fragment compatible with SQLite. A
// nil limit means "no upper bound" (LIMIT -1).
func limitClause(limit *int, offset int) string {
	n := -1
	if limit != nil {
		n = *limit
	}
	return " LIMIT " + strconv.Itoa(n) + " OFFSET " + strconv.Itoa(offset)
}
