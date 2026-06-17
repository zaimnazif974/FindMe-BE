package websocket

import "time"

const (
	EventLiveUpdated = "live_location.updated"
	EventLiveStopped = "live_location.stopped"
	EventLiveExpired = "live_location.expired"
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type LiveLocationPayload struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	GroupID   string    `json:"group_id"`
	Latitude  float64   `json:"latitude,omitempty"`
	Longitude float64   `json:"longitude,omitempty"`
	Accuracy  *float64  `json:"accuracy,omitempty"`
	Heading   *float64  `json:"heading,omitempty"`
	Speed     *float64  `json:"speed,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}
