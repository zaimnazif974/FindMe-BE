package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"

	"findme/backend/internal/apperror"
	memorydto "findme/backend/internal/dto/memories"
	"findme/backend/internal/storage"
	"findme/backend/internal/validator"

	"github.com/google/uuid"
)

type MemoryService struct {
	repository memoryRepository
	groups     groupAccessRepository
	storage    *storage.Service
}

type memoryRepository interface {
	Create(context.Context, uuid.UUID, uuid.UUID, memorydto.CreateRequest) (memorydto.MemoryPoint, error)
	List(context.Context, uuid.UUID) ([]memorydto.MemoryPoint, error)
	Get(context.Context, uuid.UUID) (memorydto.MemoryPoint, error)
	Update(context.Context, uuid.UUID, memorydto.UpdateRequest) (memorydto.MemoryPoint, error)
	Delete(context.Context, uuid.UUID) error
	Rate(context.Context, uuid.UUID, uuid.UUID, int) (memorydto.Rating, error)
	AddComment(context.Context, uuid.UUID, uuid.UUID, string) (memorydto.Comment, error)
	Comments(context.Context, uuid.UUID) ([]memorydto.Comment, error)
	PhotoCount(context.Context, uuid.UUID) (int, error)
	AddPhoto(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, string, string, string, int64) (memorydto.Photo, error)
	PhotosForPoints(context.Context, []uuid.UUID) (map[uuid.UUID][]memorydto.Photo, error)
}

