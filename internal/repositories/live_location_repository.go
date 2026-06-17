package repositories

import (
	"context"
	"time"

	"findme/backend/internal/apperror"
	livedto "findme/backend/internal/dto/live_locations"
	"findme/backend/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LiveLocationRepository struct {
	db *gorm.DB
}

func NewLiveLocationRepository(db *gorm.DB) *LiveLocationRepository {
	return &LiveLocationRepository{db: db}
}

func (r *LiveLocationRepository) Start(ctx context.Context, userID, groupID uuid.UUID, expiresAt time.Time) (livedto.Session, error) {
	var session livedto.Session
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO live_location_sessions (user_id, group_id, status, expires_at)
		VALUES (?, ?, 'active', ?)
		RETURNING id, user_id, group_id, status, started_at, ended_at, expires_at
	`, userID, groupID, expiresAt).Scan(&session).Error
	if err != nil && utils.IsUniqueViolation(err) {
		return livedto.Session{}, apperror.ErrActiveLiveExists
	}
	return session, err
}

func (r *LiveLocationRepository) Update(ctx context.Context, userID, groupID uuid.UUID, request livedto.UpdateRequest) (livedto.ActivePosition, error) {
	var session livedto.Session
	var position livedto.ActivePosition
	expired := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Table("live_location_sessions").
			Where("user_id = ? AND group_id = ? AND status = 'active'", userID, groupID).
			Clauses(clause.Locking{Strength: "UPDATE"}).Take(&session).Error
		if err != nil {
			return utils.DatabaseError(err)
		}
		if time.Now().After(session.ExpiresAt) {
			if err := tx.Table("live_location_sessions").Where("id = ?", session.ID).
				Updates(map[string]any{"status": "expired", "ended_at": gorm.Expr("now()")}).Error; err != nil {
				return err
			}
			expired = true
			return nil
		}
		position.Session = session
		return tx.Raw(`
			INSERT INTO live_location_latest_positions
				(session_id, user_id, group_id, latitude, longitude, accuracy, heading, speed)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (session_id) DO UPDATE SET
				latitude = excluded.latitude, longitude = excluded.longitude, accuracy = excluded.accuracy,
				heading = excluded.heading, speed = excluded.speed, updated_at = now()
			RETURNING latitude, longitude, accuracy, heading, speed, updated_at
		`, session.ID, userID, groupID, *request.Latitude, *request.Longitude, request.Accuracy, request.Heading, request.Speed).
			Scan(&position).Error
	})
	if err != nil {
		return livedto.ActivePosition{}, err
	}
	if expired {
		return livedto.ActivePosition{}, apperror.ErrNotFound
	}
	return position, nil
}

func (r *LiveLocationRepository) Stop(ctx context.Context, userID, groupID uuid.UUID) (livedto.Session, error) {
	var session livedto.Session
	err := r.db.WithContext(ctx).Raw(`
		UPDATE live_location_sessions
		SET status = 'stopped', ended_at = now()
		WHERE user_id = ? AND group_id = ? AND status = 'active'
		RETURNING id, user_id, group_id, status, started_at, ended_at, expires_at
	`, userID, groupID).Scan(&session).Error
	if err != nil {
		return livedto.Session{}, err
	}
	if session.ID == uuid.Nil {
		return livedto.Session{}, apperror.ErrNotFound
	}
	return session, nil
}

func (r *LiveLocationRepository) Active(ctx context.Context, groupID uuid.UUID) ([]livedto.ActivePosition, error) {
	result := []livedto.ActivePosition{}
	err := r.db.WithContext(ctx).Raw(`
		SELECT s.id, s.user_id, u.name AS user_name, s.group_id, s.status, s.started_at, s.ended_at, s.expires_at,
		       p.latitude, p.longitude, p.accuracy, p.heading, p.speed, p.updated_at
		FROM live_location_sessions s
		JOIN users u ON u.id = s.user_id
		JOIN live_location_latest_positions p ON p.session_id = s.id
		WHERE s.group_id = ? AND s.status = 'active' AND s.expires_at > now()
		ORDER BY p.updated_at DESC
	`, groupID).Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *LiveLocationRepository) Expire(ctx context.Context) ([]livedto.Session, error) {
	result := []livedto.Session{}
	err := r.db.WithContext(ctx).Raw(`
		UPDATE live_location_sessions SET status = 'expired', ended_at = now()
		WHERE status = 'active' AND expires_at <= now()
		RETURNING id, user_id, group_id, status, started_at, ended_at, expires_at
	`).Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}
