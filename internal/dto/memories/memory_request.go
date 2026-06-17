package memories

type CreateRequest struct {
	Title       string   `json:"title" binding:"required,max=150"`
	Description *string  `json:"description"`
	Latitude    *float64 `json:"latitude" binding:"required"`
	Longitude   *float64 `json:"longitude" binding:"required"`
	AddressText *string  `json:"address_text"`
}

type UpdateRequest struct {
	Title       string  `json:"title" binding:"required,max=150"`
	Description *string `json:"description"`
	AddressText *string `json:"address_text"`
}

type RatingRequest struct {
	RatingValue int `json:"rating_value" binding:"required,min=1,max=5"`
}

type CommentRequest struct {
	CommentText string `json:"comment_text" binding:"required,max=2000"`
}
