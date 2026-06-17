package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"findme/backend/internal/apperror"
	livedto "findme/backend/internal/dto/live_locations"
	"findme/backend/internal/validator"
	ws "findme/backend/internal/websocket"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type LiveLocationService struct {
	repository liveLocationRepository
	groups     liveGroupRepository
	redis      *redis.Client
	hub        *ws.Hub
}

type liveLocationRepository interface {
	Start(context.Context, uuid.UUID, uuid.UUID, time.Time) (livedto.Session, error)
	Update(context.Context, uuid.UUID, uuid.UUID, livedto.UpdateRequest) (livedto.ActivePosition, error)
	Stop(context.Context, uuid.UUID, uuid.UUID) (livedto.Session, error)
	Active(context.Context, uuid.UUID) ([]livedto.ActivePosition, error)
	Expire(context.Context) ([]livedto.Session, error)
}

type liveGroupRepository interface {
	AssertMember(context.Context, uuid.UUID, uuid.UUID) error
}

func NewLiveLocationService(repository liveLocationRepository, groupRepo liveGroupRepository, redisClient *redis.Client, hub *ws.Hub) *LiveLocationService {
	return &LiveLocationService{repository: repository, groups: groupRepo, redis: redisClient, hub: hub}
}

func (s *LiveLocationService) Start(ctx context.Context, groupID, userID uuid.UUID, durationMinutes int) (livedto.Session, error) {
	if err := s.groups.AssertMember(ctx, groupID, userID); err != nil {
		return livedto.Session{}, err
	}
	if durationMinutes == 0 {
		durationMinutes = 60
	}
	if durationMinutes < 1 || durationMinutes > 240 {
		return livedto.Session{}, fmt.Errorf("%w: duration must be between 1 and 240 minutes", apperror.ErrBadRequest)
	}
	return s.repository.Start(ctx, userID, groupID, time.Now().Add(time.Duration(durationMinutes)*time.Minute))
}

func (s *LiveLocationService) Update(ctx context.Context, groupID, userID uuid.UUID, request livedto.UpdateRequest) (livedto.ActivePosition, error) {
	if request.Latitude == nil || request.Longitude == nil {
		return livedto.ActivePosition{}, fmt.Errorf("%w: latitude and longitude are required", apperror.ErrBadRequest)
	}
	if err := validator.Coordinates(*request.Latitude, *request.Longitude); err != nil {
		return livedto.ActivePosition{}, fmt.Errorf("%w: %v", apperror.ErrBadRequest, err)
	}
	position, err := s.repository.Update(ctx, userID, groupID, request)
	if err != nil {
		return livedto.ActivePosition{}, err
	}
	cacheKey := fmt.Sprintf("live:group:%s:user:%s", groupID, userID)
	if payload, marshalErr := json.Marshal(position); marshalErr == nil {
		_ = s.redis.Set(ctx, cacheKey, payload, time.Until(position.ExpiresAt)).Err()
	}
	s.hub.Broadcast(groupID.String(), ws.Event{
		Type: ws.EventLiveUpdated,
		Data: ws.LiveLocationPayload{
			SessionID: position.ID.String(), UserID: userID.String(), GroupID: groupID.String(),
			Latitude: position.Latitude, Longitude: position.Longitude, Accuracy: position.Accuracy,
			Heading: position.Heading, Speed: position.Speed, UpdatedAt: position.UpdatedAt,
		},
	})
	return position, nil
}

func (s *LiveLocationService) Stop(ctx context.Context, groupID, userID uuid.UUID) (livedto.Session, error) {
	session, err := s.repository.Stop(ctx, userID, groupID)
	if err != nil {
		return livedto.Session{}, err
	}
	_ = s.redis.Del(ctx, fmt.Sprintf("live:group:%s:user:%s", groupID, userID)).Err()
	s.broadcastSession(ws.EventLiveStopped, session)
	return session, nil
}

func (s *LiveLocationService) Active(ctx context.Context, groupID, userID uuid.UUID) ([]livedto.ActivePosition, error) {
	if err := s.groups.AssertMember(ctx, groupID, userID); err != nil {
		return nil, err
	}
	return s.repository.Active(ctx, groupID)
}

func (s *LiveLocationService) RunExpirationWorker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sessions, err := s.repository.Expire(ctx)
			if err != nil {
				continue
			}
			for _, session := range sessions {
				_ = s.redis.Del(ctx, fmt.Sprintf("live:group:%s:user:%s", session.GroupID, session.UserID)).Err()
				s.broadcastSession(ws.EventLiveExpired, session)
			}
		}
	}
}

func (s *LiveLocationService) broadcastSession(eventType string, session livedto.Session) {
	s.hub.Broadcast(session.GroupID.String(), ws.Event{
		Type: eventType,
		Data: ws.LiveLocationPayload{
			SessionID: session.ID.String(),
			UserID:    session.UserID.String(),
			GroupID:   session.GroupID.String(),
		},
	})
}
