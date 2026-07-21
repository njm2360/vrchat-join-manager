package core

import (
	"errors"
	"io"
	"log"
	"reflect"
	"sort"
	"testing"
	"time"
)

type sendEventCall struct {
	event      string
	locationID string
	name       string
	userID     string
	internalID int
	ts         time.Time
	estimated  bool
}

type checkinCall struct {
	locationID string
	at         time.Time
	self       ObservedPlayer
	players    []ObservedPlayer
}

type closeCall struct {
	locationID string
	userID     string
	ts         time.Time
}

type fakeAPI struct {
	sendEvents     []sendEventCall
	checkins       []checkinCall
	checkinResumed []string
	checkinErr     error
	closes         []closeCall
}

func (f *fakeAPI) SendEvent(event, locationID, name, userID string, internalID int, ts time.Time, estimated bool) {
	f.sendEvents = append(f.sendEvents, sendEventCall{event, locationID, name, userID, internalID, ts, estimated})
}

func (f *fakeAPI) Checkin(locationID string, at time.Time, self ObservedPlayer, players []ObservedPlayer) ([]string, error) {
	cp := append([]ObservedPlayer(nil), players...)
	f.checkins = append(f.checkins, checkinCall{locationID, at, self, cp})
	return f.checkinResumed, f.checkinErr
}

func (f *fakeAPI) CloseLocation(locationID string, userID string, ts time.Time) {
	f.closes = append(f.closes, closeCall{locationID, userID, ts})
}

func newTestParser() (*LogParser, *fakeAPI) {
	api := &fakeAPI{}
	p := &LogParser{api: api, loc: time.UTC}
	return p, api
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	tt, err := time.ParseInLocation("2006.01.02 15:04:05", s, time.UTC)
	if err != nil {
		t.Fatalf("mustTime: %v", err)
	}
	return tt.UTC()
}

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	m.Run()
}

func TestParseTimestamp(t *testing.T) {
	p, _ := newTestParser()

	tests := []struct {
		name   string
		line   string
		want   time.Time
		wantOK bool
	}{
		{"valid", "2024.01.02 03:04:05 something", mustTime(t, "2024.01.02 03:04:05"), true},
		{"no timestamp", "no leading timestamp", time.Time{}, false},
		{"empty", "", time.Time{}, false},
		{"bad date", "2024.13.40 99:99:99 stuff", time.Time{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := p.parseTimestamp(tc.line)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v want %v", ok, tc.wantOK)
			}
			if ok && !got.Equal(tc.want) {
				t.Fatalf("time = %v want %v", got, tc.want)
			}
		})
	}
}

func TestParseTimestampConvertsToUTC(t *testing.T) {
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Skipf("Asia/Tokyo unavailable: %v", err)
	}
	p := &LogParser{api: &fakeAPI{}, loc: jst}
	got, ok := p.parseTimestamp("2024.01.02 12:00:00 something")
	if !ok {
		t.Fatal("expected ok")
	}
	want := time.Date(2024, 1, 2, 3, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if got.Location() != time.UTC {
		t.Fatalf("location = %v want UTC", got.Location())
	}
}

// helper to feed several full log lines.
func feed(p *LogParser, lines ...string) {
	for _, l := range lines {
		p.OnLine("test.log", l)
	}
}

// Verify FIFO of pendingRestored: with 3 players (2 pre-existing remotes + self),
// each Restored line is paired with the player whose PlayerAPI fired in the same
// order, producing the expected (name, userID, internalID) tuple per join event.
func TestOnLine_NameUserIDInternalIDMapping_ThreePlayers(t *testing.T) {
	p, api := newTestParser()
	api.checkinResumed = nil // no resume → all preJoins are sent as new

	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		// Pre-existing Bob.
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		// Pre-existing Carol.
		"2024.01.01 00:00:03 [Behaviour] OnPlayerJoined Carol (usr_cccc)",
		`2024.01.01 00:00:04 [Behaviour] Initialized PlayerAPI "Carol" is remote`,
		// Self.
		"2024.01.01 00:00:05 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:06 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		// Restores in PlayerAPI order: Bob → Carol → Alice.
		"2024.01.01 00:00:07 [Behaviour] Restored player 10",
		"2024.01.01 00:00:08 [Behaviour] Restored player 20",
		"2024.01.01 00:00:09 [Behaviour] Restored player 30",
	)

	wantTs := mustTime(t, "2024.01.01 00:00:09") // Alice's Restored triggers resolvePreJoins.
	wantEvents := []sendEventCall{
		{"join", "wrld_x:1", "Bob", "usr_bbbb", 10, wantTs, true},
		{"join", "wrld_x:1", "Carol", "usr_cccc", 20, wantTs, true},
		{"join", "wrld_x:1", "Alice", "usr_aaaa", 30, wantTs, false},
	}
	if !reflect.DeepEqual(api.sendEvents, wantEvents) {
		t.Fatalf("sendEvents mismatch\n got: %+v\nwant: %+v", api.sendEvents, wantEvents)
	}

	// Internal state mirrors the same mapping.
	wantPlayers := map[string]struct {
		userID     string
		internalID int
	}{
		"Alice": {"usr_aaaa", 30},
		"Bob":   {"usr_bbbb", 10},
		"Carol": {"usr_cccc", 20},
	}
	for name, w := range wantPlayers {
		pl, ok := p.instance.players[name]
		if !ok {
			t.Errorf("%s missing from players map", name)
			continue
		}
		if pl.userID != w.userID || pl.internalID != w.internalID || !pl.restored {
			t.Errorf("%s = {userID:%q internalID:%d restored:%v} want {%q %d true}",
				name, pl.userID, pl.internalID, pl.restored, w.userID, w.internalID)
		}
	}
	if len(p.instance.pendingRestored) != 0 {
		t.Errorf("pendingRestored should be drained, got %v", p.instance.pendingRestored)
	}
}

