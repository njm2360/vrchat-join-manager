package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type GroupsRepo struct {
	DB *sqlx.DB
}

func NewGroupsRepo(db *sqlx.DB) *GroupsRepo { return &GroupsRepo{DB: db} }

func (r *GroupsRepo) List(ctx context.Context, order string, limit *int, offset int) ([]GroupRow, error) {
	q := `SELECT group_id, name, created_at, updated_at
	      FROM groups
	      ORDER BY created_at ` + orderUpper(order) + limitClause(limit, offset)

	rows := []GroupRow{}
	if err := r.DB.SelectContext(ctx, &rows, q); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *GroupsRepo) Rename(ctx context.Context, groupID, name, ts string) (bool, error) {
	res, err := r.DB.ExecContext(ctx,
		`UPDATE groups SET name = ?, updated_at = ? WHERE group_id = ?`,
		name, ts, groupID,
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

func (r *GroupsRepo) Delete(ctx context.Context, groupID string) (bool, error) {
	res, err := r.DB.ExecContext(ctx,
		`DELETE FROM groups WHERE group_id = ?`,
		groupID,
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
