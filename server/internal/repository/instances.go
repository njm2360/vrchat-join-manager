package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/njm2360/vrchat-join-manager/server/internal/domain"
)

type InstancesRepo struct {
	DB *sqlx.DB
}

func NewInstancesRepo(db *sqlx.DB) *InstancesRepo { return &InstancesRepo{DB: db} }

func (r *InstancesRepo) GetOrCreate(
	ctx context.Context,
	tx *sqlx.Tx,
	locationID, worldID string,
	instanceID, groupID, groupAccessType, region, friends, hidden, private *string,
	ts string,
) (int, error) {
	var id int
	row := tx.QueryRowxContext(ctx,
		`INSERT OR IGNORE INTO instances
			(location_id, world_id, instance_id, group_id, group_access_type, region, friends, hidden, private, opened_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 RETURNING id`,
		locationID, worldID,
		nullableString(instanceID),
		nullableString(groupID),
		nullableString(groupAccessType),
		nullableString(region),
		nullableString(friends),
		nullableString(hidden),
		nullableString(private),
		ts,
	)
	if err := row.Scan(&id); err == nil {
		return id, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	if err := tx.GetContext(ctx, &id,
		`SELECT id FROM instances WHERE location_id = ? AND closed_at IS NULL`,
		locationID,
	); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *InstancesRepo) GetOpenInstanceID(ctx context.Context, tx *sqlx.Tx, locationID string) (*int, error) {
	var id int
	q := tx.QueryRowxContext(ctx,
		`SELECT id FROM instances WHERE location_id = ? AND closed_at IS NULL`,
		locationID,
	)
	if err := q.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &id, nil
}

type ClosedInstance struct {
	ID       int    `db:"id"`
	OpenedAt string `db:"opened_at"`
}

func (r *InstancesRepo) LastClosedInstance(ctx context.Context, tx *sqlx.Tx, locationID string) (*ClosedInstance, error) {
	var row ClosedInstance
	if err := tx.GetContext(ctx, &row,
		`SELECT id, opened_at FROM instances
		 WHERE location_id = ? AND closed_at IS NOT NULL
		 ORDER BY closed_at DESC LIMIT 1`,
		locationID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *InstancesRepo) MaxInternalID(ctx context.Context, tx *sqlx.Tx, instanceID int) (int, error) {
	var maxID int
	err := tx.GetContext(ctx, &maxID,
		`SELECT COALESCE(MAX(internal_id), 0) FROM sessions WHERE instance_id = ?`,
		instanceID,
	)
	return maxID, err
}

func (r *InstancesRepo) SessionsForUsers(ctx context.Context, tx *sqlx.Tx, instanceID int, userIDs []string) ([]domain.SessionRecord, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	q, args, err := sqlx.In(
		`SELECT id, user_id, internal_id, join_ts, is_estimated_leave
		 FROM sessions
		 WHERE instance_id = ? AND user_id IN (?)`,
		instanceID, userIDs,
	)
	if err != nil {
		return nil, err
	}
	rows := []struct {
		ID               int    `db:"id"`
		UserID           string `db:"user_id"`
		InternalID       int    `db:"internal_id"`
		JoinTs           string `db:"join_ts"`
		IsEstimatedLeave bool   `db:"is_estimated_leave"`
	}{}
	if err := tx.SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, err
	}
	records := make([]domain.SessionRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, domain.SessionRecord{
			ID:               row.ID,
			UserID:           row.UserID,
			InternalID:       row.InternalID,
			JoinTs:           row.JoinTs,
			IsEstimatedLeave: row.IsEstimatedLeave,
		})
	}
	return records, nil
}

func (r *InstancesRepo) Reopen(ctx context.Context, tx *sqlx.Tx, instanceID int) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE instances SET closed_at = NULL WHERE id = ?`,
		instanceID,
	)
	return err
}

func (r *InstancesRepo) Delete(ctx context.Context, instanceID int) (bool, error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	var exists int
	if err := tx.GetContext(ctx, &exists,
		`SELECT id FROM instances WHERE id = ?`,
		instanceID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE instance_id = ?`, instanceID); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM events WHERE instance_id = ?`, instanceID); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM instances WHERE id = ?`, instanceID); err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (r *InstancesRepo) CloseInstance(ctx context.Context, tx *sqlx.Tx, instanceID int, ts string, selfUserID *string) error {
	if _, err := tx.ExecContext(ctx,
		`UPDATE sessions
		    SET leave_ts           = ?,
		        duration_seconds   = CAST(ROUND((julianday(?) - julianday(join_ts)) * 86400) AS INTEGER),
		        is_estimated_leave = CASE WHEN user_id = ? THEN 0 ELSE 1 END
		  WHERE instance_id = ?
		    AND leave_ts IS NULL`,
		ts, ts, nullableString(selfUserID), instanceID,
	); err != nil {
		return err
	}

	_, err := tx.ExecContext(ctx,
		`UPDATE instances SET closed_at = ? WHERE id = ?`,
		ts, instanceID,
	)
	return err
}

func nullableString(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}
