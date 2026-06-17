package services

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"

	"findme/backend/internal/apperror"
	groupdto "findme/backend/internal/dto/groups"

	"github.com/google/uuid"
)

type GroupService struct {
	repository groupRepository
}

type groupRepository interface {
	Create(context.Context, uuid.UUID, string, *string, string) (groupdto.Group, error)
	Join(context.Context, uuid.UUID, string) (groupdto.Group, error)
	List(context.Context, uuid.UUID) ([]groupdto.Group, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (groupdto.Group, error)
	Members(context.Context, uuid.UUID) ([]groupdto.Member, error)
	AssertMember(context.Context, uuid.UUID, uuid.UUID) error
	IsAdmin(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	Update(context.Context, uuid.UUID, string, *string) (groupdto.Group, error)
	Delete(context.Context, uuid.UUID) error
	Leave(context.Context, uuid.UUID, uuid.UUID) error
	RemoveMember(context.Context, uuid.UUID, uuid.UUID) error
	RegenerateInviteCode(context.Context, uuid.UUID, string) (string, error)
}

func NewGroupService(repository groupRepository) *GroupService {
	return &GroupService{repository: repository}
}

func (s *GroupService) Create(ctx context.Context, userID uuid.UUID, request groupdto.CreateRequest) (groupdto.Group, error) {
	if strings.TrimSpace(request.Name) == "" {
		return groupdto.Group{}, fmt.Errorf("%w: group name is required", apperror.ErrBadRequest)
	}
	return s.repository.Create(ctx, userID, request.Name, request.Description, inviteCode())
}

func (s *GroupService) Join(ctx context.Context, userID uuid.UUID, code string) (groupdto.Group, error) {
	return s.repository.Join(ctx, userID, code)
}

func (s *GroupService) List(ctx context.Context, userID uuid.UUID) ([]groupdto.Group, error) {
	return s.repository.List(ctx, userID)
}

func (s *GroupService) Get(ctx context.Context, groupID, userID uuid.UUID) (groupdto.Group, error) {
	return s.repository.Get(ctx, groupID, userID)
}

func (s *GroupService) Members(ctx context.Context, groupID, userID uuid.UUID) ([]groupdto.Member, error) {
	if err := s.repository.AssertMember(ctx, groupID, userID); err != nil {
		return nil, err
	}
	return s.repository.Members(ctx, groupID)
}

func (s *GroupService) Update(ctx context.Context, groupID, userID uuid.UUID, request groupdto.UpdateRequest) (groupdto.Group, error) {
	if err := s.requireAdmin(ctx, groupID, userID); err != nil {
		return groupdto.Group{}, err
	}
	if strings.TrimSpace(request.Name) == "" {
		return groupdto.Group{}, fmt.Errorf("%w: group name is required", apperror.ErrBadRequest)
	}
	group, err := s.repository.Update(ctx, groupID, request.Name, request.Description)
	group.Role = "admin"
	return group, err
}

func (s *GroupService) Delete(ctx context.Context, groupID, userID uuid.UUID) error {
	if err := s.requireAdmin(ctx, groupID, userID); err != nil {
		return err
	}
	group, err := s.repository.Get(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if group.CreatedBy != userID {
		return apperror.ErrForbidden
	}
	return s.repository.Delete(ctx, groupID)
}

func (s *GroupService) Leave(ctx context.Context, groupID, userID uuid.UUID) error {
	group, err := s.repository.Get(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if group.CreatedBy == userID {
		return fmt.Errorf("%w: the group creator must delete the group instead", apperror.ErrConflict)
	}
	return s.repository.Leave(ctx, groupID, userID)
}

func (s *GroupService) RemoveMember(ctx context.Context, groupID, userID, memberID uuid.UUID) error {
	if err := s.requireAdmin(ctx, groupID, userID); err != nil {
		return err
	}
	if userID == memberID {
		return fmt.Errorf("%w: use the leave endpoint", apperror.ErrBadRequest)
	}
	return s.repository.RemoveMember(ctx, groupID, memberID)
}

func (s *GroupService) RegenerateInviteCode(ctx context.Context, groupID, userID uuid.UUID) (string, error) {
	if err := s.requireAdmin(ctx, groupID, userID); err != nil {
		return "", err
	}
	return s.repository.RegenerateInviteCode(ctx, groupID, inviteCode())
}

func (s *GroupService) requireAdmin(ctx context.Context, groupID, userID uuid.UUID) error {
	ok, err := s.repository.IsAdmin(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return apperror.ErrForbidden
	}
	return nil
}

func inviteCode() string {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return strings.ToUpper(uuid.NewString()[:10])
	}
	return strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes), "=")
}
