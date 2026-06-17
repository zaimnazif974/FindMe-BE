package handlers

import (
	"errors"
	"net/http"

	"findme/backend/internal/apperror"
	authdto "findme/backend/internal/dto/auth"
	"findme/backend/internal/middlewares"
	"findme/backend/internal/services"
	"findme/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *services.AuthService
}

func NewAuthHandler(service *services.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var request authdto.RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid registration data", err.Error())
		return
	}
	result, err := h.service.Register(c.Request.Context(), request)
	if errors.Is(err, apperror.ErrConflict) {
		utils.ResponseFailed(c, http.StatusConflict, "EMAIL_EXISTS", "An account with this email already exists")
		return
	}
	if err != nil {
		utils.ResponseFailed(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to create account")
		return
	}
	utils.OK(c, http.StatusCreated, result)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request authdto.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid login data", err.Error())
		return
	}
	result, err := h.service.Login(c.Request.Context(), request)
	if errors.Is(err, apperror.ErrUnauthorized) {
		utils.ResponseFailed(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Email or password is incorrect")
		return
	}
	if err != nil {
		utils.ResponseFailed(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to log in")
		return
	}
	utils.OK(c, http.StatusOK, result)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	utils.OK(c, http.StatusOK, gin.H{"message": "Logged out. Discard the access token on the client."})
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, err := h.service.Profile(c.Request.Context(), middlewares.UserID(c))
	if err != nil {
		utils.ResponseFailed(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to load profile")
		return
	}
	utils.OK(c, http.StatusOK, user)
}

func (h *AuthHandler) UpdateMe(c *gin.Context) {
	var request authdto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid profile data", err.Error())
		return
	}
	user, err := h.service.UpdateProfile(c.Request.Context(), middlewares.UserID(c), request)
	if err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	utils.OK(c, http.StatusOK, user)
}
