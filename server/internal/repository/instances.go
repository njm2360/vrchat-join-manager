package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

type InstancesRepo struct {
	DB *sqlx.DB
}

func NewInstancesRepo(db *sqlx.DB) *InstancesRepo { return &InstancesRepo{DB: db} }

// GetOrCreate returns the open instance id for the location, creating one if none exists.
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

// GetOpenInstanceID returns the open instance id for a location, if any.
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

type PotentialSession struct {
	UserID     string `db:"user_id"`
	InternalID int    `db:"internal_id"`
}

func (r *InstancesRepo) GetPotentialSessions(ctx context.Context, locationID string) ([]PotentialSession, error) {
	var latestID int
	if err := r.DB.GetContext(ctx, &latestID,
		`SELECT id FROM instances
		 WHERE location_id = ? AND closed_at IS NOT NULL
		 ORDER BY closed_at DESC LIMIT 1`,
		locationID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []PotentialSession{}, nil
		}
		return nil, err
	}

	rows := []PotentialSession{}
	if err := r.DB.SelectContext(ctx, &rows,
		`SELECT user_id, internal_id FROM sessions WHERE instance_id = ? AND is_estimated_leave = 1`,
		latestID,
	); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *InstancesRepo) Resume(ctx context.Context, locationID string, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var openID int
	err = tx.GetContext(ctx, &openID,
		`SELECT id FROM instances WHERE location_id = ? AND closed_at IS NULL`,
		locationID,
	)
	if err == nil {
		return tx.Commit()
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	var latestID int
	if err := tx.GetContext(ctx, &latestID,
		`SELECT id FROM instances
		 WHERE location_id = ? AND closed_at IS NOT NULL
		 ORDER BY closed_at DESC LIMIT 1`,
		locationID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return tx.Commit()
		}
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE instances SET closed_at = NULL WHERE id = ?`,
		latestID,
	); err != nil {
		return err
	}
	for _, uid := range userIDs {
		if _, err := tx.ExecContext(ctx,
			`UPDATE sessions
			    SET leave_ts           = NULL,
			        duration_seconds   = NULL,
			        is_estimated_leave = 0
			  WHERE user_id     = ?
			    AND instance_id = ?
			    AND is_estimated_leave = 1`,
			uid, latestID,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
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

// CloseLocationSessions closes all open sessions for an instance and marks
// the instance itself as closed. The caller owns the transaction so the
// preceding lookup and these updates land atomically.
func (r *InstancesRepo) CloseLocationSessions(ctx context.Context, tx *sqlx.Tx, instanceID int, ts string, selfUserID *string) (int, error) {
	res, err := tx.ExecContext(ctx,
		`UPDATE sessions
		    SET leave_ts           = ?,
		        duration_seconds   = CAST(ROUND((julianday(?) - julianday(join_ts)) * 86400) AS INTEGER),
		        is_estimated_leave = CASE WHEN user_id = ? THEN 0 ELSE 1 END
		  WHERE instance_id = ?
		    AND leave_ts IS NULL`,
		ts, ts, nullableString(selfUserID), instanceID,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()

	if _, err := tx.ExecContext(ctx,
		`UPDATE instances SET closed_at = ? WHERE id = ?`,
		ts, instanceID,
	); err != nil {
		return 0, err
	}

	return int(n), nil
}

func nullableString(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}
