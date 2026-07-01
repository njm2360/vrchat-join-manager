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

type resumeCall struct {
	locationID string
	userIDs    []string
}

type closeCall struct {
	locationID string
	userID     string
	ts         time.Time
}

type fakeAPI struct {
	sendEvents   []sendEventCall
	potential    []PotentialSession
	potentialErr error
	getCalls     []string
	resumes      []resumeCall
	resumeErr    error
	closes       []closeCall
}

func (f *fakeAPI) SendEvent(event, locationID, name, userID string, internalID int, ts time.Time, estimated bool) {
	f.sendEvents = append(f.sendEvents, sendEventCall{event, locationID, name, userID, internalID, ts, estimated})
}

func (f *fakeAPI) GetPotentialSessions(locationID string) ([]PotentialSession, error) {
	f.getCalls = append(f.getCalls, locationID)
	return f.potential, f.potentialErr
}

func (f *fakeAPI) ResumeInstance(locationID string, userIDs []string) error {
	cp := append([]string(nil), userIDs...)
	f.resumes = append(f.resumes, resumeCall{locationID, cp})
	return f.resumeErr
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

func TestMatchSessions(t *testing.T) {
	potential := []PotentialSession{
		{UserId: "usr_a", InternalId: 1},
		{UserId: "usr_b", InternalId: 2},
	}
	pre := []*player{
		{name: "A", userID: "usr_a", internalID: 1},
		{name: "B", userID: "usr_b", internalID: 99}, // internal mismatch
		{name: "C", userID: "usr_c", internalID: 3},  // not in potential
	}
	got := matchSessions(potential, pre)
	if _, ok := got["usr_a"]; !ok {
		t.Errorf("expected usr_a matched")
	}
	if _, ok := got["usr_b"]; ok {
		t.Errorf("usr_b should not match (internal id differs)")
	}
	if _, ok := got["usr_c"]; ok {
		t.Errorf("usr_c should not match")
	}
}

func TestMatchSessionsEmpty(t *testing.T) {
	got := matchSessions(nil, []*player{{name: "A", userID: "usr_a", internalID: 1}})
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
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
	api.potential = nil // no resume → all preJoins are sent as new

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

	wantTs := mustTime(t, "2024.01.01 00:00:09") // Alice's Restored triggers flushPreJoins.
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
	if len(api.getCalls) != 0 {
		t.Errorf("GetPotentialSessions should not be called when no pre-joins")
	}
}

func TestOnLine_PreExistingPlayers_ResumeMatched(t *testing.T) {
	p, api := newTestParser()
	api.potential = []PotentialSession{{UserId: "usr_bbbb", InternalId: 1}}

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

	if len(api.getCalls) != 1 || api.getCalls[0] != "wrld_x:1" {
		t.Fatalf("GetPotentialSessions calls = %v", api.getCalls)
	}
	if len(api.resumes) != 1 {
		t.Fatalf("ResumeInstance calls = %d want 1", len(api.resumes))
	}
	if api.resumes[0].locationID != "wrld_x:1" {
		t.Errorf("resume location = %q", api.resumes[0].locationID)
	}
	if !reflect.DeepEqual(api.resumes[0].userIDs, []string{"usr_bbbb"}) {
		t.Errorf("resume userIDs = %v want [usr_bbbb]", api.resumes[0].userIDs)
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
	api.potential = nil // empty

	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		"2024.01.01 00:00:03 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:04 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		"2024.01.01 00:00:05 [Behaviour] Restored player 1",
		"2024.01.01 00:00:06 [Behaviour] Restored player 2",
	)

	if len(api.resumes) != 0 {
		t.Errorf("ResumeInstance must not be called when nothing matched")
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

func TestOnLine_PreExistingPlayers_GetPotentialError_TreatsAllAsNew(t *testing.T) {
	p, api := newTestParser()
	api.potentialErr = errors.New("boom")

	feed(p,
		"2024.01.01 00:00:00 [Behaviour] Joining wrld_x:1",
		"2024.01.01 00:00:01 [Behaviour] OnPlayerJoined Bob (usr_bbbb)",
		`2024.01.01 00:00:02 [Behaviour] Initialized PlayerAPI "Bob" is remote`,
		"2024.01.01 00:00:03 [Behaviour] OnPlayerJoined Alice (usr_aaaa)",
		`2024.01.01 00:00:04 [Behaviour] Initialized PlayerAPI "Alice" is local`,
		"2024.01.01 00:00:05 [Behaviour] Restored player 1",
		"2024.01.01 00:00:06 [Behaviour] Restored player 2",
	)
	if len(api.resumes) != 0 {
		t.Errorf("must not resume when potential lookup failed")
	}
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
		if n == "Bob" {
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
	if len(api.sendEvents)+len(api.closes)+len(api.resumes) != 0 {
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
