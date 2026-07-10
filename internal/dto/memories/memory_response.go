package memories

import (
	"time"

	"github.com/google/uuid"
)

type Photo struct {
	ID        uuid.UUID `json:"id"`
	FileName  string    `json:"file_name"`
	MimeType  string    `json:"mime_type"`
	SizeBytes int64     `json:"size_bytes"`
	URL       string    `json:"url,omitempty"`
	S3Key     string    `json:"-"`
}

type MemoryPoint struct {
	ID            uuid.UUID `json:"id"`
	GroupID       uuid.UUID `json:"group_id"`
	CreatedBy     uuid.UUID `json:"created_by"`
	CreatorName   string    `json:"creator_name"`
	Title         string    `json:"title"`
	Description   *string   `json:"description"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	AddressText   *string   `json:"address_text"`
	AverageRating float64   `json:"average_rating"`
	Photos        []Photo   `json:"photos" gorm:"-"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Rating struct {
	MemoryPointID uuid.UUID `json:"memory_point_id"`
	UserID        uuid.UUID `json:"user_id"`
	RatingValue   int       `json:"rating_value"`
	AverageRating float64   `json:"average_rating"`
}

type Comment struct {
	ID            uuid.UUID `json:"id"`
	MemoryPointID uuid.UUID `json:"memory_point_id"`
	UserID        uuid.UUID `json:"user_id"`
	UserName      string    `json:"user_name"`
	CommentText   string    `json:"comment_text"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
