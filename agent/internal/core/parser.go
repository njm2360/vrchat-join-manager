package core

import (
	"log"
	"maps"
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

type player struct {
	name       string
	userID     string
	internalID int
	restored   bool
}

type instance struct {
	locationID string
	local      *player
	players    map[string]*player

	pendingRestored []string
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
	name := s.pendingRestored[0]
	s.pendingRestored = s.pendingRestored[1:]
	return name, true
}

func (s *instance) removePendingRestored(name string) {
	if i := slices.Index(s.pendingRestored, name); i >= 0 {
		s.pendingRestored = slices.Delete(s.pendingRestored, i, i+1)
	}
}

func (s *instance) localRestored() bool {
	return s.local != nil && s.local.restored
}

type apiClient interface {
	SendEvent(event, locationID, name, userID string, internalID int, ts time.Time, estimated bool)
	GetPotentialSessions(locationID string) ([]PotentialSession, error)
	ResumeInstance(locationID string, userIDs []string) error
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
	if m := rePlayerJoin.FindStringSubmatch(line); m != nil {
		p.handlePlayerJoin(m[1], m[2])
		return
	}
	if m := rePlayerAPI.FindStringSubmatch(line); m != nil {
		p.handlePlayerAPI(m[1], m[2] == "local")
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
func (p *LogParser) handlePlayerAPI(name string, isLocal bool) {
	pl := p.instance.player(name)
	if isLocal {
		p.instance.local = pl
	}
	p.instance.pendingRestored = append(p.instance.pendingRestored, name)
}

// 3rd [Behaviour] Restored player {InternalID}
// Note: 異常Joinが起きた人はこれは出ない
func (p *LogParser) handleRestored(internalID int, ts time.Time) {
	name, ok := p.instance.popPendingRestored()
	if !ok {
		return
	}
	pl := p.instance.player(name)
	pl.internalID = internalID
	pl.restored = true

	// 1stが抜けない限り無いが念のためガード
	if pl.userID == "" {
		return
	}

	s := p.instance
	switch {
	case pl == s.local:
		// 自分自身のイベント
		p.flushPreJoins(ts)
		p.sendJoin(pl, ts)
	case !s.localRestored():
		// 自分より前に滞在しているプレイヤー
		s.preJoins = append(s.preJoins, pl)
	default:
		// 自分のRestore後は通常送信
		p.sendJoin(pl, ts)
	}
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

func (p *LogParser) flushPreJoins(ts time.Time) {
	s := p.instance
	if len(s.preJoins) == 0 {
		return
	}
	potential, err := p.api.GetPotentialSessions(s.locationID)
	if err != nil {
		log.Printf("ERROR GetPotentialSessions failed: %v (treating all as new)", err)
	}
	matched := matchSessions(potential, s.preJoins)

	// Note: 既存プレイヤーの正確なJoin時間は知れないため、自分のJoin時間で投げる
	if len(matched) > 0 {
		// 同一インスタンスはインスタンスとセッションを復元する
		if err := p.api.ResumeInstance(s.locationID, slices.Collect(maps.Keys(matched))); err != nil {
			log.Printf("ERROR ResumeInstance failed: %v", err)
		}
		for _, pre := range s.preJoins {
			if _, ok := matched[pre.userID]; !ok {
				p.sendEstimatedJoin(pre, ts)
			}
		}
		log.Printf("REJOIN location=%s resumed=%d new=%d", s.locationID, len(matched), len(s.preJoins)-len(matched))
	} else {
		// 新規インスタンスは全員送信する
		for _, pre := range s.preJoins {
			p.sendEstimatedJoin(pre, ts)
		}
	}
	s.preJoins = nil
}

func matchSessions(potential []PotentialSession, preJoins []*player) map[string]struct{} {
	type key struct {
		userID     string
		internalID int
	}
	set := make(map[key]struct{}, len(potential))
	for _, e := range potential {
		set[key{e.UserId, e.InternalId}] = struct{}{}
	}
	matched := make(map[string]struct{})
	for _, pre := range preJoins {
		if _, ok := set[key{pre.userID, pre.internalID}]; ok {
			matched[pre.userID] = struct{}{}
		}
	}
	return matched
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
