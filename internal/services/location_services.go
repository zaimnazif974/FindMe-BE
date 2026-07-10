package services

import (
	"context"
	"fmt"
	"mime/multipart"

	"findme/backend/internal/apperror"
	locationdto "findme/backend/internal/dto/locations"
	"findme/backend/internal/storage"
	"findme/backend/internal/validator"

	"github.com/google/uuid"
)

type LocationService struct {
	repository locationRepository
	groups     membershipRepository
	storage    *storage.Service
}

type locationRepository interface {
	Share(context.Context, uuid.UUID, locationdto.ShareRequest) ([]locationdto.LocationShare, error)
	List(context.Context, uuid.UUID, bool) ([]locationdto.LocationShare, error)
	Find(context.Context, uuid.UUID) (locationdto.LocationShare, error)
	PhotoCount(context.Context, uuid.UUID) (int, error)
	AddPhoto(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, string, string, string, int64) (locationdto.Photo, error)
	PhotosForShares(context.Context, []uuid.UUID) (map[uuid.UUID][]locationdto.Photo, error)
}

type membershipRepository interface {
	AssertMember(context.Context, uuid.UUID, uuid.UUID) error
}

func NewLocationService(repository locationRepository, groupRepo membershipRepository, storageService *storage.Service) *LocationService {
	return &LocationService{repository: repository, groups: groupRepo, storage: storageService}
}

func (s *LocationService) Share(ctx context.Context, userID uuid.UUID, request locationdto.ShareRequest) ([]locationdto.LocationShare, error) {
	if request.Latitude == nil || request.Longitude == nil {
		return nil, fmt.Errorf("%w: latitude and longitude are required", apperror.ErrBadRequest)
	}
	if err := validator.Coordinates(*request.Latitude, *request.Longitude); err != nil {
		return nil, fmt.Errorf("%w: %v", apperror.ErrBadRequest, err)
	}
	if request.ShareToAll && request.GroupID != nil {
		return nil, fmt.Errorf("%w: choose one group or share_to_all, not both", apperror.ErrBadRequest)
	}
	return s.repository.Share(ctx, userID, request)
}

func (s *LocationService) List(ctx context.Context, groupID, userID uuid.UUID, latest bool) ([]locationdto.LocationShare, error) {
	if err := s.groups.AssertMember(ctx, groupID, userID); err != nil {
		return nil, err
	}
	shares, err := s.repository.List(ctx, groupID, latest)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(shares))
	for i := range shares {
		ids[i] = shares[i].ID
	}
	photos, err := s.repository.PhotosForShares(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range shares {
		shares[i].Photos = photos[shares[i].ID]
		for j := range shares[i].Photos {
			shares[i].Photos[j].URL, _ = s.storage.PresignedDownloadURL(ctx, shares[i].Photos[j].S3Key)
		}
	}
	return shares, nil
}

func (s *LocationService) AddPhotos(ctx context.Context, shareID, userID uuid.UUID, files []*multipart.FileHeader) ([]locationdto.Photo, error) {
	share, err := s.repository.Find(ctx, shareID)
	if err != nil {
		return nil, err
	}
	if err := s.groups.AssertMember(ctx, share.GroupID, userID); err != nil {
		return nil, err
	}
	count, err := s.repository.PhotoCount(ctx, shareID)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 || len(files) > 5 || count+len(files) > 20 {
		return nil, fmt.Errorf("%w: upload 1-5 photos at a time; a location share can have at most 20 photos", apperror.ErrBadRequest)
	}
	for _, file := range files {
		if err := validator.Photo(file); err != nil {
			return nil, fmt.Errorf("%w: %v", apperror.ErrBadRequest, err)
		}
	}

	result := make([]locationdto.Photo, 0, len(files))
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			return nil, err
		}
		photoID := uuid.New()
		filename := validator.SafeFilename(header.Filename)
		key := fmt.Sprintf("location-shares/%s/%s/%s-%s", share.GroupID, shareID, photoID, filename)
		uploadErr := s.storage.Upload(ctx, key, header.Header.Get("Content-Type"), header.Size, file)
		_ = file.Close()
		if uploadErr != nil {
			return nil, uploadErr
		}
		photo, err := s.repository.AddPhoto(ctx, shareID, userID, photoID, s.storage.Bucket(), key, filename, header.Header.Get("Content-Type"), header.Size)
		if err != nil {
			_ = s.storage.Delete(ctx, key)
			return nil, err
		}
		photo.URL, _ = s.storage.PresignedDownloadURL(ctx, key)
		result = append(result, photo)
	}
	return result, nil
}