func TestOnLine_JoiningOpensInstance(t *testing.T) {
	p, _ := newTestParser()
	feed(p, "2024.01.01 00:00:00 [Behaviour] Joining wrld_abc:1234")
	if p.instance == nil {
		t.Fatal("instance not opened")
	}
	if p.instance.locationID != "wrld_abc:1234" {
		t.Fatalf("locationID = %q", p.instance.locationID)
	}
}

func TestOnLine_LocalPlayerFullFlow_NewInstance(t *testing.T) {
	p, api := newTestParser()
	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		"2024.01.01 00:00:03 [Behaviour] Restored player 42",
	)
	if len(api.sendEvents) != 1 {
		t.Fatalf("sendEvents = %d want 1: %+v", len(api.sendEvents), api.sendEvents)
	}
	got := api.sendEvents[0]
	want := sendEventCall{"join", "wrld_x:1", "Alice", "usr_aaaa", 42, mustTime(t, "2024.01.01 00:00:03"), false}
	if got != want {
		t.Fatalf("event = %+v want %+v", got, want)
	}
	if len(api.checkins) != 0 {
		t.Errorf("Checkin should not be called when no pre-joins")
	}
}

func TestOnLine_PreExistingPlayers_ResumeMatched(t *testing.T) {
	p, api := newTestParser()
	api.checkinResumed = []string{"usr_bbbb"}

	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		// Bob is already in the world (appears first).
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		// Now self.
		"2024.01.01 00:00:03 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:04 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		// Restores arrive in pendingRestored order (Bob then Alice).
		"2024.01.01 00:00:05 [Behaviour] Restored player 1",
		"2024.01.01 00:00:06 [Behaviour] Restored player 2",
	)

	if len(api.checkins) != 1 {
		t.Fatalf("Checkin calls = %d want 1", len(api.checkins))
	}
	call := api.checkins[0]
	if call.locationID != "wrld_x:1" {
		t.Errorf("checkin location = %q", call.locationID)
	}
	// 自分のRestored時刻・自分のペア・観測名簿が渡ること
	if !call.at.Equal(mustTime(t, "2024.01.01 00:00:06")) {
		t.Errorf("checkin at = %v", call.at)
	}
	if call.self != (ObservedPlayer{UserId: "usr_aaaa", InternalId: 2}) {
		t.Errorf("checkin self = %+v", call.self)
	}
	if !reflect.DeepEqual(call.players, []ObservedPlayer{{UserId: "usr_bbbb", InternalId: 1}}) {
		t.Errorf("checkin players = %+v", call.players)
	}
	// Bob is matched (resumed) → not sent. Only Alice's join is sent.
	for _, ev := range api.sendEvents {
		if ev.userID == "usr_bbbb" {
			t.Errorf("matched player Bob must not be sent: %+v", ev)
		}
	}
	if len(api.sendEvents) != 1 {
		t.Fatalf("sendEvents = %+v", api.sendEvents)
	}
	if api.sendEvents[0].name != "Alice" || api.sendEvents[0].event != "join" {
		t.Errorf("expected only Alice join, got %+v", api.sendEvents[0])
	}
}

