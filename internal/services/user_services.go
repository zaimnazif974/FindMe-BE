package services

import (
	"context"
	"fmt"
	"strings"

	userdto "findme/backend/internal/dto/users"

	"github.com/google/uuid"
)

type UserService struct {
	repository userRepository
}

type userRepository interface {
	FindByID(context.Context, uuid.UUID) (userdto.User, error)
	Update(context.Context, uuid.UUID, string, *string) (userdto.User, error)
}

func NewUserService(repository userRepository) *UserService {
	return &UserService{repository: repository}
}

func (s *UserService) Get(ctx context.Context, userID uuid.UUID) (userdto.User, error) {
	return s.repository.FindByID(ctx, userID)
}

func (s *UserService) Update(ctx context.Context, userID uuid.UUID, name string, avatarURL *string) (userdto.User, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return userdto.User{}, fmt.Errorf("name must be between 1 and 100 characters")
	}
	return s.repository.Update(ctx, userID, name, avatarURL)
}
