package handler

import (
	"context"

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

	if _, err := s.Instances.CloseLocationSessions(ctx, tx, *id, ts, request.Body.UserId); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return gen.CloseLocation204Response{}, nil
}

func (s *Server) GetPotentialSessions(ctx context.Context, request gen.GetPotentialSessionsRequestObject) (gen.GetPotentialSessionsResponseObject, error) {
	rows, err := s.Instances.GetPotentialSessions(ctx, request.LocationId)
	if err != nil {
		return nil, err
	}
	out := make(gen.GetPotentialSessions200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, gen.PotentialSessionOut{
			UserId:     r.UserID,
			InternalId: r.InternalID,
		})
	}
	return out, nil
}

func (s *Server) ResumeInstance(ctx context.Context, request gen.ResumeInstanceRequestObject) (gen.ResumeInstanceResponseObject, error) {
	if err := s.Instances.Resume(ctx, request.LocationId, request.Body.UserIds); err != nil {
		return nil, err
	}
	return gen.ResumeInstance204Response{}, nil
}