func TestOnLine_PreExistingPlayers_NoMatch_SendsAll(t *testing.T) {
	p, api := newTestParser()
	api.checkinResumed = nil // server decided: new instance

	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		"2024.01.01 00:00:03 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:04 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		"2024.01.01 00:00:05 [Behaviour] Restored player 1",
		"2024.01.01 00:00:06 [Behaviour] Restored player 2",
	)

	if len(api.checkins) != 1 {
		t.Errorf("Checkin calls = %d want 1", len(api.checkins))
	}
	if len(api.sendEvents) != 2 {
		t.Fatalf("sendEvents = %d want 2: %+v", len(api.sendEvents), api.sendEvents)
	}
	// Both should be joins; Bob uses Alice's restore timestamp (the doc says so).
	names := []string{api.sendEvents[0].name, api.sendEvents[1].name}
	sort.Strings(names)
	if !reflect.DeepEqual(names, []string{"Alice", "Bob"}) {
		t.Errorf("names = %v", names)
	}
	wantTs := mustTime(t, "2024.01.01 00:00:06")
	for _, ev := range api.sendEvents {
		if !ev.ts.Equal(wantTs) {
			t.Errorf("event %s ts = %v want %v", ev.name, ev.ts, wantTs)
		}
	}
}

// A pre-existing player who leaves before our own Restored must not appear in
// the rejoin roster nor be sent as a join.
func TestOnLine_PreExistingPlayer_LeftBeforeLocalRestored_NotSent(t *testing.T) {
	p, api := newTestParser()

	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		"2024.01.01 00:00:03 [Behaviour] Restored player 1",
		"2024.01.01 00:00:04 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:05 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		// Bob leaves before Alice finishes loading.
		"2024.01.01 00:00:06 [Behaviour] OnPlayerLeft Bob (usr_bbbb)",
		"2024.01.01 00:00:07 [Behaviour] Restored player 2",
	)

	if len(api.checkins) != 0 {
		t.Errorf("Checkin should not be called when all preJoins already left: %+v", api.checkins)
	}
	// Bobのleaveは送られる(クラッシュ復旧でopenセッションが残っている場合に
	// 正しい時刻で閉じられる)が、joinとしては送られない
	if len(api.sendEvents) != 2 {
		t.Fatalf("sendEvents = %+v", api.sendEvents)
	}
	if api.sendEvents[0].event != "leave" || api.sendEvents[0].name != "Bob" {
		t.Errorf("expected Bob leave first, got %+v", api.sendEvents[0])
	}
	if api.sendEvents[1].event != "join" || api.sendEvents[1].name != "Alice" {
		t.Errorf("expected Alice join, got %+v", api.sendEvents[1])
	}
}

func TestOnLine_PreExistingPlayers_RejoinError_TreatsAllAsNew(t *testing.T) {
	p, api := newTestParser()
	api.checkinErr = errors.New("boom")

	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		"2024.01.01 00:00:03 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:04 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		"2024.01.01 00:00:05 [Behaviour] Restored player 1",
		"2024.01.01 00:00:06 [Behaviour] Restored player 2",
	)
	// 判定不能時は全員送信に倒れる (resumeされたはずの人が消えない)
	if len(api.sendEvents) != 2 {
		t.Fatalf("sendEvents = %+v", api.sendEvents)
	}
}

func TestOnLine_LateJoiner_AfterLocalRestored_SendsImmediately(t *testing.T) {
	p, api := newTestParser()

	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		"2024.01.01 00:00:03 [Behaviour] Restored player 10",
		// Later, a new player joins.
		"2024.01.01 00:01:00 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:01:01 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		"2024.01.01 00:01:02 [Behaviour] Restored player 11",
	)

	if len(api.sendEvents) != 2 {
		t.Fatalf("sendEvents = %+v", api.sendEvents)
	}
	if api.sendEvents[1].name != "Bob" || api.sendEvents[1].event != "join" {
		t.Errorf("second event = %+v", api.sendEvents[1])
	}
	if !api.sendEvents[1].ts.Equal(mustTime(t, "2024.01.01 00:01:02")) {
		t.Errorf("late joiner should use own restore ts: %v", api.sendEvents[1].ts)
	}
}

func TestOnLine_PlayerLeft_RemoteRestored_SendsLeave(t *testing.T) {
	p, api := newTestParser()
	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		"2024.01.01 00:00:03 [Behaviour] Restored player 1",
		"2024.01.01 00:01:00 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:01:01 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		"2024.01.01 00:01:02 [Behaviour] Restored player 2",
		"2024.01.01 00:02:00 [Behaviour] OnPlayerLeft Bob (usr_bbbb)",
	)
	if len(api.sendEvents) != 3 {
		t.Fatalf("sendEvents = %+v", api.sendEvents)
	}
	leave := api.sendEvents[2]
	if leave.event != "leave" || leave.name != "Bob" || leave.userID != "usr_bbbb" || leave.internalID != 2 {
		t.Errorf("leave event = %+v", leave)
	}
	if _, ok := p.instance.players["Bob"]; ok {
		t.Errorf("Bob should be removed from players map after leave")
	}
}

