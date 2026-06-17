package live_locations

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	UserName  string     `json:"user_name,omitempty"`
	GroupID   uuid.UUID  `json:"group_id"`
	Status    string     `json:"status"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
	ExpiresAt time.Time  `json:"expires_at"`
}

type ActivePosition struct {
	Session
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Accuracy  *float64  `json:"accuracy"`
	Heading   *float64  `json:"heading"`
	Speed     *float64  `json:"speed"`
	UpdatedAt time.Time `json:"updated_at"`
}
