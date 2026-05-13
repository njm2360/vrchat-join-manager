package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

type EventsRepo struct {
	DB *sqlx.DB
}

func NewEventsRepo(db *sqlx.DB) *EventsRepo { return &EventsRepo{DB: db} }

func (r *EventsRepo) UpsertGroup(ctx context.Context, tx *sqlx.Tx, groupID, ts string) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO groups(group_id, created_at, updated_at)
		 VALUES(?, ?, ?)
		 ON CONFLICT(group_id) DO NOTHING`,
		groupID, ts, ts,
	)
	return err
}

func (r *EventsRepo) UpsertWorld(ctx context.Context, tx *sqlx.Tx, worldID, ts string) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO worlds(world_id, created_at, updated_at)
		 VALUES(?, ?, ?)
		 ON CONFLICT(world_id) DO NOTHING`,
		worldID, ts, ts,
	)
	return err
}

func (r *EventsRepo) UpsertPlayer(ctx context.Context, tx *sqlx.Tx, userID, name, ts string) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO players(user_id, display_name, created_at, updated_at)
		 VALUES(?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE
		     SET display_name = excluded.display_name,
		         updated_at   = excluded.updated_at`,
		userID, name, ts, ts,
	)
	return err
}

// InsertEvent inserts a (idempotent) event row and returns the new id, or nil
// if a duplicate row already existed.
func (r *EventsRepo) InsertEvent(
	ctx context.Context,
	tx *sqlx.Tx,
	eventType string,
	instanceID int,
	worldID, userID, locationID, ts string,
) (*int, error) {
	res, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO events(
		    event_type, instance_id, world_id,
		    user_id, location_id, timestamp
		 )
		 VALUES(?, ?, ?, ?, ?, ?)`,
		eventType, instanceID, worldID, userID, locationID, ts,
	)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	v := int(id)
	return &v, nil
}

func (r *EventsRepo) OpenSession(
	ctx context.Context,
	tx *sqlx.Tx,
	instanceID, eventID int,
	worldID, userID, ts string,
	internalID int,
) error {
	_, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO sessions(
		    instance_id, world_id,
		    user_id, internal_id, join_event_id, join_ts
		 )
		 VALUES(?, ?, ?, ?, ?, ?)`,
		instanceID, worldID, userID, internalID, eventID, ts,
	)
	return err
}

func (r *EventsRepo) CloseSession(
	ctx context.Context,
	tx *sqlx.Tx,
	userID string,
	instanceID, eventID int,
	ts string,
) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE sessions
		    SET leave_event_id   = ?,
		        leave_ts         = ?,
		        duration_seconds = CAST(ROUND((julianday(?) - julianday(join_ts)) * 86400) AS INTEGER)
		  WHERE id = (
		      SELECT id FROM sessions
		      WHERE user_id     = ?
		        AND instance_id = ?
		        AND leave_ts IS NULL
		      ORDER BY join_ts DESC
		      LIMIT 1
		  )`,
		eventID, ts, ts, userID, instanceID,
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return nil
}
