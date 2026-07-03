package handler

import (
	"context"

	"github.com/njm2360/vrchat-join-manager/server/internal/gen"
	"github.com/njm2360/vrchat-join-manager/server/internal/repository"
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

	if _, err := s.Instances.CloseLocationSessions(ctx, tx, *id, ts, request.Body.UserId); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return gen.CloseLocation204Response{}, nil
}

func (s *Server) CheckinLocation(ctx context.Context, request gen.CheckinLocationRequestObject) (gen.CheckinLocationResponseObject, error) {
	body := request.Body
	self := repository.ObservedPlayer{UserID: body.Self.UserId, InternalID: body.Self.InternalId}
	players := make([]repository.ObservedPlayer, 0, len(body.Players))
	for _, p := range body.Players {
		players = append(players, repository.ObservedPlayer{UserID: p.UserId, InternalID: p.InternalId})
	}
	resumed, err := s.Instances.Checkin(ctx, request.LocationId, timeToStr(body.At), self, players)
	if err != nil {
		return nil, err
	}
	return gen.CheckinLocation200JSONResponse{
		Resumed:        len(resumed) > 0,
		ResumedUserIds: resumed,
	}, nil
}
