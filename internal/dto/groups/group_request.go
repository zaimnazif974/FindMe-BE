package groups

type CreateRequest struct {
	Name        string  `json:"name" binding:"required,max=100"`
	Description *string `json:"description"`
}

type UpdateRequest struct {
	Name        string  `json:"name" binding:"required,max=100"`
	Description *string `json:"description"`
}

type JoinRequest struct {
	InviteCode string `json:"invite_code" binding:"required,max=20"`
}
