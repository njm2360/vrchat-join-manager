package handler

import (
	"context"
	"time"

	"github.com/njm2360/vrchat-join-manager/server/internal/gen"
	"github.com/njm2360/vrchat-join-manager/server/internal/timeutil"
)

func (s *Server) ListGroups(ctx context.Context, request gen.ListGroupsRequestObject) (gen.ListGroupsResponseObject, error) {
	order := enumStrOr(request.Params.Order, "desc")
	offset := derefInt(request.Params.Offset)
	rows, err := s.Groups.List(ctx, order, request.Params.Limit, offset)
	if err != nil {
		return nil, err
	}
	out := make(gen.ListGroups200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, gen.GroupOut{
			GroupId:   r.GroupID,
			Name:      strPtr(r.Name),
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
		})
	}
	return out, nil
}

func (s *Server) RenameGroup(ctx context.Context, request gen.RenameGroupRequestObject) (gen.RenameGroupResponseObject, error) {
	ts := timeutil.FormatUTC(time.Now())
	ok, err := s.Groups.Rename(ctx, request.GroupId, request.Body.Name, ts)
	if err != nil {
		return nil, err
	}
	if !ok {
		return gen.RenameGroup404Response{}, nil
	}
	return gen.RenameGroup204Response{}, nil
}

func (s *Server) DeleteGroup(ctx context.Context, request gen.DeleteGroupRequestObject) (gen.DeleteGroupResponseObject, error) {
	ok, err := s.Groups.Delete(ctx, request.GroupId)
	if err != nil {
		return nil, err
	}
	if !ok {
		return gen.DeleteGroup404Response{}, nil
	}
	return gen.DeleteGroup204Response{}, nil
}
