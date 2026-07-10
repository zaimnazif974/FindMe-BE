package locations

import (
	"time"

	"github.com/google/uuid"
)

type Photo struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	UserName   string    `json:"user_name"`
	UserAvatar *string   `json:"user_avatar"`
	FileName   string    `json:"file_name"`
	MimeType   string    `json:"mime_type"`
	SizeBytes  int64     `json:"size_bytes"`
	URL        string    `json:"url,omitempty"`
	S3Key      string    `json:"-"`
}

type LocationShare struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	UserName    string    `json:"user_name"`
	UserAvatar  *string   `json:"user_avatar"`
	GroupID     uuid.UUID `json:"group_id"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	Accuracy    *float64  `json:"accuracy"`
	Name        *string   `json:"name"`
	AddressText *string   `json:"address_text"`
	Note        *string   `json:"note"`
	Photos      []Photo   `json:"photos" gorm:"-"`
	CreatedAt   time.Time `json:"created_at"`
}
