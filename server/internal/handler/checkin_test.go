package handler

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/njm2360/vrchat-join-manager/server/internal/db"
	"github.com/njm2360/vrchat-join-manager/server/internal/gen"
)

const testLocation = "wrld_test:12345~region(jp)"

func newTestServer(t *testing.T) *Server {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return New(conn)
}

func sendJoin(t *testing.T, s *Server, userID string, internalID int, ts time.Time) {
	t.Helper()
	_, err := s.ReceiveEvent(context.Background(), gen.ReceiveEventRequestObject{
		Body: &gen.ReceiveEventJSONRequestBody{
			Event:      gen.PlayerEventEventJoin,
			LocationId: testLocation,
			UserId:     userID,
			Name:       userID,
			InternalId: internalID,
			Timestamp:  ts,
		},
	})
	if err != nil {
		t.Fatalf("join %s: %v", userID, err)
	}
}

func closeLocation(t *testing.T, s *Server, selfUserID string, ts time.Time) {
	t.Helper()
	_, err := s.CloseLocation(context.Background(), gen.CloseLocationRequestObject{
		LocationId: testLocation,
		Body:       &gen.CloseLocationJSONRequestBody{At: ts, UserId: &selfUserID},
	})
	if err != nil {
		t.Fatalf("close location: %v", err)
	}
}

func checkin(t *testing.T, s *Server, at time.Time, self gen.ObservedPlayer, players []gen.ObservedPlayer) gen.CheckinLocation200JSONResponse {
	t.Helper()
	resp, err := s.CheckinLocation(context.Background(), gen.CheckinLocationRequestObject{
		LocationId: testLocation,
		Body:       &gen.CheckinLocationJSONRequestBody{At: at, Self: self, Players: players},
	})
	if err != nil {
		t.Fatalf("checkin: %v", err)
	}
	out, ok := resp.(gen.CheckinLocation200JSONResponse)
	if !ok {
		t.Fatalf("checkin response = %T, want 200", resp)
	}
	return out
}

// 自分がJoinした後のチェックインで、直前に閉じたインスタンスと
// 証人(推定Leaveで閉じられた他プレイヤー)のセッションが復元される
func TestCheckinLocationResumesRecentInstance(t *testing.T) {
	s := newTestServer(t)
	base := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	sendJoin(t, s, "usr_self", 1, base)
	sendJoin(t, s, "usr_a", 2, base.Add(time.Minute))
	sendJoin(t, s, "usr_b", 3, base.Add(2*time.Minute))
	closeLocation(t, s, "usr_self", base.Add(10*time.Minute))

	out := checkin(t, s, base.Add(15*time.Minute),
		gen.ObservedPlayer{UserId: "usr_self", InternalId: 4},
		[]gen.ObservedPlayer{
			{UserId: "usr_a", InternalId: 2},
			{UserId: "usr_b", InternalId: 3},
		},
	)
	if !out.Resumed {
		t.Fatalf("Resumed = false, want true (user ids: %v)", out.ResumedUserIds)
	}
	if len(out.ResumedUserIds) != 2 {
		t.Fatalf("ResumedUserIds = %v, want usr_a and usr_b", out.ResumedUserIds)
	}

	// インスタンスがopenに戻り、証人が在室扱いになっていること
	tx, err := s.DB.BeginTxx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	openID, err := s.Instances.GetOpenInstanceID(context.Background(), tx, testLocation)
	if err != nil || openID == nil {
		t.Fatalf("open instance after resume: id=%v err=%v", openID, err)
	}
	var present int
	if err := tx.GetContext(context.Background(), &present,
		`SELECT COUNT(*) FROM sessions WHERE instance_id = ? AND leave_ts IS NULL`, *openID,
	); err != nil {
		t.Fatal(err)
	}
	if present != 2 {
		t.Fatalf("open sessions = %d, want 2", present)
	}
}

// internal_id が巻き戻っている場合は復元しない
func TestCheckinLocationRejectsResetInstance(t *testing.T) {
	s := newTestServer(t)
	base := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	sendJoin(t, s, "usr_self", 1, base)
	sendJoin(t, s, "usr_a", 2, base.Add(time.Minute))
	closeLocation(t, s, "usr_self", base.Add(10*time.Minute))

	out := checkin(t, s, base.Add(15*time.Minute),
		gen.ObservedPlayer{UserId: "usr_self", InternalId: 1},
		[]gen.ObservedPlayer{{UserId: "usr_a", InternalId: 2}},
	)
	if out.Resumed {
		t.Fatalf("Resumed = true, want false")
	}
}

// 復元ウィンドウより古いインスタンスは復元しない
func TestCheckinLocationRejectsStaleInstance(t *testing.T) {
	s := newTestServer(t)
	base := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	sendJoin(t, s, "usr_self", 1, base)
	sendJoin(t, s, "usr_a", 2, base.Add(time.Minute))
	closeLocation(t, s, "usr_self", base.Add(10*time.Minute))

	out := checkin(t, s, base.Add(8*24*time.Hour),
		gen.ObservedPlayer{UserId: "usr_self", InternalId: 3},
		[]gen.ObservedPlayer{{UserId: "usr_a", InternalId: 2}},
	)
	if out.Resumed {
		t.Fatalf("Resumed = true, want false")
	}
}

// Rejoinした際に、推定Leaveで閉じたセッションが再開される
func TestReceiveEventJoinResumesEstimatedLeave(t *testing.T) {
	s := newTestServer(t)
	base := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	sendJoin(t, s, "usr_self", 1, base)
	sendJoin(t, s, "usr_a", 2, base.Add(time.Minute))
	closeLocation(t, s, "usr_self", base.Add(10*time.Minute))

	out := checkin(t, s, base.Add(15*time.Minute),
		gen.ObservedPlayer{UserId: "usr_self", InternalId: 3},
		[]gen.ObservedPlayer{{UserId: "usr_a", InternalId: 2}},
	)
	if !out.Resumed {
		t.Fatal("precondition: checkin should resume")
	}
	sendJoin(t, s, "usr_a", 2, base.Add(16*time.Minute))

	var count int
	if err := s.DB.GetContext(context.Background(), &count,
		`SELECT COUNT(*) FROM sessions WHERE user_id = 'usr_a'`,
	); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("sessions for usr_a = %d, want 1 (resumed, not duplicated)", count)
	}
}
