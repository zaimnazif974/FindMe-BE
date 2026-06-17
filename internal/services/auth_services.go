package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"findme/backend/internal/apperror"
	authdto "findme/backend/internal/dto/auth"
	userdto "findme/backend/internal/dto/users"
	"findme/backend/internal/middlewares"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	users      authUserRepository
	userSvc    profileService
	jwtSecret  string
	jwtExpires time.Duration
}

type authUserRepository interface {
	Create(context.Context, string, string, string) (userdto.User, error)
	FindByEmail(context.Context, string) (userdto.UserWithPassword, error)
}

type profileService interface {
	Get(context.Context, uuid.UUID) (userdto.User, error)
	Update(context.Context, uuid.UUID, string, *string) (userdto.User, error)
}

func NewAuthService(repository authUserRepository, userSvc profileService, secret string, expires time.Duration) *AuthService {
	return &AuthService{users: repository, userSvc: userSvc, jwtSecret: secret, jwtExpires: expires}
}

func (s *AuthService) Register(ctx context.Context, request authdto.RegisterRequest) (authdto.AuthResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return authdto.AuthResponse{}, err
	}
	user, err := s.users.Create(ctx, request.Name, request.Email, string(hash))
	if err != nil {
		return authdto.AuthResponse{}, err
	}
	return s.authResponse(user.ID, user)
}

func (s *AuthService) Login(ctx context.Context, request authdto.LoginRequest) (authdto.AuthResponse, error) {
	user, err := s.users.FindByEmail(ctx, strings.TrimSpace(request.Email))
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return authdto.AuthResponse{}, apperror.ErrUnauthorized
		}
		return authdto.AuthResponse{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)) != nil {
		return authdto.AuthResponse{}, apperror.ErrUnauthorized
	}
	return s.authResponse(user.ID, user.User)
}

func (s *AuthService) Profile(ctx context.Context, userID uuid.UUID) (userdto.User, error) {
	return s.userSvc.Get(ctx, userID)
}

func (s *AuthService) UpdateProfile(ctx context.Context, userID uuid.UUID, request authdto.UpdateProfileRequest) (userdto.User, error) {
	return s.userSvc.Update(ctx, userID, request.Name, request.AvatarURL)
}

func (s *AuthService) authResponse(userID uuid.UUID, user userdto.User) (authdto.AuthResponse, error) {
	token, err := middlewares.IssueToken(userID, s.jwtSecret, s.jwtExpires)
	if err != nil {
		return authdto.AuthResponse{}, err
	}
	return authdto.AuthResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.jwtExpires.Seconds()),
		User:        user,
	}, nil
}
