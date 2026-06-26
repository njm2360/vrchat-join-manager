package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"
)

type PlayersRepo struct {
	DB *sqlx.DB
}

func NewPlayersRepo(db *sqlx.DB) *PlayersRepo { return &PlayersRepo{DB: db} }

// SetDiscord はプレイヤーの Discord ID を登録/上書きする。
// discordID が nil または空文字の場合は登録を削除する。
// プレイヤーが存在しない場合は (false, nil) を返す。
func (r *PlayersRepo) SetDiscord(ctx context.Context, userID string, discordID *string) (bool, error) {
	var exists int
	if err := r.DB.GetContext(ctx, &exists,
		`SELECT 1 FROM players WHERE user_id = ?`, userID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	if discordID == nil || *discordID == "" {
		if _, err := r.DB.ExecContext(ctx,
			`DELETE FROM player_discord WHERE user_id = ?`, userID,
		); err != nil {
			return false, err
		}
		return true, nil
	}

	if _, err := r.DB.ExecContext(ctx,
		`INSERT INTO player_discord (user_id, discord_id)
		 VALUES (?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET discord_id = excluded.discord_id`,
		userID, *discordID,
	); err != nil {
		return false, err
	}
	return true, nil
}

// GetDetail はプレイヤーのプロフィール (Discord含む) と通算統計を1リクエストで返す。
func (r *PlayersRepo) GetDetail(ctx context.Context, userID string) (*PlayerDetailRow, error) {
	q := `
		SELECT p.user_id,
		       p.display_name,
		       pd.discord_id                                        AS discord_id,
		       p.created_at,
		       p.updated_at,
		       COUNT(s.id)                                          AS total_visits,
		       COALESCE(SUM(COALESCE(s.duration_seconds,
		           CAST(ROUND((julianday('now') - julianday(s.join_ts)) * 86400) AS INTEGER)
		       )), 0)                                               AS total_duration_seconds,
		       MIN(s.join_ts)                                       AS first_seen,
		       MAX(CASE WHEN s.id IS NOT NULL
		                THEN COALESCE(s.leave_ts, strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		           END)                                             AS last_seen,
		       EXISTS(SELECT 1 FROM sessions s2
		              WHERE s2.user_id = p.user_id AND s2.leave_ts IS NULL) AS in_room
		FROM players p
		LEFT JOIN player_discord pd ON pd.user_id = p.user_id
		LEFT JOIN sessions s         ON s.user_id = p.user_id
		WHERE p.user_id = ?
		GROUP BY p.user_id`

	var row PlayerDetailRow
	if err := r.DB.GetContext(ctx, &row, q, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *PlayersRepo) List(ctx context.Context, name *string, order string, limit *int, offset int) ([]PlayerRow, error) {
	var (
		conditions []string
		args       = map[string]interface{}{}
	)
	if name != nil {
		conditions = append(conditions, "display_name LIKE :name")
		args["name"] = "%" + *name + "%"
	}
	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	q := `
		SELECT user_id, display_name, created_at, updated_at
		FROM players
		` + where + `
		ORDER BY created_at ` + orderUpper(order) + `
		` + limitClause(limit, offset)

	stmt, err := r.DB.PrepareNamedContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows := []PlayerRow{}
	if err := stmt.SelectContext(ctx, &rows, args); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *PlayersRepo) Events(ctx context.Context, userID string, instanceID *int, start, end *string, order string, limit *int, offset int) ([]EventRow, error) {
	conditions := []string{"e.user_id = :user_id"}
	args := map[string]interface{}{"user_id": userID}
	if instanceID != nil {
		conditions = append(conditions, "instance_id = :instance_id")
		args["instance_id"] = *instanceID
	}
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

func (r *PlayersRepo) Sessions(ctx context.Context, userID string, instanceID *int, worldID, groupID, start, end *string, order string, limit *int, offset int) ([]PlayerSessionRow, error) {
	conditions := []string{"s.user_id = :user_id"}
	args := map[string]interface{}{"user_id": userID}
	if instanceID != nil {
		conditions = append(conditions, "s.instance_id = :instance_id")
		args["instance_id"] = *instanceID
	}
	if worldID != nil {
		conditions = append(conditions, "i.world_id = :world_id")
		args["world_id"] = *worldID
	}
	if groupID != nil {
		conditions = append(conditions, "i.group_id = :group_id")
		args["group_id"] = *groupID
	}
	if start != nil {
		conditions = append(conditions, "(s.leave_ts >= :start OR s.leave_ts IS NULL)")
		args["start"] = *start
	}
	if end != nil {
		conditions = append(conditions, "s.join_ts <= :end")
		args["end"] = *end
	}
	where := strings.Join(conditions, " AND ")

	q := `
		SELECT s.id, s.instance_id, i.world_id, s.join_ts, s.leave_ts,
		       COALESCE(s.duration_seconds,
		           CAST(ROUND((julianday('now') - julianday(s.join_ts)) * 86400) AS INTEGER)
		       ) AS duration_seconds,
		       s.is_estimated_leave
		FROM sessions s
		JOIN instances i ON i.id = s.instance_id
		WHERE ` + where + `
		ORDER BY s.join_ts ` + orderUpper(order) + `
		` + limitClause(limit, offset)

	stmt, err := r.DB.PrepareNamedContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows := []PlayerSessionRow{}
	if err := stmt.SelectContext(ctx, &rows, args); err != nil {
		return nil, err
	}
	return rows, nil
}