type groupAccessRepository interface {
	AssertMember(context.Context, uuid.UUID, uuid.UUID) error
	IsAdmin(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}

func NewMemoryService(repository memoryRepository, groupRepo groupAccessRepository, storageService *storage.Service) *MemoryService {
	return &MemoryService{repository: repository, groups: groupRepo, storage: storageService}
}

func (s *MemoryService) Create(ctx context.Context, groupID, userID uuid.UUID, request memorydto.CreateRequest) (memorydto.MemoryPoint, error) {
	if err := s.groups.AssertMember(ctx, groupID, userID); err != nil {
		return memorydto.MemoryPoint{}, err
	}
	if strings.TrimSpace(request.Title) == "" {
		return memorydto.MemoryPoint{}, fmt.Errorf("%w: title is required", apperror.ErrBadRequest)
	}
	if request.Latitude == nil || request.Longitude == nil {
		return memorydto.MemoryPoint{}, fmt.Errorf("%w: latitude and longitude are required", apperror.ErrBadRequest)
	}
	if err := validator.Coordinates(*request.Latitude, *request.Longitude); err != nil {
		return memorydto.MemoryPoint{}, fmt.Errorf("%w: %v", apperror.ErrBadRequest, err)
	}
	return s.repository.Create(ctx, groupID, userID, request)
}

func (s *MemoryService) List(ctx context.Context, groupID, userID uuid.UUID) ([]memorydto.MemoryPoint, error) {
	if err := s.groups.AssertMember(ctx, groupID, userID); err != nil {
		return nil, err
	}
	points, err := s.repository.List(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return s.withPhotos(ctx, points)
}

func (s *MemoryService) Get(ctx context.Context, pointID, userID uuid.UUID) (memorydto.MemoryPoint, error) {
	point, err := s.repository.Get(ctx, pointID)
	if err != nil {
		return memorydto.MemoryPoint{}, err
	}
	if err := s.groups.AssertMember(ctx, point.GroupID, userID); err != nil {
		return memorydto.MemoryPoint{}, err
	}
	points, err := s.withPhotos(ctx, []memorydto.MemoryPoint{point})
	if err != nil {
		return memorydto.MemoryPoint{}, err
	}
	return points[0], nil
}

func (s *MemoryService) Update(ctx context.Context, pointID, userID uuid.UUID, request memorydto.UpdateRequest) (memorydto.MemoryPoint, error) {
	point, err := s.authorizeOwnerOrAdmin(ctx, pointID, userID)
	if err != nil {
		return memorydto.MemoryPoint{}, err
	}
	if strings.TrimSpace(request.Title) == "" {
		return memorydto.MemoryPoint{}, fmt.Errorf("%w: title is required", apperror.ErrBadRequest)
	}
	updated, err := s.repository.Update(ctx, point.ID, request)
	updated.Photos = point.Photos
	return updated, err
}

func (s *MemoryService) Delete(ctx context.Context, pointID, userID uuid.UUID) error {
	if _, err := s.authorizeOwnerOrAdmin(ctx, pointID, userID); err != nil {
		return err
	}
	return s.repository.Delete(ctx, pointID)
}

func (s *MemoryService) Rate(ctx context.Context, pointID, userID uuid.UUID, value int) (memorydto.Rating, error) {
	point, err := s.repository.Get(ctx, pointID)
	if err != nil {
		return memorydto.Rating{}, err
	}
	if err := s.groups.AssertMember(ctx, point.GroupID, userID); err != nil {
		return memorydto.Rating{}, err
	}
	if value < 1 || value > 5 {
		return memorydto.Rating{}, fmt.Errorf("%w: rating must be between 1 and 5", apperror.ErrBadRequest)
	}
	return s.repository.Rate(ctx, pointID, userID, value)
}

func (s *MemoryService) AddComment(ctx context.Context, pointID, userID uuid.UUID, text string) (memorydto.Comment, error) {
	point, err := s.repository.Get(ctx, pointID)
	if err != nil {
		return memorydto.Comment{}, err
	}
	if err := s.groups.AssertMember(ctx, point.GroupID, userID); err != nil {
		return memorydto.Comment{}, err
	}
	text = strings.TrimSpace(text)
	if text == "" || len(text) > 2000 {
		return memorydto.Comment{}, fmt.Errorf("%w: comment must be between 1 and 2000 characters", apperror.ErrBadRequest)
	}
	return s.repository.AddComment(ctx, pointID, userID, text)
}

func (s *MemoryService) Comments(ctx context.Context, pointID, userID uuid.UUID) ([]memorydto.Comment, error) {
	point, err := s.repository.Get(ctx, pointID)
	if err != nil {
		return nil, err
	}
	if err := s.groups.AssertMember(ctx, point.GroupID, userID); err != nil {
		return nil, err
	}
	return s.repository.Comments(ctx, pointID)
}

func (s *MemoryService) AddPhotos(ctx context.Context, pointID, userID uuid.UUID, files []*multipart.FileHeader) ([]memorydto.Photo, error) {
	point, err := s.repository.Get(ctx, pointID)
	if err != nil {
		return nil, err
	}
	if err := s.groups.AssertMember(ctx, point.GroupID, userID); err != nil {
		return nil, err
	}
	count, err := s.repository.PhotoCount(ctx, pointID)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 || count+len(files) > 5 {
		return nil, fmt.Errorf("%w: a memory point can have at most 5 photos", apperror.ErrBadRequest)
	}
	for _, file := range files {
		if err := validator.Photo(file); err != nil {
			return nil, fmt.Errorf("%w: %v", apperror.ErrBadRequest, err)
		}
	}
	result := make([]memorydto.Photo, 0, len(files))
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			return nil, err
		}
		photoID := uuid.New()
		filename := validator.SafeFilename(header.Filename)
		key := fmt.Sprintf("memory-points/%s/%s/%s-%s", point.GroupID, pointID, photoID, filename)
		uploadErr := s.storage.Upload(ctx, key, header.Header.Get("Content-Type"), header.Size, file)
		_ = file.Close()
		if uploadErr != nil {
			return nil, uploadErr
		}
		photo, err := s.repository.AddPhoto(ctx, pointID, userID, photoID, s.storage.Bucket(), key, filename, header.Header.Get("Content-Type"), header.Size)
		if err != nil {
			_ = s.storage.Delete(ctx, key)
			return nil, err
		}
		photo.URL, _ = s.storage.PresignedDownloadURL(ctx, key)
		result = append(result, photo)
	}
	return result, nil
}

func (s *MemoryService) authorizeOwnerOrAdmin(ctx context.Context, pointID, userID uuid.UUID) (memorydto.MemoryPoint, error) {
	point, err := s.repository.Get(ctx, pointID)
	if err != nil {
		return memorydto.MemoryPoint{}, err
	}
	if point.CreatedBy == userID {
		return point, nil
	}
	admin, err := s.groups.IsAdmin(ctx, point.GroupID, userID)
	if err != nil {
		return memorydto.MemoryPoint{}, err
	}
	if !admin {
		return memorydto.MemoryPoint{}, apperror.ErrForbidden
	}
	return point, nil
}

func (s *MemoryService) withPhotos(ctx context.Context, points []memorydto.MemoryPoint) ([]memorydto.MemoryPoint, error) {
	ids := make([]uuid.UUID, len(points))
	for i := range points {
		ids[i] = points[i].ID
	}
	photos, err := s.repository.PhotosForPoints(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range points {
		points[i].Photos = photos[points[i].ID]
		for j := range points[i].Photos {
			points[i].Photos[j].URL, _ = s.storage.PresignedDownloadURL(ctx, points[i].Photos[j].S3Key)
		}
	}
	return points, nil
}
