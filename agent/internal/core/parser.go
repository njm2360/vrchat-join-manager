package core

import (
	"log"
	"regexp"
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
)

type LogParser struct {
	api          *ApiClient
	loc          *time.Location
	location     string
	localName    string
	localUserID  string
	pendingNames []string          // FIFO queue
	internalIDs  map[string]int    // name -> internal player ID
	pendingJoins map[string]string // name -> userID
}

func NewLogParser(api *ApiClient, loc *time.Location) *LogParser {
	return &LogParser{
		api:          api,
		loc:          loc,
		internalIDs:  make(map[string]int),
		pendingJoins: make(map[string]string),
	}
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

func (p *LogParser) sendJoin(name, userID string, internalID int, ts time.Time) {
	log.Printf("JOIN  [%s] %s (%s) internal_id=%d", ts.In(p.loc).Format("2006-01-02 15:04:05"), name, userID, internalID)
	p.api.SendEvent("join", p.location, name, userID, &internalID, ts)
}

func (p *LogParser) sendLeave(name, userID string, internalID int, ts time.Time) {
	log.Printf("LEAVE [%s] %s (%s) internal_id=%d", ts.In(p.loc).Format("2006-01-02 15:04:05"), name, userID, internalID)
	p.api.SendEvent("leave", p.location, name, userID, &internalID, ts)
}

func (p *LogParser) closeCurrentLocation(ts time.Time) {
	if p.location != "" {
		log.Printf("Closing location %s at %s", p.location, ts)
		p.api.CloseLocation(p.location, p.localUserID, ts)
	}
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
	if p.location == "" {
		return
	}
	if m := rePlayerJoin.FindStringSubmatch(line); m != nil {
		p.handlePlayerJoin(m[1], m[2])
		return
	}
	if m := rePlayerAPI.FindStringSubmatch(line); m != nil {
		p.handlePlayerAPI(m[1], m[2])
		return
	}
	if m := reRestored.FindStringSubmatch(line); m != nil {
		p.handleRestored(m[1], ts)
		return
	}
	if m := rePlayerLeft.FindStringSubmatch(line); m != nil {
		p.handlePlayerLeft(m[1], m[2], ts)
		return
	}
	if p.localName != "" {
		m := reDestroying.FindStringSubmatch(line)
		if m != nil && m[1] == p.localName {
			p.handleDestroying(ts)
		}
	}
}

func (p *LogParser) handleJoining(worldID string, ts time.Time) {
	p.closeCurrentLocation(ts) // Destroying が発火しないことがあるためここでもcloseを呼ぶ
	p.location = worldID
	p.pendingJoins = make(map[string]string)
	p.pendingNames = nil
	p.internalIDs = make(map[string]int)
	p.localName = ""
	p.localUserID = ""
	log.Printf("Location: %s", p.location)
}

func (p *LogParser) handlePlayerJoin(name, userID string) {
	p.pendingJoins[name] = userID
}

func (p *LogParser) handlePlayerAPI(name, kind string) {
	p.pendingNames = append(p.pendingNames, name)
	if kind == "local" {
		p.localName = name
	}
}

func (p *LogParser) handleRestored(rawID string, ts time.Time) {
	if len(p.pendingNames) == 0 {
		return
	}
	name := p.pendingNames[0]
	p.pendingNames = p.pendingNames[1:]
	internalID, _ := strconv.Atoi(rawID)
	p.internalIDs[name] = internalID

	userID, ok := p.pendingJoins[name]
	if !ok {
		return
	}
	delete(p.pendingJoins, name)

	if name == p.localName {
		p.localUserID = userID
	}
	p.sendJoin(name, userID, internalID, ts)
}

func (p *LogParser) handlePlayerLeft(name, userID string, ts time.Time) {
	delete(p.pendingJoins, name)
	// pendingNames から削除（正常にJoinされなかったユーザーの対応）
	for i, n := range p.pendingNames {
		if n == name {
			p.pendingNames = append(p.pendingNames[:i], p.pendingNames[i+1:]...)
			break
		}
	}
	// Restored player が出ていないユーザーはLEAVE判定をスキップ
	internalID, hasID := p.internalIDs[name]
	if !hasID {
		return
	}
	p.sendLeave(name, userID, internalID, ts)
	delete(p.internalIDs, name)
}

func (p *LogParser) handleDestroying(ts time.Time) {
	if id, ok := p.internalIDs[p.localName]; ok {
		p.sendLeave(p.localName, p.localUserID, id, ts)
	}
	p.closeCurrentLocation(ts)
}
