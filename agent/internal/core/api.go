package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type ApiClient struct {
	baseURL string
	client  *http.Client
}

func NewApiClient(baseURL string) *ApiClient {
	return &ApiClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

type eventPayload struct {
	Event      string `json:"event"`
	LocationID string `json:"location_id"`
	Name       string `json:"name"`
	UserID     string `json:"user_id"`
	InternalID *int   `json:"internal_id"`
	Timestamp  string `json:"timestamp"`
}

func (a *ApiClient) SendEvent(event, locationID, name, userID string, internalID *int, ts time.Time) {
	payload := eventPayload{
		Event:      event,
		LocationID: locationID,
		Name:       name,
		UserID:     userID,
		InternalID: internalID,
		Timestamp:  ts.UTC().Format("2006-01-02T15:04:05Z"),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("SendEvent marshal: %v", err)
		return
	}
	resp, err := a.client.Post(a.baseURL+"/api/events", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("SendEvent failed: %v", err)
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("SendEvent unexpected status: %d", resp.StatusCode)
	}
}

func (a *ApiClient) CloseLocation(locationID string, userID string, ts time.Time) {
	type closeBody struct {
		At     string `json:"at"`
		UserID string `json:"user_id,omitempty"`
	}
	payload := closeBody{
		At:     ts.UTC().Format("2006-01-02T15:04:05Z"),
		UserID: userID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("CloseLocation marshal: %v", err)
		return
	}
	url := fmt.Sprintf("%s/api/locations/%s/close", a.baseURL, locationID)
	resp, err := a.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("CloseLocation failed: %v", err)
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("CloseLocation unexpected status: %d", resp.StatusCode)
	}
}