func TestOnLine_PlayerLeft_NotRestored_NoLeaveSent(t *testing.T) {
	p, api := newTestParser()
	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		"2024.01.01 00:00:03 [Behaviour] Restored player 1",
		// Bob joins but Restored never fires (abnormal).
		"2024.01.01 00:01:00 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:01:01 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		"2024.01.01 00:02:00 [Behaviour] OnPlayerLeft Bob (usr_bbbb)",
	)
	for _, ev := range api.sendEvents {
		if ev.event == "leave" {
			t.Errorf("unexpected leave for unrestored Bob: %+v", ev)
		}
	}
	// Bob should be removed regardless.
	if _, ok := p.instance.players["Bob"]; ok {
		t.Errorf("Bob should be removed even when leave is suppressed")
	}
	// pendingRestored should no longer contain Bob.
	for _, n := range p.instance.pendingRestored {
		if n.name == "Bob" {
			t.Errorf("Bob still pendingRestored")
		}
	}
}

func TestOnLine_PlayerLeft_AfterOnLeftRoom_Suppressed(t *testing.T) {
	p, api := newTestParser()
	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		"2024.01.01 00:00:03 [Behaviour] Restored player 1",
		"2024.01.01 00:01:00 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:01:01 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		"2024.01.01 00:01:02 [Behaviour] Restored player 2",
		"2024.01.01 00:02:00 [Behaviour] OnLeftRoom",
		"2024.01.01 00:02:01 [Behaviour] OnPlayerLeft Bob (usr_bbbb)",
	)
	for _, ev := range api.sendEvents {
		if ev.event == "leave" {
			t.Errorf("leave should be suppressed after OnLeftRoom: %+v", ev)
		}
	}
}

func TestOnLine_LocalLeftSuppressedAtOnPlayerLeft(t *testing.T) {
	// Local's OnPlayerLeft should be ignored: Destroying owns the local leave.
	p, api := newTestParser()
	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		"2024.01.01 00:00:03 [Behaviour] Restored player 1",
		"2024.01.01 00:01:00 [Behaviour] OnPlayerLeft Alice (usr_aaaa)",
	)
	for _, ev := range api.sendEvents {
		if ev.event == "leave" {
			t.Errorf("local OnPlayerLeft must not emit leave: %+v", ev)
		}
	}
	// Local still tracked as local in instance.
	if p.instance == nil || p.instance.local == nil || p.instance.local.name != "Alice" {
		t.Errorf("local should be preserved")
	}
}

// X close: other remotes are still in the room. Only local gets a leave;
// remotes are abandoned (closeLocation handles their cleanup server-side).
func TestOnLine_Destroying_Local_WithRemotes_OnlyLocalLeaves(t *testing.T) {
	p, api := newTestParser()
	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		"2024.01.01 00:00:03 [Behaviour] Restored player 1",
		"2024.01.01 00:01:00 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:01:01 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		"2024.01.01 00:01:02 [Behaviour] Restored player 2",
		"2024.01.01 00:01:03 [Behaviour] OnPlayerJoined Carol (usr_cccc)",
		`2024.01.01 00:01:04 [Behaviour] Initialized PlayerAPI "Carol" is remote`,
		"2024.01.01 00:01:05 [Behaviour] Restored player 3",
		// X close: no OnLeftRoom / OnPlayerLeft, only Destroying for local.
		"2024.01.01 00:05:00 [Behaviour] Destroying Alice",
	)
	// Joins: Alice, Bob, Carol. Leaves: Alice only.
	var leaves []sendEventCall
	for _, ev := range api.sendEvents {
		if ev.event == "leave" {
			leaves = append(leaves, ev)
		}
	}
	if len(leaves) != 1 {
		t.Fatalf("leave count = %d want 1: %+v", len(leaves), leaves)
	}
	if leaves[0].userID != "usr_aaaa" {
		t.Errorf("only local leave expected, got %+v", leaves[0])
	}
	for _, ev := range api.sendEvents {
		if ev.event == "leave" && (ev.userID == "usr_bbbb" || ev.userID == "usr_cccc") {
			t.Errorf("remote leave must not fire on X close: %+v", ev)
		}
	}
	if len(api.closes) != 1 {
		t.Fatalf("closes = %+v", api.closes)
	}
	c := api.closes[0]
	if c.locationID != "wrld_x:1" || c.userID != "usr_aaaa" || !c.ts.Equal(mustTime(t, "2024.01.01 00:05:00")) {
		t.Errorf("close = %+v", c)
	}
	if p.instance != nil {
		t.Errorf("instance should be cleared after destroying local")
	}
}

