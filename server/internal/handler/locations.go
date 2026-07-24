package handler

import (
	"context"

	"github.com/njm2360/vrchat-join-manager/server/internal/domain"
	"github.com/njm2360/vrchat-join-manager/server/internal/gen"
)

func (s *Server) CloseLocation(ctx context.Context, request gen.CloseLocationRequestObject) (gen.CloseLocationResponseObject, error) {
	ts := timeToStr(request.Body.At)

	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	id, err := s.Instances.GetOpenInstanceID(ctx, tx, request.LocationId)
	if err != nil {
		return nil, err
	}
	if id == nil {
		return gen.CloseLocation404Response{}, nil
	}

	// リクエスト送信者(自分)は実測、他プレイヤーは推定として閉じる
	if err := s.Instances.CloseInstance(ctx, tx, *id, ts, request.Body.UserId); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return gen.CloseLocation204Response{}, nil
}

func (s *Server) CheckinLocation(ctx context.Context, request gen.CheckinLocationRequestObject) (gen.CheckinLocationResponseObject, error) {
	body := request.Body
	self := domain.ObservedPlayer{UserID: body.Self.UserId, InternalID: body.Self.InternalId}
	players := make([]domain.ObservedPlayer, 0, len(body.Players))
	for _, p := range body.Players {
		players = append(players, domain.ObservedPlayer{UserID: p.UserId, InternalID: p.InternalId})
	}

	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	notResumed := func() (gen.CheckinLocationResponseObject, error) {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return gen.CheckinLocation200JSONResponse{
			Resumed:        false,
			ResumedUserIds: []string{},
		}, nil
	}

	// openなインスタンスが残っている場合は復元しない
	openID, err := s.Instances.GetOpenInstanceID(ctx, tx, request.LocationId)
	if err != nil {
		return nil, err
	}
	if openID != nil {
		return notResumed()
	}

	// 復元候補となるインスタンスの選定
	cand, err := s.Instances.LastClosedInstance(ctx, tx, request.LocationId)
	if err != nil {
		return nil, err
	}
	if cand == nil {
		return notResumed()
	}

	maxInternalID, err := s.Instances.MaxInternalID(ctx, tx, cand.ID)
	if err != nil {
		return nil, err
	}

	// 証人候補(自分以外)のセッション記録を引く
	userIDs := make([]string, 0, len(players))
	seen := map[string]struct{}{self.UserID: {}}
	for _, p := range players {
		if _, dup := seen[p.UserID]; dup {
			continue
		}
		seen[p.UserID] = struct{}{}
		userIDs = append(userIDs, p.UserID)
	}
	sessions, err := s.Instances.SessionsForUsers(ctx, tx, cand.ID, userIDs)
	if err != nil {
		return nil, err
	}

	d := domain.DecideCheckinResume(body.At, parseTime(cand.OpenedAt), maxInternalID, self, players, sessions)
	if !d.Resume {
		return notResumed()
	}

	// 同一インスタンス確定: インスタンスと証人のセッションを復元する
	sessionIDs := make([]int, 0, len(d.Witnesses))
	resumedUserIDs := make([]string, 0, len(d.Witnesses))
	for _, w := range d.Witnesses {
		sessionIDs = append(sessionIDs, w.ID)
		resumedUserIDs = append(resumedUserIDs, w.UserID)
	}
	if err := s.Instances.Reopen(ctx, tx, cand.ID); err != nil {
		return nil, err
	}
	if err := s.Events.ResumeSessions(ctx, tx, sessionIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return gen.CheckinLocation200JSONResponse{
		Resumed:        true,
		ResumedUserIds: resumedUserIDs,
	}, nil
}
