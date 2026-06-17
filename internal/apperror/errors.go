package apperror

import "errors"

var (
	ErrBadRequest       = errors.New("bad request")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict")
	ErrGroupLimit       = errors.New("user group limit reached")
	ErrMemberLimit      = errors.New("group member limit reached")
	ErrActiveLiveExists = errors.New("active live location session already exists")
)