// Full world-transition flow: OnLeftRoom → mass OnPlayerLeft → mass Destroying
// others (no-op) → local Destroying (leave + close) → new Joining (new instance).
func TestOnLine_WorldTransition_FullSequence(t *testing.T) {
	p, api := newTestParser()
	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		"2024.01.01 00:00:03 [Behaviour] Restored player 1",
		"2024.01.01 00:01:00 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:01:01 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		"2024.01.01 00:01:02 [Behaviour] Restored player 2",
		"2024.01.01 00:01:03 [Behaviour] OnPlayerJoined Carol (usr_cccc)",
		`2024.01.01 00:01:04 [Behaviour] Initialized PlayerAPI "Carol" is remote`,
		"2024.01.01 00:01:05 [Behaviour] Restored player 3",
		// Transition starts.
		"2024.01.01 00:05:00 [Behaviour] OnLeftRoom",
		"2024.01.01 00:05:01 [Behaviour] OnPlayerLeft Bob (usr_bbbb)",
		"2024.01.01 00:05:02 [Behaviour] OnPlayerLeft Carol (usr_cccc)",
		"2024.01.01 00:05:03 [Behaviour] OnPlayerLeft Alice (usr_aaaa)",
		// Mass Destroying for others (no-op) then local.
		"2024.01.01 00:05:10 [Behaviour] Destroying Bob",
		"2024.01.01 00:05:11 [Behaviour] Destroying Carol",
		"2024.01.01 00:05:12 [Behaviour] Destroying Alice",
		// Next world.
		"2024.01.01 00:05:20 [Behaviour] Joining wrld_y:2",
		"2024.01.01 00:05:21 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:05:22 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		"2024.01.01 00:05:23 [Behaviour] Restored player 1",
	)

	// No leave for Bob/Carol (suppressed by selfLeftRoom).
	for _, ev := range api.sendEvents {
		if ev.event == "leave" && (ev.userID == "usr_bbbb" || ev.userID == "usr_cccc") {
			t.Errorf("remote leave must be suppressed after OnLeftRoom: %+v", ev)
		}
	}

	// Only one leave overall: local Alice from wrld_x.
	var leaves []sendEventCall
	for _, ev := range api.sendEvents {
		if ev.event == "leave" {
			leaves = append(leaves, ev)
		}
	}
	if len(leaves) != 1 {
		t.Fatalf("leave count = %d want 1: %+v", len(leaves), leaves)
	}
	if leaves[0].userID != "usr_aaaa" || leaves[0].locationID != "wrld_x:1" ||
		!leaves[0].ts.Equal(mustTime(t, "2024.01.01 00:05:12")) {
		t.Errorf("local leave = %+v", leaves[0])
	}

	// Joins: Alice/Bob/Carol@wrld_x and Alice@wrld_y.
	var joinsByLoc = map[string][]string{}
	for _, ev := range api.sendEvents {
		if ev.event == "join" {
			joinsByLoc[ev.locationID] = append(joinsByLoc[ev.locationID], ev.userID)
		}
	}
	sort.Strings(joinsByLoc["wrld_x:1"])
	if !reflect.DeepEqual(joinsByLoc["wrld_x:1"], []string{"usr_aaaa", "usr_bbbb", "usr_cccc"}) {
		t.Errorf("wrld_x joins = %v", joinsByLoc["wrld_x:1"])
	}
	if !reflect.DeepEqual(joinsByLoc["wrld_y:2"], []string{"usr_aaaa"}) {
		t.Errorf("wrld_y joins = %v", joinsByLoc["wrld_y:2"])
	}

	// closeLocation must fire exactly once for wrld_x:1, triggered by local Destroying
	// (not by the next Joining — instance was already cleared).
	if len(api.closes) != 1 {
		t.Fatalf("closes = %+v", api.closes)
	}
	if api.closes[0].locationID != "wrld_x:1" ||
		!api.closes[0].ts.Equal(mustTime(t, "2024.01.01 00:05:12")) {
		t.Errorf("close = %+v", api.closes[0])
	}

	// New instance is alive and current.
	if p.instance == nil || p.instance.locationID != "wrld_y:2" {
		t.Errorf("new instance = %+v", p.instance)
	}
	if p.instance.local == nil || p.instance.local.name != "Alice" || !p.instance.local.restored {
		t.Errorf("local in new instance not set up: %+v", p.instance.local)
	}
}

