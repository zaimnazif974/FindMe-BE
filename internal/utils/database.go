package utils

import (
	"errors"

	"findme/backend/internal/apperror"

	"gorm.io/gorm"
)

func DatabaseError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperror.ErrNotFound
	}
	return err
}

func IsUniqueViolation(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey)
}
