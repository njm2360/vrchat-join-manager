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
	isEstimatedJoin bool,
) error {
	// 同一ユーザーのopen行が既にある場合はuq_sessions_openを壊さないよう何もしない
	var exists int
	err := tx.GetContext(ctx, &exists,
		`SELECT 1 FROM sessions
		 WHERE user_id = ? AND instance_id = ? AND leave_ts IS NULL`,
		userID, instanceID,
	)
	if err == nil {
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	// 同一(user_id, internal_id)の推定Leave行が残っていれば、Leaveの推定が
	// 誤りだった(実際には退出していなかった)とみなしてそのセッションを再開する
	var sessionID int
	err = tx.GetContext(ctx, &sessionID,
		`SELECT id FROM sessions
		 WHERE instance_id = ? AND user_id = ? AND internal_id = ?
		   AND is_estimated_leave = 1 AND leave_ts IS NOT NULL
		 ORDER BY join_ts DESC LIMIT 1`,
		instanceID, userID, internalID,
	)
	if err == nil {
		return resumeSessions(ctx, tx, []int{sessionID})
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO sessions(
		    instance_id, world_id,
		    user_id, internal_id, join_event_id, join_ts, is_estimated_join
		 )
		 VALUES(?, ?, ?, ?, ?, ?, ?)`,
		instanceID, worldID, userID, internalID, eventID, ts, isEstimatedJoin,
	)
	return err
}

func (r *EventsRepo) ResumeSessions(ctx context.Context, tx *sqlx.Tx, sessionIDs []int) error {
	return resumeSessions(ctx, tx, sessionIDs)
}

func resumeSessions(ctx context.Context, tx *sqlx.Tx, sessionIDs []int) error {
	if len(sessionIDs) == 0 {
		return nil
	}
	q, args, err := sqlx.In(
		`UPDATE sessions
		    SET leave_ts           = NULL,
		        leave_event_id     = NULL,
		        duration_seconds   = NULL,
		        is_estimated_leave = 0
		  WHERE id IN (?)`,
		sessionIDs,
	)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, q, args...)
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