func TestOnLine_SecondJoiningClosesPrevious(t *testing.T) {
	p, api := newTestParser()
	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_a:1",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		"2024.01.01 00:00:03 [Behaviour] Restored player 1",
		"2024.01.01 00:10:00 [Behaviour] Joining wrld_b:2",
	)
	if len(api.closes) != 1 {
		t.Fatalf("closes = %+v", api.closes)
	}
	if api.closes[0].locationID != "wrld_a:1" {
		t.Errorf("expected first location closed, got %q", api.closes[0].locationID)
	}
	if p.instance == nil || p.instance.locationID != "wrld_b:2" {
		t.Errorf("new instance not opened: %+v", p.instance)
	}
}

func TestOnLine_IgnoresEventsBeforeJoining(t *testing.T) {
	p, api := newTestParser()
	feed(p,
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		"2024.01.01 00:00:03 [Behaviour] Restored player 1",
		"2024.01.01 00:00:04 [Behaviour] OnPlayerLeft Alice (usr_aaaa)",
		"2024.01.01 00:00:05 [Behaviour] Destroying Alice",
	)
	if p.instance != nil {
		t.Errorf("no instance should be created without Joining")
	}
	if len(api.sendEvents)+len(api.closes)+len(api.checkins) != 0 {
		t.Errorf("no api calls expected before Joining")
	}
}

func TestOnLine_RestoredWithoutPendingIsNoop(t *testing.T) {
	p, api := newTestParser()
	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		// Restored arrives with nothing queued.
		"2024.01.01 00:00:01 [Behaviour] Restored player 1",
	)
	if len(api.sendEvents) != 0 {
		t.Errorf("no events expected: %+v", api.sendEvents)
	}
}

// setupSelf brings the parser to steady state: instance open, self restored.
func setupSelf(t *testing.T, p *LogParser, api *fakeAPI) {
	t.Helper()
	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		"2024.01.01 00:00:03 [Behaviour] Restored player 1",
	)
	if len(api.sendEvents) != 1 || api.sendEvents[0].name != "Alice" {
		t.Fatalf("setup: self join expected, got %+v", api.sendEvents)
	}
}

// 近接入室した2人のRestoredがロード完了順で逆に出ても、pending全員分そろってから
// 昇順の番号を入室順に割り当てる。即割当だと番号を取り違える。
func TestOnLine_SteadyOverlap_OutOfOrderRestored_AssignedAscendingByJoinOrder(t *testing.T) {
	p, api := newTestParser()
	setupSelf(t, p, api)

	feed(p,
		"2024.01.01 00:01:00 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:01:00 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		"2024.01.01 00:01:03 [Behaviour] OnPlayerJoined Carol (usr_cccc)",
		`2024.01.01 00:01:03 [Behaviour] Initialized PlayerAPI "Carol" is remote`,
		"2024.01.01 00:01:04 [Behaviour] Restored player 20",
	)
	// 番号1つではどちらのものか確定しないため未送信
	if len(api.sendEvents) != 1 {
		t.Fatalf("no join must be sent while ambiguous, got %+v", api.sendEvents)
	}

	feed(p, "2024.01.01 00:01:20 [Behaviour] Restored player 10")

	if len(api.sendEvents) != 3 {
		t.Fatalf("sendEvents = %+v", api.sendEvents)
	}
	wantTs := mustTime(t, "2024.01.01 00:01:20")
	bob := api.sendEvents[1]
	carol := api.sendEvents[2]
	if bob.name != "Bob" || bob.internalID != 10 || !bob.ts.Equal(wantTs) || bob.estimated {
		t.Errorf("bob = %+v want internal_id=10", bob)
	}
	if carol.name != "Carol" || carol.internalID != 20 || !carol.ts.Equal(wantTs) || carol.estimated {
		t.Errorf("carol = %+v want internal_id=20", carol)
	}
	if len(p.instance.pendingRestored) != 0 || len(p.instance.restoredBuf) != 0 {
		t.Errorf("pending/buffer should be drained: %+v %+v", p.instance.pendingRestored, p.instance.restoredBuf)
	}
}

