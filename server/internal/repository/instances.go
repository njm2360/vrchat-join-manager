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

type ObservedPlayer struct {
	UserID     string
	InternalID int
}

func (r *InstancesRepo) Checkin(ctx context.Context, locationID, at string, self ObservedPlayer, players []ObservedPlayer) ([]string, error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	none := func() ([]string, error) {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return []string{}, nil
	}

	// openなインスタンスが残っている場合は復元しない
	var openID int
	err = tx.GetContext(ctx, &openID,
		`SELECT id FROM instances WHERE location_id = ? AND closed_at IS NULL`,
		locationID,
	)
	if err == nil {
		return none()
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// LocationIDで最後に閉じたインスタンスを探す
	var cand struct {
		ID       int    `db:"id"`
		OpenedAt string `db:"opened_at"`
	}
	if err := tx.GetContext(ctx, &cand,
		`SELECT id, opened_at FROM instances
		 WHERE location_id = ? AND closed_at IS NOT NULL
		 ORDER BY closed_at DESC LIMIT 1`,
		locationID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return none()
		}
		return nil, err
	}

	// 7日以上前に開かれたインスタンスは除外する
	var ageDays float64
	if err := tx.GetContext(ctx, &ageDays,
		`SELECT julianday(?) - julianday(?)`,
		at, cand.OpenedAt,
	); err != nil {
		return nil, err
	}
	if ageDays < 0 || ageDays >= 7 {
		return none()
	}

	// Note: internal_idはインスタンス生存中は巻き戻らない
	var maxInternalId int
	if err := tx.GetContext(ctx, &maxInternalId,
		`SELECT COALESCE(MAX(internal_id), 0) FROM sessions WHERE instance_id = ?`,
		cand.ID,
	); err != nil {
		return nil, err
	}

	// 自分のIDが記録上の最大を超えていなければ別インスタンス
	// ※インスタンスリセットならほぼここで弾ける
	if self.InternalID <= maxInternalId {
		return none()
	}

	type witness struct {
		userID    string
		sessionID int
	}
	witnesses := []witness{}
	mismatches := 0
	seen := map[string]struct{}{self.UserID: {}}
	for _, p := range players {
		if _, dup := seen[p.UserID]; dup {
			continue
		}
		seen[p.UserID] = struct{}{}

		// 不在中に採番されたIDは記録に存在し得ないため照合しない
		if p.InternalID > maxInternalId {
			continue
		}

		// InternalIDとUserIDがマッチする推定Leaveなセッションを探す
		var sessionID int
		err := tx.GetContext(ctx, &sessionID,
			`SELECT id FROM sessions
			 WHERE instance_id = ? AND user_id = ? AND internal_id = ?
			   AND is_estimated_leave = 1
			 ORDER BY join_ts DESC LIMIT 1`,
			cand.ID, p.UserID, p.InternalID,
		)
		if err == nil {
			witnesses = append(witnesses, witness{p.UserID, sessionID})
			continue
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}

		// 既知ユーザーの低ID不一致は再採番かID取り違えの証拠
		// ※未知ユーザーは記録漏れがありうるため許容
		var known int
		if err := tx.GetContext(ctx, &known,
			`SELECT COUNT(*) FROM sessions WHERE instance_id = ? AND user_id = ?`,
			cand.ID, p.UserID,
		); err != nil {
			return nil, err
		}
		if known > 0 {
			mismatches++
		}
	}

	// 一致が不一致を上回る場合のみ復元する（同数は安全側で新規扱い）
	if mismatches >= len(witnesses) {
		return none()
	}

	// ↓↓↓ 同一インスタンス確定 ↓↓↓

	// インスタンスを復元
	if _, err := tx.ExecContext(ctx,
		`UPDATE instances SET closed_at = NULL WHERE id = ?`,
		cand.ID,
	); err != nil {
		return nil, err
	}
	// 証人のセッションを復元する
	resumed := make([]string, 0, len(witnesses))
	for _, w := range witnesses {
		if _, err := tx.ExecContext(ctx,
			`UPDATE sessions
			    SET leave_ts           = NULL,
			        leave_event_id     = NULL,
			        duration_seconds   = NULL,
			        is_estimated_leave = 0
			  WHERE id = ?`,
			w.sessionID,
		); err != nil {
			return nil, err
		}
		resumed = append(resumed, w.userID)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return resumed, nil
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
