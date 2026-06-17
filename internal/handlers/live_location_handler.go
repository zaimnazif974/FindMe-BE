package handlers

import (
	"errors"
	"net/http"

	"findme/backend/internal/apperror"
	livedto "findme/backend/internal/dto/live_locations"
	"findme/backend/internal/middlewares"
	"findme/backend/internal/services"
	"findme/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type LiveLocationHandler struct {
	service *services.LiveLocationService
}

func NewLiveLocationHandler(service *services.LiveLocationService) *LiveLocationHandler {
	return &LiveLocationHandler{service: service}
}

func (h *LiveLocationHandler) Start(c *gin.Context) {
	groupID, ok := groupID(c)
	if !ok {
		return
	}
	var request livedto.StartRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid duration")
		return
	}
	session, err := h.service.Start(c.Request.Context(), groupID, middlewares.UserID(c), request.DurationMinutes)
	h.respond(c, http.StatusCreated, session, err)
}

func (h *LiveLocationHandler) Update(c *gin.Context) {
	groupID, ok := groupID(c)
	if !ok {
		return
	}
	var request livedto.UpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid live location data", err.Error())
		return
	}
	position, err := h.service.Update(c.Request.Context(), groupID, middlewares.UserID(c), request)
	h.respond(c, http.StatusOK, position, err)
}

func (h *LiveLocationHandler) Stop(c *gin.Context) {
	groupID, ok := groupID(c)
	if !ok {
		return
	}
	session, err := h.service.Stop(c.Request.Context(), groupID, middlewares.UserID(c))
	h.respond(c, http.StatusOK, session, err)
}

func (h *LiveLocationHandler) Active(c *gin.Context) {
	groupID, ok := groupID(c)
	if !ok {
		return
	}
	positions, err := h.service.Active(c.Request.Context(), groupID, middlewares.UserID(c))
	h.respond(c, http.StatusOK, positions, err)
}

func (h *LiveLocationHandler) respond(c *gin.Context, status int, data any, err error) {
	switch {
	case err == nil:
		utils.OK(c, status, data)
	case errors.Is(err, apperror.ErrBadRequest):
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, apperror.ErrForbidden):
		utils.ResponseFailed(c, http.StatusForbidden, "FORBIDDEN", "Group membership is required")
	case errors.Is(err, apperror.ErrActiveLiveExists):
		utils.ResponseFailed(c, http.StatusConflict, "ACTIVE_SESSION_EXISTS", "Only one live location session can be active at a time")
	case errors.Is(err, apperror.ErrNotFound):
		utils.ResponseFailed(c, http.StatusNotFound, "NO_ACTIVE_SESSION", "No active live location session was found")
	default:
		utils.ResponseFailed(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to process live location request")
	}
}
