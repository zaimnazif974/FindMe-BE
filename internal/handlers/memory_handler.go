package handlers

import (
	"errors"
	"net/http"

	"findme/backend/internal/apperror"
	memorydto "findme/backend/internal/dto/memories"
	"findme/backend/internal/middlewares"
	"findme/backend/internal/services"
	"findme/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type MemoryHandler struct {
	service *services.MemoryService
}

func NewMemoryHandler(service *services.MemoryService) *MemoryHandler {
	return &MemoryHandler{service: service}
}

func (h *MemoryHandler) Create(c *gin.Context) {
	groupID, ok := parseID(c, "groupId")
	if !ok {
		return
	}
	var request memorydto.CreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid memory point data", err.Error())
		return
	}
	point, err := h.service.Create(c.Request.Context(), groupID, middlewares.UserID(c), request)
	h.respond(c, http.StatusCreated, point, err)
}

func (h *MemoryHandler) List(c *gin.Context) {
	groupID, ok := parseID(c, "groupId")
	if !ok {
		return
	}
	points, err := h.service.List(c.Request.Context(), groupID, middlewares.UserID(c))
	h.respond(c, http.StatusOK, points, err)
}

func (h *MemoryHandler) Get(c *gin.Context) {
	pointID, ok := parseID(c, "memoryPointId")
	if !ok {
		return
	}
	point, err := h.service.Get(c.Request.Context(), pointID, middlewares.UserID(c))
	h.respond(c, http.StatusOK, point, err)
}

func (h *MemoryHandler) Update(c *gin.Context) {
	pointID, ok := parseID(c, "memoryPointId")
	if !ok {
		return
	}
	var request memorydto.UpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid memory point data", err.Error())
		return
	}
	point, err := h.service.Update(c.Request.Context(), pointID, middlewares.UserID(c), request)
	h.respond(c, http.StatusOK, point, err)
}

func (h *MemoryHandler) Delete(c *gin.Context) {
	pointID, ok := parseID(c, "memoryPointId")
	if !ok {
		return
	}
	err := h.service.Delete(c.Request.Context(), pointID, middlewares.UserID(c))
	h.respond(c, http.StatusOK, gin.H{"message": "Memory point deleted"}, err)
}

func (h *MemoryHandler) Rate(c *gin.Context) {
	pointID, ok := parseID(c, "memoryPointId")
	if !ok {
		return
	}
	var request memorydto.RatingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", "Rating must be between 1 and 5")
		return
	}
	rating, err := h.service.Rate(c.Request.Context(), pointID, middlewares.UserID(c), request.RatingValue)
	h.respond(c, http.StatusOK, rating, err)
}

func (h *MemoryHandler) AddComment(c *gin.Context) {
	pointID, ok := parseID(c, "memoryPointId")
	if !ok {
		return
	}
	var request memorydto.CommentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", "Comment text is required")
		return
	}
	comment, err := h.service.AddComment(c.Request.Context(), pointID, middlewares.UserID(c), request.CommentText)
	h.respond(c, http.StatusCreated, comment, err)
}

func (h *MemoryHandler) Comments(c *gin.Context) {
	pointID, ok := parseID(c, "memoryPointId")
	if !ok {
		return
	}
	comments, err := h.service.Comments(c.Request.Context(), pointID, middlewares.UserID(c))
	h.respond(c, http.StatusOK, comments, err)
}

func (h *MemoryHandler) AddPhotos(c *gin.Context) {
	pointID, ok := parseID(c, "memoryPointId")
	if !ok {
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", "Expected multipart/form-data")
		return
	}
	photos, serviceErr := h.service.AddPhotos(c.Request.Context(), pointID, middlewares.UserID(c), form.File["photos"])
	h.respond(c, http.StatusCreated, photos, serviceErr)
}

func (h *MemoryHandler) respond(c *gin.Context, status int, data any, err error) {
	switch {
	case err == nil:
		utils.OK(c, status, data)
	case errors.Is(err, apperror.ErrBadRequest):
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, apperror.ErrForbidden):
		utils.ResponseFailed(c, http.StatusForbidden, "FORBIDDEN", "You do not have permission for this memory point")
	case errors.Is(err, apperror.ErrNotFound):
		utils.ResponseFailed(c, http.StatusNotFound, "NOT_FOUND", "Memory point not found")
	default:
		utils.ResponseFailed(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to process memory point request")
	}
}