// 入場バースト(自分のRestored前の列挙)ではRestoredも列挙順で出るため、位置対応
// (FIFO)を保ち昇順ソートしない。番号列が非単調でもこれは取り違えではない。
func TestOnLine_EntryBurst_KeepsPositionalFIFO_NotSorted(t *testing.T) {
	p, api := newTestParser()
	api.checkinResumed = nil

	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Carol (usr_cccc)",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Dave (usr_dddd)",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Erin (usr_eeee)",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:01 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		`2024.01.01 00:00:01 [Behaviour] Initialized PlayerAPI "Carol" is remote`,
		`2024.01.01 00:00:01 [Behaviour] Initialized PlayerAPI "Dave" is remote`,
		`2024.01.01 00:00:01 [Behaviour] Initialized PlayerAPI "Erin" is remote`,
		`2024.01.01 00:00:01 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		"2024.01.01 00:00:02 [Behaviour] Restored player 94",
		"2024.01.01 00:00:02 [Behaviour] Restored player 96",
		"2024.01.01 00:00:02 [Behaviour] Restored player 65",
		"2024.01.01 00:00:02 [Behaviour] Restored player 5",
		"2024.01.01 00:00:02 [Behaviour] Restored player 100",
	)

	// 名簿は位置対応(Bob=94, Carol=96, Dave=65, Erin=5)であること
	if len(api.checkins) != 1 {
		t.Fatalf("checkins = %+v", api.checkins)
	}
	wantRoster := []ObservedPlayer{
		{UserId: "usr_bbbb", InternalId: 94},
		{UserId: "usr_cccc", InternalId: 96},
		{UserId: "usr_dddd", InternalId: 65},
		{UserId: "usr_eeee", InternalId: 5},
	}
	if !reflect.DeepEqual(api.checkins[0].players, wantRoster) {
		t.Errorf("checkin roster (positional FIFO) mismatch\n got: %+v\nwant: %+v", api.checkins[0].players, wantRoster)
	}
	if api.checkins[0].self != (ObservedPlayer{UserId: "usr_aaaa", InternalId: 100}) {
		t.Errorf("checkin self = %+v", api.checkins[0].self)
	}

	// joinも位置対応のまま(昇順ソートならBob=5になってしまう)
	byName := map[string]int{}
	for _, ev := range api.sendEvents {
		byName[ev.name] = ev.internalID
	}
	want := map[string]int{"Bob": 94, "Carol": 96, "Dave": 65, "Erin": 5, "Alice": 100}
	for name, id := range want {
		if byName[name] != id {
			t.Errorf("%s internal_id = %d want %d (burst must keep FIFO)", name, byName[name], id)
		}
	}
}

// 異常Join(Restoredが出ない)がpendingを塞いでも、その退室でpendingが減れば
// 保留中の割当が確定すること。
func TestOnLine_AbnormalJoin_LeaveRebalancesPendingAndFlushes(t *testing.T) {
	p, api := newTestParser()
	setupSelf(t, p, api)

	feed(p,
		"2024.01.01 00:01:00 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:01:00 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		// Carol: 異常Join (Restoredが出ない)
		"2024.01.01 00:01:02 [Behaviour] OnPlayerJoined Carol (usr_cccc)",
		`2024.01.01 00:01:02 [Behaviour] Initialized PlayerAPI "Carol" is remote`,
		"2024.01.01 00:01:04 [Behaviour] Restored player 11",
	)
	// pending2人に番号1つ → 未送信
	if len(api.sendEvents) != 1 {
		t.Fatalf("join must be held while pending unbalanced: %+v", api.sendEvents)
	}

	// Carol退室でpendingが1人になり割当確定
	feed(p, "2024.01.01 00:01:20 [Behaviour] OnPlayerLeft Carol (usr_cccc)")

	if len(api.sendEvents) != 2 {
		t.Fatalf("sendEvents = %+v", api.sendEvents)
	}
	bob := api.sendEvents[1]
	if bob.event != "join" || bob.name != "Bob" || bob.internalID != 11 ||
		!bob.ts.Equal(mustTime(t, "2024.01.01 00:01:20")) {
		t.Errorf("bob = %+v want join internal_id=11 at leave ts", bob)
	}
	// 未RestoredのCarolにはjoinもleaveも出ない
	for _, ev := range api.sendEvents {
		if ev.userID == "usr_cccc" {
			t.Errorf("abnormal joiner must not produce events: %+v", ev)
		}
	}
	if len(p.instance.pendingRestored) != 0 || len(p.instance.restoredBuf) != 0 {
		t.Errorf("pending/buffer should be drained")
	}
}

// 異常Joinが退室もしない場合、pendingRestoredTimeoutでグループごと破棄する。番号の
// 持ち主を特定できないため巻き添えの正常Join(Bob)も欠測になるが、誤った番号を記録
// するより安全。破棄後の新規Joinは通常どおり処理されること。
func TestOnLine_AbnormalJoin_NeverLeaves_PendingDroppedAfterTimeout(t *testing.T) {
	p, api := newTestParser()
	setupSelf(t, p, api)

	feed(p,
		// Carol: 異常Join、退室もしない
		"2024.01.01 00:01:00 [Behaviour] OnPlayerJoined Carol (usr_cccc)",
		`2024.01.01 00:01:00 [Behaviour] Initialized PlayerAPI "Carol" is remote`,
		"2024.01.01 00:01:02 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:01:02 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		"2024.01.01 00:01:05 [Behaviour] Restored player 11",
	)
	if len(api.sendEvents) != 1 {
		t.Fatalf("join must be held while pending unbalanced: %+v", api.sendEvents)
	}

	// タイムアウト経過後の行でpendingが破棄され、以降のJoinは正常処理
	feed(p,
		"2024.01.01 00:02:10 [Behaviour] OnPlayerJoined Dave (usr_dddd)",
		`2024.01.01 00:02:10 [Behaviour] Initialized PlayerAPI "Dave" is remote`,
		"2024.01.01 00:02:11 [Behaviour] Restored player 12",
	)

	if len(api.sendEvents) != 2 {
		t.Fatalf("sendEvents = %+v", api.sendEvents)
	}
	dave := api.sendEvents[1]
	if dave.name != "Dave" || dave.internalID != 12 || !dave.ts.Equal(mustTime(t, "2024.01.01 00:02:11")) {
		t.Errorf("dave = %+v want join internal_id=12", dave)
	}
	// CarolとBob(巻き添え)は送信されない
	for _, ev := range api.sendEvents {
		if ev.name == "Carol" || ev.name == "Bob" {
			t.Errorf("dropped pending must not produce events: %+v", ev)
		}
	}
	if len(p.instance.pendingRestored) != 0 || len(p.instance.restoredBuf) != 0 {
		t.Errorf("pending/buffer should be drained")
	}
}

// タイムアウト未満のロード遅延では破棄せず割り当てること。
func TestOnLine_SlowRestoreWithinLimit_StillAssigned(t *testing.T) {
	p, api := newTestParser()
	setupSelf(t, p, api)

	feed(p,
		"2024.01.01 00:01:00 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:01:00 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		// 45秒後にRestored
		"2024.01.01 00:01:45 [Behaviour] Restored player 30",
	)
	if len(api.sendEvents) != 2 {
		t.Fatalf("sendEvents = %+v", api.sendEvents)
	}
	bob := api.sendEvents[1]
	if bob.name != "Bob" || bob.internalID != 30 || !bob.ts.Equal(mustTime(t, "2024.01.01 00:01:45")) {
		t.Errorf("bob = %+v want join internal_id=30", bob)
	}
}

// 各人のJoin→API→Restoredの3行組ごと遅延して先着分が後から出る場合、各Restoredは
// その時点で唯一のpendingに確定する。グループをまたいで再ソートしないこと。
func TestOnLine_TripletShift_ConsecutiveSingles_NotResorted(t *testing.T) {
	p, api := newTestParser()
	setupSelf(t, p, api)

	feed(p,
		// 後着(番号20)の3行組が先に出る
		"2024.01.01 00:01:00 [Behaviour] OnPlayerJoined Carol (usr_cccc)",
		`2024.01.01 00:01:00 [Behaviour] Initialized PlayerAPI "Carol" is remote`,
		"2024.01.01 00:01:00 [Behaviour] Restored player 20",
		// 先着(番号10)の3行組が遅れて出る
		"2024.01.01 00:01:01 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:01:01 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		"2024.01.01 00:01:01 [Behaviour] Restored player 10",
	)

	if len(api.sendEvents) != 3 {
		t.Fatalf("sendEvents = %+v", api.sendEvents)
	}
	first := api.sendEvents[1]
	second := api.sendEvents[2]
	// Restored 20の時点でpendingはCarolのみ → 強制確定で即送信
	if first.name != "Carol" || first.internalID != 20 || !first.ts.Equal(mustTime(t, "2024.01.01 00:01:00")) {
		t.Errorf("first = %+v want Carol=20 (immediate forced assignment)", first)
	}
	if second.name != "Bob" || second.internalID != 10 {
		t.Errorf("second = %+v want Bob=10", second)
	}
}

// OnLeftRoom後にpendingが減っても保留中の割当をフラッシュしないこと。
func TestOnLine_PendingBuffer_NoFlushAfterOnLeftRoom(t *testing.T) {
	p, api := newTestParser()
	setupSelf(t, p, api)

	feed(p,
		"2024.01.01 00:01:00 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:01:00 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		"2024.01.01 00:01:01 [Behaviour] OnPlayerJoined Carol (usr_cccc)",
		`2024.01.01 00:01:01 [Behaviour] Initialized PlayerAPI "Carol" is remote`,
		"2024.01.01 00:01:05 [Behaviour] Restored player 11",
		"2024.01.01 00:02:00 [Behaviour] OnLeftRoom",
		"2024.01.01 00:02:01 [Behaviour] OnPlayerLeft Bob (usr_bbbb)",
		"2024.01.01 00:02:02 [Behaviour] OnPlayerLeft Carol (usr_cccc)",
	)

	// selfのjoin以外は送信されない
	if len(api.sendEvents) != 1 {
		t.Errorf("no events expected after OnLeftRoom: %+v", api.sendEvents)
	}
}
