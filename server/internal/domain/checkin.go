package domain

import "time"

const CheckinResumeWindow = 7 * 24 * time.Hour

type ObservedPlayer struct {
	UserID     string
	InternalID int
}

type SessionRecord struct {
	ID               int
	UserID           string
	InternalID       int
	JoinTs           string
	IsEstimatedLeave bool
}

type CheckinDecision struct {
	Resume    bool
	Witnesses []SessionRecord
}

func DecideCheckinResume(
	at, openedAt time.Time,
	maxInternalID int,
	self ObservedPlayer,
	players []ObservedPlayer,
	sessions []SessionRecord,
) CheckinDecision {
	none := CheckinDecision{}

	// 7日以上前に開かれたインスタンスは除外する
	age := at.Sub(openedAt)
	if age < 0 || age >= CheckinResumeWindow {
		return none
	}

	// Note: internal_idはインスタンス生存中は巻き戻らない

	// 自分のIDが記録上の最大を超えていなければ別インスタンス
	// ※インスタンスリセットならほぼここで弾ける
	if self.InternalID <= maxInternalID {
		return none
	}

	var witnesses []SessionRecord
	mismatches := 0
	seen := map[string]struct{}{self.UserID: {}}
	for _, p := range players {
		if _, dup := seen[p.UserID]; dup {
			continue
		}
		seen[p.UserID] = struct{}{}

		// 不在中に採番されたIDは記録に存在し得ないため照合しない
		if p.InternalID > maxInternalID {
			continue
		}

		// InternalIDとUserIDがマッチする推定Leaveなセッションを探す
		if s := latestEstimatedLeave(sessions, p); s != nil {
			witnesses = append(witnesses, *s)
			continue
		}

		// 既知ユーザーの低ID不一致は再採番かID取り違えの証拠
		// ※未知ユーザーは記録漏れがありうるため許容
		if isKnown(sessions, p.UserID) {
			mismatches++
		}
	}

	// 一致が不一致を上回る場合のみ復元する(同数は安全側で新規扱い)
	if mismatches >= len(witnesses) {
		return none
	}

	return CheckinDecision{Resume: true, Witnesses: witnesses}
}

func latestEstimatedLeave(sessions []SessionRecord, p ObservedPlayer) *SessionRecord {
	var latest *SessionRecord
	for i := range sessions {
		s := &sessions[i]
		if s.UserID != p.UserID || s.InternalID != p.InternalID || !s.IsEstimatedLeave {
			continue
		}
		if latest == nil || s.JoinTs > latest.JoinTs {
			latest = s
		}
	}
	return latest
}

func isKnown(sessions []SessionRecord, userID string) bool {
	for _, s := range sessions {
		if s.UserID == userID {
			return true
		}
	}
	return false
}
