package core

import (
	"log"
	"regexp"
	"strconv"
	"strings"
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

type VRChatLogParser struct {
	api          *ApiClient
	loc          *time.Location
	location     string
	localName    string
	pendingNames []string          // FIFO queue
	internalIDs  map[string]int    // name -> internal player ID
	pendingJoins map[string]string // name -> userID
}

func NewVRChatLogParser(api *ApiClient, loc *time.Location) *VRChatLogParser {
	return &VRChatLogParser{
		api:          api,
		loc:          loc,
		internalIDs:  make(map[string]int),
		pendingJoins: make(map[string]string),
	}
}

func (p *VRChatLogParser) parseTimestamp(line string) (time.Time, bool) {
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

func (p *VRChatLogParser) closeCurrentLocation(ts time.Time) {
	if p.location != "" {
		log.Printf("Closing location %s at %s", p.location, ts)
		p.api.CloseLocation(p.location, ts)
	}
}

func (p *VRChatLogParser) OnLine(_ string, line string) {
	ts, ok := p.parseTimestamp(line)
	if !ok {
		return
	}

	// ロケーション移動検知
	if m := reJoining.FindStringSubmatch(line); m != nil {
		// Destroying が発火しないことがあるためここでもcloseを呼ぶ
		p.closeCurrentLocation(ts)
		p.pendingJoins = make(map[string]string)
		p.pendingNames = nil
		p.internalIDs = make(map[string]int)
		p.localName = ""
		p.location = m[1]
		log.Printf("Location: %s", p.location)
		return
	}

	// ロケーションが取得できていない場合は処理しない
	if p.location == "" {
		return
	}

	// OnPlayerJoined
	if m := rePlayerJoin.FindStringSubmatch(line); m != nil {
		p.pendingJoins[m[1]] = m[2]
		return
	}

	// Initialized PlayerAPI "XXXX" is (remote | local)
	if strings.Contains(line, "Initialized PlayerAPI") {
		if m := rePlayerAPI.FindStringSubmatch(line); m != nil {
			name, kind := m[1], m[2]
			p.pendingNames = append(p.pendingNames, name)
			if kind == "local" {
				p.localName = name
			}
		}
		return
	}

	// Restored player N
	if m := reRestored.FindStringSubmatch(line); m != nil {
		if len(p.pendingNames) > 0 {
			name := p.pendingNames[0]
			p.pendingNames = p.pendingNames[1:]
			internalID, _ := strconv.Atoi(m[1])
			p.internalIDs[name] = internalID
			if userID, ok := p.pendingJoins[name]; ok {
				delete(p.pendingJoins, name)
				log.Printf("JOIN  [%s] %s (%s) internal_id=%d", ts.In(p.loc).Format("2006-01-02 15:04:05"), name, userID, internalID)
				id := internalID
				p.api.SendEvent("join", p.location, name, userID, &id, ts)
			}
		}
		return
	}

	// OnPlayerLeft
	if m := rePlayerLeft.FindStringSubmatch(line); m != nil {
		name, userID := m[1], m[2]
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
		log.Printf("LEAVE [%s] %s (%s) internal_id=%d", ts.In(p.loc).Format("2006-01-02 15:04:05"), name, userID, internalID)
		id := internalID
		p.api.SendEvent("leave", p.location, name, userID, &id, ts)
		delete(p.internalIDs, name)
		return
	}

	// インスタンス移動 or アプリ終了時: Destroying <local_name>
	if p.localName != "" && strings.Contains(line, "Destroying") {
		if m := reDestroying.FindStringSubmatch(line); m != nil && m[1] == p.localName {
			p.closeCurrentLocation(ts)
		}
	}
}
