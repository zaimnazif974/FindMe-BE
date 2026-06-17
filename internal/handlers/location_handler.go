package handlers

import (
	"errors"
	"net/http"

	"findme/backend/internal/apperror"
	locationdto "findme/backend/internal/dto/locations"
	"findme/backend/internal/middlewares"
	"findme/backend/internal/services"
	"findme/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LocationHandler struct {
	service *services.LocationService
}

func NewLocationHandler(service *services.LocationService) *LocationHandler {
	return &LocationHandler{service: service}
}

func (h *LocationHandler) Share(c *gin.Context) {
	var request locationdto.ShareRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid location data", err.Error())
		return
	}
	shares, err := h.service.Share(c.Request.Context(), middlewares.UserID(c), request)
	h.respond(c, http.StatusCreated, shares, err)
}

func (h *LocationHandler) List(c *gin.Context) {
	h.list(c, false)
}

func (h *LocationHandler) Latest(c *gin.Context) {
	h.list(c, true)
}

func (h *LocationHandler) list(c *gin.Context, latest bool) {
	groupID, err := uuid.Parse(c.Param("groupId"))
	if err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "INVALID_ID", "Invalid group ID")
		return
	}
	shares, serviceErr := h.service.List(c.Request.Context(), groupID, middlewares.UserID(c), latest)
	h.respond(c, http.StatusOK, shares, serviceErr)
}

func (h *LocationHandler) AddPhotos(c *gin.Context) {
	shareID, err := uuid.Parse(c.Param("locationShareId"))
	if err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "INVALID_ID", "Invalid location share ID")
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", "Expected multipart/form-data")
		return
	}
	photos, serviceErr := h.service.AddPhotos(c.Request.Context(), shareID, middlewares.UserID(c), form.File["photos"])
	h.respond(c, http.StatusCreated, photos, serviceErr)
}

func (h *LocationHandler) respond(c *gin.Context, status int, data any, err error) {
	switch {
	case err == nil:
		utils.OK(c, status, data)
	case errors.Is(err, apperror.ErrBadRequest):
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, apperror.ErrForbidden):
		utils.ResponseFailed(c, http.StatusForbidden, "FORBIDDEN", "Group membership is required")
	case errors.Is(err, apperror.ErrNotFound):
		utils.ResponseFailed(c, http.StatusNotFound, "NOT_FOUND", "Location share not found")
	default:
		utils.ResponseFailed(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to process location request")
	}
}
