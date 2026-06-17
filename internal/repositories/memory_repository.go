package repositories

import (
	"context"

	"findme/backend/internal/apperror"
	memorydto "findme/backend/internal/dto/memories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MemoryRepository struct {
	db *gorm.DB
}

func NewMemoryRepository(db *gorm.DB) *MemoryRepository {
	return &MemoryRepository{db: db}
}

func (r *MemoryRepository) Create(ctx context.Context, groupID, userID uuid.UUID, request memorydto.CreateRequest) (memorydto.MemoryPoint, error) {
	var point memorydto.MemoryPoint
	err := r.db.WithContext(ctx).Raw(`
		WITH inserted AS (
			INSERT INTO memory_points (group_id, created_by, title, description, latitude, longitude, address_text)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			RETURNING *
		)
		SELECT i.id, i.group_id, i.created_by, u.name AS creator_name, i.title, i.description, i.latitude,
		       i.longitude, i.address_text, i.average_rating, i.created_at, i.updated_at
		FROM inserted i JOIN users u ON u.id = i.created_by
	`, groupID, userID, request.Title, request.Description, *request.Latitude, *request.Longitude, request.AddressText).Scan(&point).Error
	point.Photos = []memorydto.Photo{}
	return point, err
}

func (r *MemoryRepository) List(ctx context.Context, groupID uuid.UUID) ([]memorydto.MemoryPoint, error) {
	result := []memorydto.MemoryPoint{}
	err := r.db.WithContext(ctx).Raw(`
		SELECT m.id, m.group_id, m.created_by, u.name AS creator_name, m.title, m.description, m.latitude,
		       m.longitude, m.address_text, m.average_rating, m.created_at, m.updated_at
		FROM memory_points m JOIN users u ON u.id = m.created_by
		WHERE m.group_id = ? ORDER BY m.updated_at DESC
	`, groupID).Scan(&result).Error
	if err != nil {
		return nil, err
	}
	for i := range result {
		result[i].Photos = []memorydto.Photo{}
	}
	return result, nil
}

func (r *MemoryRepository) Get(ctx context.Context, pointID uuid.UUID) (memorydto.MemoryPoint, error) {
	var point memorydto.MemoryPoint
	err := r.db.WithContext(ctx).Raw(`
		SELECT m.id, m.group_id, m.created_by, u.name AS creator_name, m.title, m.description, m.latitude,
		       m.longitude, m.address_text, m.average_rating, m.created_at, m.updated_at
		FROM memory_points m JOIN users u ON u.id = m.created_by WHERE m.id = ?
	`, pointID).Scan(&point).Error
	if err != nil {
		return memorydto.MemoryPoint{}, err
	}
	if point.ID == uuid.Nil {
		return memorydto.MemoryPoint{}, apperror.ErrNotFound
	}
	point.Photos = []memorydto.Photo{}
	return point, nil
}

func (r *MemoryRepository) Update(ctx context.Context, pointID uuid.UUID, request memorydto.UpdateRequest) (memorydto.MemoryPoint, error) {
	var point memorydto.MemoryPoint
	err := r.db.WithContext(ctx).Raw(`
		WITH updated AS (
			UPDATE memory_points SET title = ?, description = ?, address_text = ?, updated_at = now()
			WHERE id = ? RETURNING *
		)
		SELECT m.id, m.group_id, m.created_by, u.name AS creator_name, m.title, m.description, m.latitude,
		       m.longitude, m.address_text, m.average_rating, m.created_at, m.updated_at
		FROM updated m JOIN users u ON u.id = m.created_by
	`, request.Title, request.Description, request.AddressText, pointID).Scan(&point).Error
	return point, err
}

func (r *MemoryRepository) Delete(ctx context.Context, pointID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM memory_points WHERE id = ?", pointID).Error
}

func (r *MemoryRepository) Rate(ctx context.Context, pointID, userID uuid.UUID, value int) (memorydto.Rating, error) {
	var average float64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rating := map[string]any{
			"memory_point_id": pointID,
			"user_id":         userID,
			"rating_value":    value,
		}
		if err := tx.Table("memory_point_ratings").Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "memory_point_id"}, {Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{"rating_value": value, "updated_at": gorm.Expr("now()")}),
		}).Create(rating).Error; err != nil {
			return err
		}
		return tx.Raw(`
			UPDATE memory_points
			SET average_rating = (
				SELECT COALESCE(avg(rating_value), 0) FROM memory_point_ratings WHERE memory_point_id = ?
			), updated_at = now()
			WHERE id = ? RETURNING average_rating
		`, pointID, pointID).Scan(&average).Error
	})
	if err != nil {
		return memorydto.Rating{}, err
	}
	return memorydto.Rating{MemoryPointID: pointID, UserID: userID, RatingValue: value, AverageRating: average}, nil
}

func (r *MemoryRepository) AddComment(ctx context.Context, pointID, userID uuid.UUID, text string) (memorydto.Comment, error) {
	var comment memorydto.Comment
	err := r.db.WithContext(ctx).Raw(`
		WITH inserted AS (
			INSERT INTO memory_point_comments (memory_point_id, user_id, comment_text)
			VALUES (?, ?, ?) RETURNING *
		)
		SELECT i.id, i.memory_point_id, i.user_id, u.name AS user_name, i.comment_text, i.created_at, i.updated_at
		FROM inserted i JOIN users u ON u.id = i.user_id
	`, pointID, userID, text).Scan(&comment).Error
	return comment, err
}

func (r *MemoryRepository) Comments(ctx context.Context, pointID uuid.UUID) ([]memorydto.Comment, error) {
	result := []memorydto.Comment{}
	err := r.db.WithContext(ctx).Raw(`
		SELECT c.id, c.memory_point_id, c.user_id, u.name AS user_name, c.comment_text, c.created_at, c.updated_at
		FROM memory_point_comments c JOIN users u ON u.id = c.user_id
		WHERE c.memory_point_id = ? ORDER BY c.created_at
	`, pointID).Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *MemoryRepository) PhotoCount(ctx context.Context, pointID uuid.UUID) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("memory_point_photos").Where("memory_point_id = ?", pointID).Count(&count).Error
	return int(count), err
}

func (r *MemoryRepository) AddPhoto(ctx context.Context, pointID, userID, photoID uuid.UUID, bucket, key, filename, mime string, size int64) (memorydto.Photo, error) {
	var photo memorydto.Photo
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO memory_point_photos (id, memory_point_id, user_id, s3_bucket, s3_key, file_name, mime_type, size_bytes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, file_name, mime_type, size_bytes, s3_key
	`, photoID, pointID, userID, bucket, key, filename, mime, size).Scan(&photo).Error
	return photo, err
}

func (r *MemoryRepository) PhotosForPoints(ctx context.Context, pointIDs []uuid.UUID) (map[uuid.UUID][]memorydto.Photo, error) {
	result := make(map[uuid.UUID][]memorydto.Photo)
	if len(pointIDs) == 0 {
		return result, nil
	}
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT memory_point_id, id, file_name, mime_type, size_bytes, s3_key
		FROM memory_point_photos WHERE memory_point_id IN ?
		ORDER BY created_at
	`, pointIDs).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pointID uuid.UUID
		var photo memorydto.Photo
		if err := rows.Scan(&pointID, &photo.ID, &photo.FileName, &photo.MimeType, &photo.SizeBytes, &photo.S3Key); err != nil {
			return nil, err
		}
		result[pointID] = append(result[pointID], photo)
	}
	return result, rows.Err()
}
