package handler

import (
	"context"
	"strings"

	"github.com/njm2360/vrchat-join-manager/server/internal/gen"
	"github.com/njm2360/vrchat-join-manager/server/internal/repository"
)

func instanceRowToOut(r repository.InstanceRow) gen.InstanceOut {
	return gen.InstanceOut{
		Id:              r.ID,
		LocationId:      r.LocationID,
		WorldId:         r.WorldID,
		WorldName:       strPtr(r.WorldName),
		InstanceId:      strPtr(r.InstanceID),
		GroupId:         strPtr(r.GroupID),
		GroupName:       strPtr(r.GroupName),
		GroupAccessType: strPtr(r.GroupAccessType),
		Region:          strPtr(r.Region),
		Friends:         strPtr(r.Friends),
		Hidden:          strPtr(r.Hidden),
		Private:         strPtr(r.Private),
		OpenedAt:        parseTime(r.OpenedAt),
		ClosedAt:        parseTimeFromNullable(r.ClosedAt),
		UserCount:       r.UserCount,
	}
}

func sessionRowToOut(r repository.SessionRow) gen.SessionOut {
	return gen.SessionOut{
		Id:               r.ID,
		InstanceId:       r.InstanceID,
		InternalId:       r.InternalID,
		UserId:           r.UserID,
		DisplayName:      r.DisplayName,
		DiscordId:        strPtr(r.DiscordID),
		JoinTs:           parseTime(r.JoinTs),
		LeaveTs:          parseTimeFromNullable(r.LeaveTs),
		DurationSeconds:  intPtr(r.DurationSeconds),
		IsEstimatedJoin:  r.IsEstimatedJoin,
		IsEstimatedLeave: r.IsEstimatedLeave,
	}
}

func (s *Server) ListInstances(ctx context.Context, request gen.ListInstancesRequestObject) (gen.ListInstancesResponseObject, error) {
	start := timePtrToStrPtr(request.Params.Start)
	end := timePtrToStrPtr(request.Params.End)
	order := enumStrOr(request.Params.Order, "desc")
	sortBy := enumStrOr(request.Params.SortBy, "opened_at")
	offset := derefInt(request.Params.Offset)
	rows, err := s.Locations.ListInstances(ctx,
		start, end,
		request.Params.IsOpen,
		request.Params.WorldId,
		request.Params.GroupId,
		request.Params.Region,
		sortBy, order,
		request.Params.Limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	out := make(gen.ListInstances200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, instanceRowToOut(r))
	}
	return out, nil
}

func (s *Server) GetInstance(ctx context.Context, request gen.GetInstanceRequestObject) (gen.GetInstanceResponseObject, error) {
	r, err := s.Locations.GetInstance(ctx, request.InstanceId)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return gen.GetInstance404Response{}, nil
	}
	return gen.GetInstance200JSONResponse(instanceRowToOut(*r)), nil
}

func (s *Server) DeleteInstance(ctx context.Context, request gen.DeleteInstanceRequestObject) (gen.DeleteInstanceResponseObject, error) {
	ok, err := s.Instances.Delete(ctx, request.InstanceId)
	if err != nil {
		return nil, err
	}
	if !ok {
		return gen.DeleteInstance404Response{}, nil
	}
	return gen.DeleteInstance204Response{}, nil
}

func (s *Server) GetInstancePresence(ctx context.Context, request gen.GetInstancePresenceRequestObject) (gen.GetInstancePresenceResponseObject, error) {
	at := timeToStr(request.Params.At)
	rows, err := s.Locations.GetPresence(ctx, request.InstanceId, at)
	if err != nil {
		return nil, err
	}
	out := make(gen.GetInstancePresence200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, sessionRowToOut(r))
	}
	return out, nil
}

func (s *Server) GetInstancePlayers(ctx context.Context, request gen.GetInstancePlayersRequestObject) (gen.GetInstancePlayersResponseObject, error) {
	rows, err := s.Locations.GetLocationPlayers(ctx,
		request.InstanceId,
		enumStrOr(request.Params.SortBy, "internal_id"),
		enumStrOr(request.Params.Order, "asc"),
	)
	if err != nil {
		return nil, err
	}
	out := make(gen.GetInstancePlayers200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, gen.LocationPlayerOut{
			UserId:      r.UserID,
			DisplayName: r.DisplayName,
			DiscordId:   strPtr(r.DiscordID),
			InternalId:  r.InternalID,
			JoinTs:      parseTime(r.JoinTs),
			JoinCount:   r.JoinCount,
		})
	}
	return out, nil
}

func (s *Server) GetInstanceVisitors(ctx context.Context, request gen.GetInstanceVisitorsRequestObject) (gen.GetInstanceVisitorsResponseObject, error) {
	offset := derefInt(request.Params.Offset)
	rows, err := s.Locations.GetLocationVisitors(ctx,
		request.InstanceId,
		enumStrOr(request.Params.SortBy, "last_seen"),
		enumStrOr(request.Params.Order, "desc"),
		request.Params.Limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	out := make(gen.GetInstanceVisitors200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, gen.VisitorOut{
			UserId:               r.UserID,
			DisplayName:          r.DisplayName,
			FirstSeen:            parseTime(r.FirstSeen),
			LastSeen:             parseTime(r.LastSeen),
			JoinCount:            r.JoinCount,
			TotalDurationSeconds: r.TotalDurationSeconds,
		})
	}
	return out, nil
}

