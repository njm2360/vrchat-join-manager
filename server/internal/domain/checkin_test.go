package domain

import (
	"reflect"
	"testing"
	"time"
)

func TestDecideCheckinResume(t *testing.T) {
	at := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	opened := at.Add(-24 * time.Hour)
	self := ObservedPlayer{UserID: "self", InternalID: 10}

	tests := []struct {
		name     string
		at       time.Time
		openedAt time.Time
		maxID    int
		self     ObservedPlayer
		players  []ObservedPlayer
		sessions []SessionRecord
		want     CheckinDecision
	}{
		{
			name: "witness match resumes",
			at:   at, openedAt: opened, maxID: 5, self: self,
			players: []ObservedPlayer{{UserID: "a", InternalID: 3}},
			sessions: []SessionRecord{
				{ID: 1, UserID: "a", InternalID: 3, IsEstimatedLeave: true},
			},
			want: CheckinDecision{Resume: true, Witnesses: []SessionRecord{
				{ID: 1, UserID: "a", InternalID: 3, IsEstimatedLeave: true},
			}},
		},
		{
			name: "opened at window boundary is too old",
			at:   at, openedAt: at.Add(-CheckinResumeWindow), maxID: 5, self: self,
			players: []ObservedPlayer{{UserID: "a", InternalID: 3}},
			sessions: []SessionRecord{
				{ID: 1, UserID: "a", InternalID: 3, IsEstimatedLeave: true},
			},
			want: CheckinDecision{},
		},
		{
			name: "opened in the future is rejected",
			at:   at, openedAt: at.Add(time.Minute), maxID: 5, self: self,
			players: []ObservedPlayer{{UserID: "a", InternalID: 3}},
			sessions: []SessionRecord{
				{ID: 1, UserID: "a", InternalID: 3, IsEstimatedLeave: true},
			},
			want: CheckinDecision{},
		},
		{
			name: "self internal id not above recorded max means new instance",
			at:   at, openedAt: opened, maxID: 10, self: self,
			players: []ObservedPlayer{{UserID: "a", InternalID: 3}},
			sessions: []SessionRecord{
				{ID: 1, UserID: "a", InternalID: 3, IsEstimatedLeave: true},
			},
			want: CheckinDecision{},
		},
		{
			name: "no witnesses and no mismatches stays new",
			at:   at, openedAt: opened, maxID: 5, self: self,
			players:  []ObservedPlayer{{UserID: "stranger", InternalID: 3}},
			sessions: nil,
			want:     CheckinDecision{},
		},
		{
			name: "tie between witnesses and mismatches stays new",
			at:   at, openedAt: opened, maxID: 5, self: self,
			players: []ObservedPlayer{
				{UserID: "a", InternalID: 3},
				{UserID: "b", InternalID: 4}, // InternalID取り違え
			},
			sessions: []SessionRecord{
				{ID: 1, UserID: "a", InternalID: 3, IsEstimatedLeave: true},
				{ID: 2, UserID: "b", InternalID: 2, IsEstimatedLeave: true}, // 別IDで既知 → 不一致
			},
			want: CheckinDecision{},
		},
		{
			name: "majority of witnesses resumes",
			at:   at, openedAt: opened, maxID: 5, self: self,
			players: []ObservedPlayer{
				{UserID: "a", InternalID: 3},
				{UserID: "b", InternalID: 4},
				{UserID: "c", InternalID: 5},
			},
			sessions: []SessionRecord{
				{ID: 1, UserID: "a", InternalID: 3, IsEstimatedLeave: true},
				{ID: 2, UserID: "b", InternalID: 4, IsEstimatedLeave: true},
				{ID: 3, UserID: "c", InternalID: 1, IsEstimatedLeave: true}, // 不一致
			},
			want: CheckinDecision{Resume: true, Witnesses: []SessionRecord{
				{ID: 1, UserID: "a", InternalID: 3, IsEstimatedLeave: true},
				{ID: 2, UserID: "b", InternalID: 4, IsEstimatedLeave: true},
			}},
		},
		{
			name: "unknown users are not counted as mismatches",
			at:   at, openedAt: opened, maxID: 5, self: self,
			players: []ObservedPlayer{
				{UserID: "a", InternalID: 3},
				{UserID: "stranger", InternalID: 2},
			},
			sessions: []SessionRecord{
				{ID: 1, UserID: "a", InternalID: 3, IsEstimatedLeave: true},
			},
			want: CheckinDecision{Resume: true, Witnesses: []SessionRecord{
				{ID: 1, UserID: "a", InternalID: 3, IsEstimatedLeave: true},
			}},
		},
		{
			name: "ids assigned while absent are not matched",
			at:   at, openedAt: opened, maxID: 5, self: self,
			players: []ObservedPlayer{
				{UserID: "a", InternalID: 3},
				{UserID: "b", InternalID: 8}, // Agent不在中にRejoinした人
			},
			sessions: []SessionRecord{
				{ID: 1, UserID: "a", InternalID: 3, IsEstimatedLeave: true},
				{ID: 2, UserID: "b", InternalID: 2, IsEstimatedLeave: true},
			},
			want: CheckinDecision{Resume: true, Witnesses: []SessionRecord{
				{ID: 1, UserID: "a", InternalID: 3, IsEstimatedLeave: true},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecideCheckinResume(tt.at, tt.openedAt, tt.maxID, tt.self, tt.players, tt.sessions)
			if got.Resume != tt.want.Resume {
				t.Fatalf("Resume = %v, want %v", got.Resume, tt.want.Resume)
			}
			if tt.want.Resume && !reflect.DeepEqual(got.Witnesses, tt.want.Witnesses) {
				t.Errorf("Witnesses = %v, want %v", got.Witnesses, tt.want.Witnesses)
			}
		})
	}
}
