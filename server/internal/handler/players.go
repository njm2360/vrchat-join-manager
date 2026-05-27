package handler

import (
	"context"

	"github.com/njm2360/vrchat-join-manager/server/internal/gen"
)

func (s *Server) GetPlayer(ctx context.Context, request gen.GetPlayerRequestObject) (gen.GetPlayerResponseObject, error) {
	r, err := s.Players.GetDetail(ctx, request.UserId)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return gen.GetPlayer404Response{}, nil
	}
	return gen.GetPlayer200JSONResponse{
		UserId:               r.UserID,
		DisplayName:          r.DisplayName,
		DiscordId:            strPtr(r.DiscordID),
		CreatedAt:            parseTime(r.CreatedAt),
		UpdatedAt:            parseTime(r.UpdatedAt),
		TotalVisits:          r.TotalVisits,
		TotalDurationSeconds: r.TotalDurationSeconds,
		FirstSeen:            parseTimeFromNullable(r.FirstSeen),
		LastSeen:             parseTimeFromNullable(r.LastSeen),
		InRoom:               r.InRoom,
	}, nil
}

func (s *Server) ListPlayers(ctx context.Context, request gen.ListPlayersRequestObject) (gen.ListPlayersResponseObject, error) {
	order := enumStrOr(request.Params.Order, "asc")
	offset := derefInt(request.Params.Offset)
	rows, err := s.Players.List(ctx, request.Params.Name, order, request.Params.Limit, offset)
	if err != nil {
		return nil, err
	}
	out := make(gen.ListPlayers200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, gen.PlayerOut{
			UserId:      r.UserID,
			DisplayName: r.DisplayName,
			CreatedAt:   parseTime(r.CreatedAt),
			UpdatedAt:   parseTime(r.UpdatedAt),
		})
	}
	return out, nil
}

func (s *Server) GetPlayerEvents(ctx context.Context, request gen.GetPlayerEventsRequestObject) (gen.GetPlayerEventsResponseObject, error) {
	start := timePtrToStrPtr(request.Params.Start)
	end := timePtrToStrPtr(request.Params.End)
	order := enumStrOr(request.Params.Order, "asc")
	offset := derefInt(request.Params.Offset)
	rows, err := s.Players.Events(ctx, request.UserId, request.Params.InstanceId, start, end, order, request.Params.Limit, offset)
	if err != nil {
		return nil, err
	}
	out := make(gen.GetPlayerEvents200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, gen.EventOut{
			Id:          r.ID,
			EventType:   gen.EventOutEventType(r.EventType),
			InstanceId:  r.InstanceID,
			WorldId:     r.WorldID,
			UserId:      r.UserID,
			DisplayName: r.DisplayName,
			Timestamp:   parseTime(r.Timestamp),
		})
	}
	return out, nil
}

func (s *Server) GetPlayerSessions(ctx context.Context, request gen.GetPlayerSessionsRequestObject) (gen.GetPlayerSessionsResponseObject, error) {
	start := timePtrToStrPtr(request.Params.Start)
	end := timePtrToStrPtr(request.Params.End)
	order := enumStrOr(request.Params.Order, "asc")
	offset := derefInt(request.Params.Offset)
	rows, err := s.Players.Sessions(ctx,
		request.UserId,
		request.Params.InstanceId,
		request.Params.WorldId,
		request.Params.GroupId,
		start, end,
		order,
		request.Params.Limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	out := make(gen.GetPlayerSessions200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, gen.PlayerSessionOut{
			Id:               r.ID,
			InstanceId:       r.InstanceID,
			WorldId:          r.WorldID,
			JoinTs:           parseTime(r.JoinTs),
			LeaveTs:          parseTimeFromNullable(r.LeaveTs),
			DurationSeconds:  intPtr(r.DurationSeconds),
			IsEstimatedLeave: r.IsEstimatedLeave,
		})
	}
	return out, nil
}
