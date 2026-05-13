package repository

import (
	"context"
	"strings"

	"github.com/jmoiron/sqlx"
)

type PlayersRepo struct {
	DB *sqlx.DB
}

func NewPlayersRepo(db *sqlx.DB) *PlayersRepo { return &PlayersRepo{DB: db} }

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