func (s *Server) GetInstancePresenceTimeline(ctx context.Context, request gen.GetInstancePresenceTimelineRequestObject) (gen.GetInstancePresenceTimelineResponseObject, error) {
	start := timePtrToStrPtr(request.Params.Start)
	end := timePtrToStrPtr(request.Params.End)
	rows, err := s.Locations.GetPresenceTimeline(ctx, request.InstanceId, start, end)
	if err != nil {
		return nil, err
	}
	out := make(gen.GetInstancePresenceTimeline200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, gen.TimelinePoint{
			Timestamp:   parseTime(r.Timestamp),
			Count:       r.Count,
			UserId:      strPtr(r.UserID),
			DisplayName: strPtr(r.DisplayName),
		})
	}
	return out, nil
}

func (s *Server) GetInstanceEvents(ctx context.Context, request gen.GetInstanceEventsRequestObject) (gen.GetInstanceEventsResponseObject, error) {
	start := timePtrToStrPtr(request.Params.Start)
	end := timePtrToStrPtr(request.Params.End)
	offset := derefInt(request.Params.Offset)
	rows, err := s.Locations.GetLocationEvents(ctx, request.InstanceId, start, end, enumStrOr(request.Params.Order, "desc"), request.Params.Limit, offset)
	if err != nil {
		return nil, err
	}
	out := make(gen.GetInstanceEvents200JSONResponse, 0, len(rows))
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

func (s *Server) GetInstanceStats(ctx context.Context, request gen.GetInstanceStatsRequestObject) (gen.GetInstanceStatsResponseObject, error) {
	r, err := s.Locations.GetInstanceStats(ctx, request.InstanceId)
	if err != nil {
		return nil, err
	}
	return gen.GetInstanceStats200JSONResponse{
		EventCount:           r.EventCount,
		SessionCount:         r.SessionCount,
		VisitorCount:         r.VisitorCount,
		PresentCount:         r.PresentCount,
		PeakConcurrent:       r.PeakConcurrent,
		RepeatVisitorCount:   r.RepeatVisitorCount,
		TotalDurationSeconds: r.TotalDurationSeconds,
		AvgSessionSeconds:    r.AvgSessionSeconds,
		FirstEventAt:         parseTimeFromNullable(r.FirstEventAt),
		LastEventAt:          parseTimeFromNullable(r.LastEventAt),
	}, nil
}

func (s *Server) GetInstanceDiscordMentions(ctx context.Context, request gen.GetInstanceDiscordMentionsRequestObject) (gen.GetInstanceDiscordMentionsResponseObject, error) {
	var (
		ids []string
		err error
	)
	if request.Params.Scope != nil && *request.Params.Scope == gen.GetInstanceDiscordMentionsParamsScopeLastSeen {
		ids, err = s.Locations.DiscordIDsAtClose(ctx, request.InstanceId)
	} else {
		ids, err = s.Locations.DiscordIDsPresent(ctx, request.InstanceId)
	}
	if err != nil {
		return nil, err
	}
	return gen.GetInstanceDiscordMentions200JSONResponse{DiscordIds: ids}, nil
}

func (s *Server) ListDiscordMentions(ctx context.Context, request gen.ListDiscordMentionsRequestObject) (gen.ListDiscordMentionsResponseObject, error) {
	start := timePtrToStrPtr(request.Params.Start)
	end := timePtrToStrPtr(request.Params.End)
	ids, err := s.Locations.ListDiscordMentions(ctx,
		start, end,
		request.Params.GroupId,
		request.Params.WorldId,
		request.Params.Region,
		request.Params.InstanceId,
		request.Params.Present,
	)
	if err != nil {
		return nil, err
	}
	mentions := make([]string, len(ids))
	for i, id := range ids {
		mentions[i] = "@" + id
	}
	return gen.ListDiscordMentions200TextResponse(strings.Join(mentions, " ")), nil
}

func (s *Server) GetInstanceSessions(ctx context.Context, request gen.GetInstanceSessionsRequestObject) (gen.GetInstanceSessionsResponseObject, error) {
	start := timePtrToStrPtr(request.Params.Start)
	end := timePtrToStrPtr(request.Params.End)
	offset := derefInt(request.Params.Offset)
	rows, err := s.Locations.GetLocationSessions(ctx, request.InstanceId, start, end, enumStrOr(request.Params.SortBy, "join_ts"), enumStrOr(request.Params.Order, "asc"), request.Params.Limit, offset)
	if err != nil {
		return nil, err
	}
	out := make(gen.GetInstanceSessions200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, sessionRowToOut(r))
	}
	return out, nil
}
