package repositories

import (
	"context"

	"findme/backend/internal/apperror"
	locationdto "findme/backend/internal/dto/locations"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LocationRepository struct {
	db *gorm.DB
}

func NewLocationRepository(db *gorm.DB) *LocationRepository {
	return &LocationRepository{db: db}
}

func (r *LocationRepository) Share(ctx context.Context, userID uuid.UUID, request locationdto.ShareRequest) ([]locationdto.LocationShare, error) {
	groupIDs := []uuid.UUID{}
	result := []locationdto.LocationShare{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if request.ShareToAll {
			if err := tx.Table("group_members").Where("user_id = ?", userID).Pluck("group_id", &groupIDs).Error; err != nil {
				return err
			}
		} else if request.GroupID != nil {
			var count int64
			if err := tx.Table("group_members").Where("group_id = ? AND user_id = ?", *request.GroupID, userID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return apperror.ErrForbidden
			}
			groupIDs = append(groupIDs, *request.GroupID)
		}
		if len(groupIDs) == 0 {
			return apperror.ErrBadRequest
		}
		result = make([]locationdto.LocationShare, 0, len(groupIDs))
		for _, groupID := range groupIDs {
			var share locationdto.LocationShare
			if err := tx.Raw(`
				INSERT INTO location_shares (user_id, group_id, latitude, longitude, accuracy, name, address_text, note)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				RETURNING id, user_id, group_id, latitude, longitude, accuracy, name, address_text, note, created_at
			`, userID, groupID, *request.Latitude, *request.Longitude, request.Accuracy, request.Name, request.AddressText, request.Note).
				Scan(&share).Error; err != nil {
				return err
			}
			share.Photos = []locationdto.Photo{}
			result = append(result, share)
		}
		return nil
	})
	return result, err
}

func (r *LocationRepository) List(ctx context.Context, groupID uuid.UUID, latest bool) ([]locationdto.LocationShare, error) {
	query := `
		SELECT ls.id, ls.user_id, u.name AS user_name, u.avatar_url AS user_avatar, ls.group_id, ls.latitude, ls.longitude,
		       ls.accuracy, ls.name, ls.address_text, ls.note, ls.created_at
		FROM location_shares ls JOIN users u ON u.id = ls.user_id
		WHERE ls.group_id = ? ORDER BY ls.created_at DESC LIMIT 200
	`
	if latest {
		query = `
			SELECT DISTINCT ON (ls.user_id) ls.id, ls.user_id, u.name AS user_name,
			       u.avatar_url AS user_avatar, ls.group_id,
			       ls.latitude, ls.longitude, ls.accuracy, ls.name, ls.address_text, ls.note, ls.created_at
			FROM location_shares ls JOIN users u ON u.id = ls.user_id
			WHERE ls.group_id = ? ORDER BY ls.user_id, ls.created_at DESC
		`
	}
	result := []locationdto.LocationShare{}
	if err := r.db.WithContext(ctx).Raw(query, groupID).Scan(&result).Error; err != nil {
		return nil, err
	}
	for i := range result {
		share := &result[i]
		share.Photos = []locationdto.Photo{}
	}
	return result, nil
}

func (r *LocationRepository) Find(ctx context.Context, shareID uuid.UUID) (locationdto.LocationShare, error) {
	var share locationdto.LocationShare
	err := r.db.WithContext(ctx).Raw(`
		SELECT ls.id, ls.user_id, u.name AS user_name, u.avatar_url AS user_avatar, ls.group_id, ls.latitude, ls.longitude,
		       ls.accuracy, ls.name, ls.address_text, ls.note, ls.created_at
		FROM location_shares ls JOIN users u ON u.id = ls.user_id WHERE ls.id = ?
	`, shareID).Scan(&share).Error
	if err != nil {
		return locationdto.LocationShare{}, err
	}
	if share.ID == uuid.Nil {
		return locationdto.LocationShare{}, apperror.ErrNotFound
	}
	return share, nil
}

func (r *LocationRepository) PhotoCount(ctx context.Context, shareID uuid.UUID) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("location_share_photos").Where("location_share_id = ?", shareID).Count(&count).Error
	return int(count), err
}

func (r *LocationRepository) AddPhoto(ctx context.Context, shareID, userID, photoID uuid.UUID, bucket, key, filename, mime string, size int64) (locationdto.Photo, error) {
	var photo locationdto.Photo
	err := r.db.WithContext(ctx).Raw(`
		WITH inserted AS (
			INSERT INTO location_share_photos (id, location_share_id, user_id, s3_bucket, s3_key, file_name, mime_type, size_bytes)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			RETURNING id, user_id, file_name, mime_type, size_bytes, s3_key
		)
		SELECT i.id, i.user_id, u.name AS user_name, u.avatar_url AS user_avatar,
		       i.file_name, i.mime_type, i.size_bytes, i.s3_key
		FROM inserted i JOIN users u ON u.id = i.user_id
	`, photoID, shareID, userID, bucket, key, filename, mime, size).Scan(&photo).Error
	return photo, err
}

func (r *LocationRepository) PhotosForShares(ctx context.Context, shareIDs []uuid.UUID) (map[uuid.UUID][]locationdto.Photo, error) {
	result := make(map[uuid.UUID][]locationdto.Photo)
	if len(shareIDs) == 0 {
		return result, nil
	}
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT p.location_share_id, p.id, p.user_id, u.name AS user_name, u.avatar_url AS user_avatar,
		       p.file_name, p.mime_type, p.size_bytes, p.s3_key
		FROM location_share_photos p JOIN users u ON u.id = p.user_id
		WHERE p.location_share_id IN ?
		ORDER BY p.created_at
	`, shareIDs).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var shareID uuid.UUID
		var photo locationdto.Photo
		if err := rows.Scan(&shareID, &photo.ID, &photo.UserID, &photo.UserName, &photo.UserAvatar, &photo.FileName, &photo.MimeType, &photo.SizeBytes, &photo.S3Key); err != nil {
			return nil, err
		}
		result[shareID] = append(result[shareID], photo)
	}
	return result, rows.Err()
}
