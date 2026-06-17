package live_locations

type StartRequest struct {
	DurationMinutes int `json:"duration_minutes"`
}

type UpdateRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
	Accuracy  *float64 `json:"accuracy"`
	Heading   *float64 `json:"heading"`
	Speed     *float64 `json:"speed"`
}
