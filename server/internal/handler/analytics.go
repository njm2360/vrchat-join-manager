package handler

import (
	"context"

	"github.com/njm2360/vrchat-join-manager/server/internal/gen"
	"github.com/njm2360/vrchat-join-manager/server/internal/repository"
)

func (s *Server) GetDailyActiveUsers(ctx context.Context, request gen.GetDailyActiveUsersRequestObject) (gen.GetDailyActiveUsersResponseObject, error) {
	start := timePtrToStrPtr(request.Params.Start)
	end := timePtrToStrPtr(request.Params.End)
	rows, err := s.Analytics.DailyActiveUsers(ctx, request.Params.WorldId, request.Params.GroupId, start, end)
	if err != nil {
		return nil, err
	}
	out := make(gen.GetDailyActiveUsers200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, gen.DailyActiveUsersPoint{
			Day:         r.Day,
			ActiveUsers: r.ActiveUsers,
		})
	}
	return out, nil
}

func (s *Server) GetHourlyActiveUsers(ctx context.Context, request gen.GetHourlyActiveUsersRequestObject) (gen.GetHourlyActiveUsersResponseObject, error) {
	start := timePtrToStrPtr(request.Params.Start)
	end := timePtrToStrPtr(request.Params.End)
	rows, err := s.Analytics.HourlyActiveUsers(ctx, request.Params.WorldId, request.Params.GroupId, start, end)
	if err != nil {
		return nil, err
	}
	out := make(gen.GetHourlyActiveUsers200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, gen.HourlyActiveUsersPoint{
			Hour:        r.Hour,
			ActiveUsers: r.ActiveUsers,
		})
	}
	return out, nil
}

func (s *Server) GetPlayerRankings(ctx context.Context, request gen.GetPlayerRankingsRequestObject) (gen.GetPlayerRankingsResponseObject, error) {
	order := enumStrOr(request.Params.Order, "desc")
	offset := derefInt(request.Params.Offset)
	rows, err := s.Analytics.PlayerRankings(ctx, request.Params.WorldId, request.Params.GroupId, order, request.Params.Limit, offset)
	if err != nil {
		return nil, err
	}
	out := make(gen.GetPlayerRankings200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, gen.PlayerRankOut{
			Rank:                 r.Rank,
			UserId:               r.UserID,
			DisplayName:          r.DisplayName,
			TotalDurationSeconds: r.TotalDurationSeconds,
			SessionCount:         r.SessionCount,
		})
	}
	return out, nil
}

func (s *Server) GetJoinViolationRankings(ctx context.Context, request gen.GetJoinViolationRankingsRequestObject) (gen.GetJoinViolationRankingsResponseObject, error) {
	start := timePtrToStrPtr(request.Params.Start)
	end := timePtrToStrPtr(request.Params.End)
	order := enumStrOr(request.Params.Order, "desc")
	offset := derefInt(request.Params.Offset)
	allowDiff := 0
	if request.Params.AllowDiff != nil {
		allowDiff = *request.Params.AllowDiff
	}
	rejoin := 180
	if request.Params.RejoinSeconds != nil {
		rejoin = *request.Params.RejoinSeconds
	}
	grace := 900
	if request.Params.GraceSeconds != nil {
		grace = *request.Params.GraceSeconds
	}
	rows, err := s.Analytics.JoinViolationRankings(ctx, repository.JoinViolationRankingsParams{
		GroupID:       request.Params.GroupId,
		Start:         start,
		End:           end,
		Order:         order,
		Limit:         request.Params.Limit,
		Offset:        offset,
		AllowDiff:     allowDiff,
		MinDuration:   request.Params.MinDuration,
		RejoinSeconds: rejoin,
		GraceSeconds:  grace,
	})
	if err != nil {
		return nil, err
	}
	out := make(gen.GetJoinViolationRankings200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, gen.JoinViolationRankOut{
			Rank:           r.Rank,
			UserId:         r.UserID,
			DisplayName:    r.DisplayName,
			ViolationCount: r.ViolationCount,
			TotalJoins:     r.TotalJoins,
		})
	}
	return out, nil
}
