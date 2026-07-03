package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/njm2360/vrchat-join-manager/agent/internal/gen"
)

type ApiClient struct {
	c *gen.Client
}

type ObservedPlayer = gen.ObservedPlayer

func NewApiClient(baseURL string) *ApiClient {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	c, err := gen.NewClient(baseURL, gen.WithHTTPClient(httpClient))
	if err != nil {
		log.Fatalf("NewApiClient: %v", err)
	}
	return &ApiClient{c: c}
}

func drain(resp *http.Response) {
	if resp == nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func okStatus(code int) bool {
	return code >= 200 && code < 300
}

func (a *ApiClient) SendEvent(event, locationID, name, userID string, internalID int, ts time.Time, estimated bool) {
	body := gen.PlayerEvent{
		Event:      gen.PlayerEventEvent(event),
		LocationId: locationID,
		Name:       name,
		UserId:     userID,
		InternalId: internalID,
		Timestamp:  ts.UTC(),
	}
	if estimated {
		body.Estimated = &estimated
	}
	resp, err := a.c.ReceiveEvent(context.Background(), body)
	if err != nil {
		log.Printf("SendEvent failed: %v", err)
		return
	}
	defer drain(resp)
	if !okStatus(resp.StatusCode) {
		log.Printf("SendEvent unexpected status: %d", resp.StatusCode)
	}
}

func (a *ApiClient) Checkin(locationID string, at time.Time, self ObservedPlayer, players []ObservedPlayer) ([]string, error) {
	body := gen.CheckinIn{
		At:      at.UTC(),
		Self:    self,
		Players: players,
	}
	resp, err := a.c.CheckinLocation(context.Background(), locationID, body)
	if err != nil {
		return nil, fmt.Errorf("Checkin request: %w", err)
	}
	defer drain(resp)
	if !okStatus(resp.StatusCode) {
		return nil, fmt.Errorf("Checkin unexpected status: %d", resp.StatusCode)
	}
	var result gen.CheckinOut
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("Checkin decode: %w", err)
	}
	return result.ResumedUserIds, nil
}

func (a *ApiClient) CloseLocation(locationID string, userID string, ts time.Time) {
	body := gen.CloseLocationIn{At: ts.UTC()}
	if userID != "" {
		body.UserId = &userID
	}
	resp, err := a.c.CloseLocation(context.Background(), locationID, body)
	if err != nil {
		log.Printf("CloseLocation failed: %v", err)
		return
	}
	defer drain(resp)
	if resp.StatusCode == http.StatusNotFound {
		return
	}
	if !okStatus(resp.StatusCode) {
		log.Printf("CloseLocation unexpected status: %d", resp.StatusCode)
	}
}
