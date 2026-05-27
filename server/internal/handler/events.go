package handler

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/njm2360/vrchat-join-manager/server/internal/domain"
	"github.com/njm2360/vrchat-join-manager/server/internal/gen"
	"github.com/njm2360/vrchat-join-manager/server/internal/timeutil"
)

func (s *Server) ReceiveEvent(ctx context.Context, request gen.ReceiveEventRequestObject) (gen.ReceiveEventResponseObject, error) {
	body := request.Body
	if body == nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "missing body")
	}

	loc, err := domain.ParseLocationID(body.LocationId)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	ts := timeutil.FormatUTC(body.Timestamp)

	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	// Update player list
	if err := s.Events.UpsertPlayer(ctx, tx, body.UserId, body.Name, ts); err != nil {
		return nil, err
	}

	switch body.Event {
	case gen.PlayerEventEventJoin:
		// Update world list
		if err := s.Events.UpsertWorld(ctx, tx, loc.WorldID, ts); err != nil {
			return nil, err
		}
		if loc.GroupID != nil {
			// Update group list
			if err := s.Events.UpsertGroup(ctx, tx, *loc.GroupID, ts); err != nil {
				return nil, err
			}
		}
		instanceIDStr := &loc.InstanceID
		region := &loc.Region
		instID, err := s.Instances.GetOrCreate(ctx, tx,
			body.LocationId,
			loc.WorldID,
			instanceIDStr,
			loc.GroupID,
			loc.GroupAccessType,
			region,
			loc.Friends,
			loc.Hidden,
			loc.Private,
			ts,
		)
		if err != nil {
			return nil, err
		}
		// Insert join event
		eventID, err := s.Events.InsertEvent(ctx, tx, "join", instID, loc.WorldID, body.UserId, body.LocationId, ts)
		if err != nil {
			return nil, err
		}
		if eventID != nil {
			// Open user session
			if err := s.Events.OpenSession(ctx, tx, instID, *eventID, loc.WorldID, body.UserId, ts, body.InternalId); err != nil {
				return nil, err
			}
		}

	case gen.PlayerEventEventLeave:
		instID, err := s.Instances.GetOpenInstanceID(ctx, tx, body.LocationId)
		if err != nil {
			return nil, err
		}
		if instID != nil {
			// Insert leave event
			eventID, err := s.Events.InsertEvent(ctx, tx, "leave", *instID, loc.WorldID, body.UserId, body.LocationId, ts)
			if err != nil {
				return nil, err
			}
			if eventID != nil {
				// Close user session
				if err := s.Events.CloseSession(ctx, tx, body.UserId, *instID, *eventID, ts); err != nil {
					return nil, err
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return gen.ReceiveEvent200Response{}, nil
}
