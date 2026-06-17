package repositories

import (
	"context"
	"strings"

	"findme/backend/internal/apperror"
	userdto "findme/backend/internal/dto/users"
	"findme/backend/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, name, email, passwordHash string) (userdto.User, error) {
	var user userdto.User
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO users (name, email, password_hash)
		VALUES (?, lower(?), ?)
		RETURNING id, name, email, avatar_url, created_at, updated_at
	`, strings.TrimSpace(name), strings.TrimSpace(email), passwordHash).Scan(&user).Error
	if err != nil {
		if utils.IsUniqueViolation(err) {
			return userdto.User{}, apperror.ErrConflict
		}
		return userdto.User{}, err
	}
	return user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (userdto.UserWithPassword, error) {
	var user userdto.UserWithPassword
	err := r.db.WithContext(ctx).
		Table("users").
		Where("email = lower(?)", strings.TrimSpace(email)).
		Take(&user).Error
	if err != nil {
		return userdto.UserWithPassword{}, utils.DatabaseError(err)
	}
	return user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, userID uuid.UUID) (userdto.User, error) {
	var user userdto.User
	err := r.db.WithContext(ctx).Table("users").Where("id = ?", userID).Take(&user).Error
	if err != nil {
		return userdto.User{}, utils.DatabaseError(err)
	}
	return user, nil
}

func (r *UserRepository) Update(ctx context.Context, userID uuid.UUID, name string, avatarURL *string) (userdto.User, error) {
	result := r.db.WithContext(ctx).Table("users").Where("id = ?", userID).Updates(map[string]any{
		"name":       strings.TrimSpace(name),
		"avatar_url": avatarURL,
		"updated_at": gorm.Expr("now()"),
	})
	if result.Error != nil {
		return userdto.User{}, result.Error
	}
	if result.RowsAffected == 0 {
		return userdto.User{}, apperror.ErrNotFound
	}
	return r.FindByID(ctx, userID)
}
