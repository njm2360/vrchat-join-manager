package handler

import (
	"context"
	"time"

	"github.com/njm2360/vrchat-join-manager/server/internal/gen"
	"github.com/njm2360/vrchat-join-manager/server/internal/timeutil"
)

func (s *Server) ListWorlds(ctx context.Context, request gen.ListWorldsRequestObject) (gen.ListWorldsResponseObject, error) {
	start := timePtrToStrPtr(request.Params.Start)
	end := timePtrToStrPtr(request.Params.End)
	order := enumStrOr(request.Params.Order, "desc")
	offset := derefInt(request.Params.Offset)
	rows, err := s.Worlds.List(ctx, start, end, order, request.Params.Limit, offset)
	if err != nil {
		return nil, err
	}
	out := make(gen.ListWorlds200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, gen.WorldOut{
			WorldId:      r.WorldID,
			Name:         strPtr(r.Name),
			CreatedAt:    r.CreatedAt,
			UpdatedAt:    r.UpdatedAt,
			LastSeen:     strPtr(r.LastSeen),
			SessionCount: r.SessionCount,
		})
	}
	return out, nil
}

func (s *Server) RenameWorld(ctx context.Context, request gen.RenameWorldRequestObject) (gen.RenameWorldResponseObject, error) {
	ts := timeutil.FormatUTC(time.Now())
	ok, err := s.Worlds.Rename(ctx, request.WorldId, request.Body.Name, ts)
	if err != nil {
		return nil, err
	}
	if !ok {
		return gen.RenameWorld404Response{}, nil
	}
	return gen.RenameWorld204Response{}, nil
}

func (s *Server) DeleteWorld(ctx context.Context, request gen.DeleteWorldRequestObject) (gen.DeleteWorldResponseObject, error) {
	ok, err := s.Worlds.Delete(ctx, request.WorldId)
	if err != nil {
		return nil, err
	}
	if !ok {
		return gen.DeleteWorld404Response{}, nil
	}
	return gen.DeleteWorld204Response{}, nil
}
