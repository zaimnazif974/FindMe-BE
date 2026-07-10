package locations

import "github.com/google/uuid"

type ShareRequest struct {
	GroupID     *uuid.UUID `json:"group_id"`
	ShareToAll  bool       `json:"share_to_all"`
	Latitude    *float64   `json:"latitude" binding:"required"`
	Longitude   *float64   `json:"longitude" binding:"required"`
	Accuracy    *float64   `json:"accuracy"`
	Name        *string    `json:"name" binding:"omitempty,max=50"`
	AddressText *string    `json:"address_text"`
	Note        *string    `json:"note"`
}
