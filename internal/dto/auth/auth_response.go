package auth

import userdto "findme/backend/internal/dto/users"

type AuthResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int64        `json:"expires_in"`
	User        userdto.User `json:"user"`
}
