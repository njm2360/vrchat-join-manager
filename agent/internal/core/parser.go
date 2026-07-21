package core

import (
	"log"
	"regexp"
	"slices"
	"strconv"
	"time"
)

var (
	reLogTime    = regexp.MustCompile(`^(\d{4}\.\d{2}\.\d{2} \d{2}:\d{2}:\d{2})`)
	reJoining    = regexp.MustCompile(`\[Behaviour\] Joining (wrld_\S+)`)
	rePlayerJoin = regexp.MustCompile(`\[Behaviour\] OnPlayerJoined (.+) \((usr_[0-9a-f\-]+)\)`)
	rePlayerLeft = regexp.MustCompile(`\[Behaviour\] OnPlayerLeft (.+) \((usr_[0-9a-f\-]+)\)`)
	rePlayerAPI  = regexp.MustCompile(`\[Behaviour\] Initialized PlayerAPI "(.+)" is (remote|local)`)
	reRestored   = regexp.MustCompile(`\[Behaviour\] Restored player (\d+)`)
	reDestroying = regexp.MustCompile(`\[Behaviour\] Destroying (.+)`)
	reOnLeftRoom = regexp.MustCompile(`\[Behaviour\] OnLeftRoom`)
)

// 異常Joinが退室せず残った場合にpendingを破棄する上限
const pendingRestoredTimeout = 60 * time.Second

type player struct {
	name       string
	userID     string
	internalID int
	restored   bool
}

type pendingRestore struct {
	name  string
	since time.Time
	// 自分のRestored前に現れた人(入場バーストの列挙)。バーストでは位置対応(FIFO)
	burst bool
}

type restoredNumber struct {
	id    int
	since time.Time
}

type instance struct {
	locationID string
	local      *player
	players    map[string]*player

	pendingRestored []pendingRestore
	restoredBuf     []restoredNumber
	preJoins        []*player

	selfLeftRoom bool
}

func newInstance(locationID string) *instance {
	return &instance{
		locationID: locationID,
		players:    make(map[string]*player),
	}
}

func (s *instance) player(name string) *player {
	pl, ok := s.players[name]
	if !ok {
		pl = &player{name: name}
		s.players[name] = pl
	}
	return pl
}

func (s *instance) popPendingRestored() (string, bool) {
	if len(s.pendingRestored) == 0 {
		return "", false
	}
	name := s.pendingRestored[0].name
	s.pendingRestored = s.pendingRestored[1:]
	return name, true
}

func (s *instance) removePendingRestored(name string) {
	if i := slices.IndexFunc(s.pendingRestored, func(e pendingRestore) bool { return e.name == name }); i >= 0 {
		s.pendingRestored = slices.Delete(s.pendingRestored, i, i+1)
	}
}

func (s *instance) localRestored() bool {
	return s.local != nil && s.local.restored
}

type apiClient interface {
	SendEvent(event, locationID, name, userID string, internalID int, ts time.Time, estimated bool)
	Checkin(locationID string, at time.Time, self ObservedPlayer, players []ObservedPlayer) ([]string, error)
	CloseLocation(locationID string, userID string, ts time.Time)
}

type LogParser struct {
	api      apiClient
	loc      *time.Location
	instance *instance
}

func NewLogParser(api *ApiClient, loc *time.Location) *LogParser {
	return &LogParser{api: api, loc: loc}
}

