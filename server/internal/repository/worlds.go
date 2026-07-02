package repository

import (
	"context"
	"strings"

	"github.com/jmoiron/sqlx"
)

type WorldsRepo struct {
	DB *sqlx.DB
}

func NewWorldsRepo(db *sqlx.DB) *WorldsRepo { return &WorldsRepo{DB: db} }

func (r *WorldsRepo) List(ctx context.Context, start, end *string, order string, limit *int, offset int) ([]WorldRow, error) {
	var (
		having []string
		args   = map[string]interface{}{}
	)
	if start != nil {
		having = append(having, "w.created_at >= :start")
		args["start"] = *start
	}
	if end != nil {
		having = append(having, "last_seen <= :end")
		args["end"] = *end
	}
	havingClause := ""
	if len(having) > 0 {
		havingClause = "HAVING " + strings.Join(having, " AND ")
	}

	q := `
		SELECT w.world_id,
		       w.name,
		       w.created_at,
		       w.updated_at,
		       MAX(CASE WHEN s.id IS NOT NULL
		                THEN COALESCE(s.leave_ts, strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		           END) AS last_seen,
		       COUNT(s.id)    AS session_count
		FROM worlds w
		LEFT JOIN sessions s ON s.world_id = w.world_id
		GROUP BY w.world_id
		` + havingClause + `
		ORDER BY last_seen ` + orderUpper(order) + ` NULLS LAST, w.world_id ` + orderUpper(order) + `
		` + limitClause(limit, offset)

	rows := []WorldRow{}
	stmt, err := r.DB.PrepareNamedContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	if err := stmt.SelectContext(ctx, &rows, args); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *WorldsRepo) Rename(ctx context.Context, worldID, name, ts string) (bool, error) {
	res, err := r.DB.ExecContext(ctx,
		`UPDATE worlds SET name = ?, updated_at = ? WHERE world_id = ?`,
		name, ts, worldID,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *WorldsRepo) Delete(ctx context.Context, worldID string) (bool, error) {
	res, err := r.DB.ExecContext(ctx,
		`DELETE FROM worlds WHERE world_id = ?`,
		worldID,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