func (p *LogParser) parseTimestamp(line string) (time.Time, bool) {
	m := reLogTime.FindStringSubmatch(line)
	if m == nil {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006.01.02 15:04:05", m[1], p.loc)
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

func (p *LogParser) OnLine(_ string, line string) {
	ts, ok := p.parseTimestamp(line)
	if !ok {
		return
	}
	if m := reJoining.FindStringSubmatch(line); m != nil {
		p.handleJoining(m[1], ts)
		return
	}
	if p.instance == nil {
		return
	}
	p.expireRestoreTracking(ts)
	if m := rePlayerJoin.FindStringSubmatch(line); m != nil {
		p.handlePlayerJoin(m[1], m[2])
		return
	}
	if m := rePlayerAPI.FindStringSubmatch(line); m != nil {
		p.handlePlayerAPI(m[1], m[2] == "local", ts)
		return
	}
	if m := reRestored.FindStringSubmatch(line); m != nil {
		id, _ := strconv.Atoi(m[1])
		p.handleRestored(id, ts)
		return
	}
	if reOnLeftRoom.MatchString(line) {
		p.instance.selfLeftRoom = true
		return
	}
	if m := rePlayerLeft.FindStringSubmatch(line); m != nil {
		p.handlePlayerLeft(m[1], ts)
		return
	}
	if m := reDestroying.FindStringSubmatch(line); m != nil {
		p.handleDestroying(m[1], ts)
		return
	}
}

func (p *LogParser) handleJoining(locationID string, ts time.Time) {
	// Destroyingが稀に発火しないことがあるためここでもcloseを呼ぶ
	if p.instance != nil {
		p.closeLocation(ts)
	}
	p.instance = newInstance(locationID)
	log.Printf("OPEN  [%s] location=%s", p.fmtTs(ts), locationID)
}

// 1st [Behaviour] OnPlayerJoined {Name} ({UserID})
func (p *LogParser) handlePlayerJoin(name, userID string) {
	p.instance.player(name).userID = userID
}

// 2nd [Behaviour] Initialized PlayerAPI {Name} is [local|remote]
func (p *LogParser) handlePlayerAPI(name string, isLocal bool, ts time.Time) {
	pl := p.instance.player(name)
	if isLocal {
		p.instance.local = pl
	}
	p.instance.pendingRestored = append(p.instance.pendingRestored, pendingRestore{
		name:  name,
		since: ts,
		burst: !p.instance.localRestored(),
	})
}

// 3rd [Behaviour] Restored player {InternalID}
// Note: 異常Joinが起きた人はこれは出ない
func (p *LogParser) handleRestored(internalID int, ts time.Time) {
	s := p.instance

	// 入場バーストでは位置対応(FIFO)
	if len(s.pendingRestored) > 0 && s.pendingRestored[0].burst {
		name, _ := s.popPendingRestored()
		pl := s.player(name)
		pl.internalID = internalID
		pl.restored = true

		// 1stが抜けない限り無いが念のためガード
		if pl.userID == "" {
			return
		}

		switch {
		case pl == s.local:
			// 自分自身のイベント
			p.resolvePreJoins(ts)
			p.sendJoin(pl, ts)
		case !s.localRestored():
			// 自分より前に滞在しているプレイヤー
			s.preJoins = append(s.preJoins, pl)
		default:
			// 自分のRestored後に処理される入場バーストの残り
			p.sendJoin(pl, ts)
		}
		return
	}

	if len(s.pendingRestored) == 0 {
		return
	}

	s.restoredBuf = append(s.restoredBuf, restoredNumber{id: internalID, since: ts})
	p.flushRestored(ts)
}

func (p *LogParser) flushRestored(ts time.Time) {
	s := p.instance
	if s.selfLeftRoom {
		return
	}
	if len(s.restoredBuf) == 0 || len(s.restoredBuf) != len(s.pendingRestored) {
		return
	}
	if s.pendingRestored[0].burst {
		return
	}

	ids := make([]int, 0, len(s.restoredBuf))
	for _, r := range s.restoredBuf {
		ids = append(ids, r.id)
	}
	slices.Sort(ids)

	entries := s.pendingRestored
	s.pendingRestored = nil
	s.restoredBuf = nil

	for i, e := range entries {
		pl := s.player(e.name)
		pl.internalID = ids[i]
		pl.restored = true
		if pl.userID == "" {
			continue
		}
		p.sendJoin(pl, ts)
	}
}

func (p *LogParser) expireRestoreTracking(ts time.Time) {
	s := p.instance
	stale := (len(s.pendingRestored) > 0 && ts.Sub(s.pendingRestored[0].since) >= pendingRestoredTimeout) ||
		(len(s.restoredBuf) > 0 && ts.Sub(s.restoredBuf[0].since) >= pendingRestoredTimeout)
	if !stale {
		return
	}
	for _, e := range s.pendingRestored {
		log.Printf("DROP  [%s] %s (restore pending timeout)", p.fmtTs(ts), e.name)
	}
	s.pendingRestored = nil
	s.restoredBuf = nil
}

// 4th [Behaviour] OnPlayerLeft {Name} ({UserID})
// ※バツでとじるとこの行は出てこない (Destroyingは出る)
func (p *LogParser) handlePlayerLeft(name string, ts time.Time) {
	s := p.instance
	pl, ok := s.players[name]
	if !ok {
		return
	}
	// 自分のLeaveはDestroyingで送るためここでは処理しない
	if pl == s.local {
		return
	}
	s.removePendingRestored(name)
	delete(s.players, name)
	// 自分のRestored前に退室した人はpreJoinsから外す
	s.preJoins = slices.DeleteFunc(s.preJoins, func(x *player) bool { return x == pl })
	// 異常Join者の退室でpendingが減ると保留中の割当が確定しうる
	p.flushRestored(ts)
	// Restoredしてない異常プレイヤーはLeaveイベントを送信しない
	if !pl.restored {
		return
	}
	// OnLeftRoomの検知後はLeaveイベントを送らない
	// ※後に他人のOnPlayerLeftが一括で出るが推定Leaveのため送らない
	if s.selfLeftRoom {
		return
	}
	p.sendLeave(pl, ts)
}

// アプリの終了、インスタンスを抜けるケースであればどんな条件でもここが走る
func (p *LogParser) handleDestroying(name string, ts time.Time) {
	local := p.instance.local
	if local == nil || name != local.name {
		return
	}
	if local.restored {
		p.sendLeave(local, ts)
	}
	p.closeLocation(ts)
	p.instance = nil
}

func (p *LogParser) resolvePreJoins(ts time.Time) {
	s := p.instance
	if len(s.preJoins) == 0 {
		return
	}
	// サーバー側で同一インスタンスへの復帰か判定する
	self := ObservedPlayer{UserId: s.local.userID, InternalId: s.local.internalID}
	players := make([]ObservedPlayer, 0, len(s.preJoins))
	for _, pre := range s.preJoins {
		players = append(players, ObservedPlayer{UserId: pre.userID, InternalId: pre.internalID})
	}
	resumedIDs, err := p.api.Checkin(s.locationID, ts, self, players)
	if err != nil {
		// 判定不能時は全員送信する（サーバーの冪等性で保証）
		log.Printf("ERROR Checkin failed: %v (treating all as new)", err)
	}
	resumed := make(map[string]struct{}, len(resumedIDs))
	for _, id := range resumedIDs {
		resumed[id] = struct{}{}
	}

	// 既に在室している人のJoin時間は不明のため、自分の時間で投げる（推定）
	sent := 0
	for _, pre := range s.preJoins {
		// 証人はサーバー側で復元済みなので送信しない
		if _, ok := resumed[pre.userID]; !ok {
			p.sendEstimatedJoin(pre, ts)
			sent++
		}
	}
	if len(resumed) > 0 {
		log.Printf("REJOIN location=%s resumed=%d new=%d", s.locationID, len(resumed), sent)
	}
	s.preJoins = nil
}

func (p *LogParser) fmtTs(ts time.Time) string {
	return ts.In(p.loc).Format("2006-01-02 15:04:05")
}

func (p *LogParser) sendJoin(pl *player, ts time.Time) {
	log.Printf("JOIN  [%s] %s (%s) internal_id=%d", p.fmtTs(ts), pl.name, pl.userID, pl.internalID)
	p.api.SendEvent("join", p.instance.locationID, pl.name, pl.userID, pl.internalID, ts, false)
}

func (p *LogParser) sendEstimatedJoin(pl *player, ts time.Time) {
	log.Printf("JOIN~ [%s] %s (%s) internal_id=%d (estimated)", p.fmtTs(ts), pl.name, pl.userID, pl.internalID)
	p.api.SendEvent("join", p.instance.locationID, pl.name, pl.userID, pl.internalID, ts, true)
}

func (p *LogParser) sendLeave(pl *player, ts time.Time) {
	log.Printf("LEAVE [%s] %s (%s) internal_id=%d", p.fmtTs(ts), pl.name, pl.userID, pl.internalID)
	p.api.SendEvent("leave", p.instance.locationID, pl.name, pl.userID, pl.internalID, ts, false)
}

func (p *LogParser) closeLocation(ts time.Time) {
	var localUserID string
	if local := p.instance.local; local != nil {
		localUserID = local.userID
	}
	log.Printf("CLOSE [%s] location=%s", p.fmtTs(ts), p.instance.locationID)
	p.api.CloseLocation(p.instance.locationID, localUserID, ts)
}
