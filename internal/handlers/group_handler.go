package handlers

import (
	"errors"
	"net/http"

	"findme/backend/internal/apperror"
	groupdto "findme/backend/internal/dto/groups"
	"findme/backend/internal/middlewares"
	"findme/backend/internal/services"
	"findme/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GroupHandler struct {
	service *services.GroupService
}

func NewGroupHandler(service *services.GroupService) *GroupHandler {
	return &GroupHandler{service: service}
}

func (h *GroupHandler) Create(c *gin.Context) {
	var request groupdto.CreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid group data", err.Error())
		return
	}
	group, err := h.service.Create(c.Request.Context(), middlewares.UserID(c), request)
	h.respond(c, http.StatusCreated, group, err)
}

func (h *GroupHandler) Join(c *gin.Context) {
	var request groupdto.JoinRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invite code is required")
		return
	}
	group, err := h.service.Join(c.Request.Context(), middlewares.UserID(c), request.InviteCode)
	h.respond(c, http.StatusOK, group, err)
}

func (h *GroupHandler) List(c *gin.Context) {
	groups, err := h.service.List(c.Request.Context(), middlewares.UserID(c))
	h.respond(c, http.StatusOK, groups, err)
}

func (h *GroupHandler) Get(c *gin.Context) {
	groupID, ok := parseID(c, "groupId")
	if !ok {
		return
	}
	group, err := h.service.Get(c.Request.Context(), groupID, middlewares.UserID(c))
	h.respond(c, http.StatusOK, group, err)
}

func (h *GroupHandler) Members(c *gin.Context) {
	groupID, ok := parseID(c, "groupId")
	if !ok {
		return
	}
	members, err := h.service.Members(c.Request.Context(), groupID, middlewares.UserID(c))
	h.respond(c, http.StatusOK, members, err)
}

func (h *GroupHandler) Update(c *gin.Context) {
	groupID, ok := parseID(c, "groupId")
	if !ok {
		return
	}
	var request groupdto.UpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid group data", err.Error())
		return
	}
	group, err := h.service.Update(c.Request.Context(), groupID, middlewares.UserID(c), request)
	h.respond(c, http.StatusOK, group, err)
}

func (h *GroupHandler) Delete(c *gin.Context) {
	groupID, ok := parseID(c, "groupId")
	if !ok {
		return
	}
	err := h.service.Delete(c.Request.Context(), groupID, middlewares.UserID(c))
	h.respond(c, http.StatusOK, gin.H{"message": "Group deleted"}, err)
}

func (h *GroupHandler) Leave(c *gin.Context) {
	groupID, ok := parseID(c, "groupId")
	if !ok {
		return
	}
	err := h.service.Leave(c.Request.Context(), groupID, middlewares.UserID(c))
	h.respond(c, http.StatusOK, gin.H{"message": "Left group"}, err)
}

func (h *GroupHandler) RemoveMember(c *gin.Context) {
	groupID, ok := parseID(c, "groupId")
	if !ok {
		return
	}
	memberID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}
	err = h.service.RemoveMember(c.Request.Context(), groupID, middlewares.UserID(c), memberID)
	h.respond(c, http.StatusOK, gin.H{"message": "Member removed"}, err)
}

func (h *GroupHandler) RegenerateInviteCode(c *gin.Context) {
	groupID, ok := parseID(c, "groupId")
	if !ok {
		return
	}
	code, err := h.service.RegenerateInviteCode(c.Request.Context(), groupID, middlewares.UserID(c))
	h.respond(c, http.StatusOK, gin.H{"invite_code": code}, err)
}

func (h *GroupHandler) respond(c *gin.Context, status int, data any, err error) {
	switch {
	case err == nil:
		utils.OK(c, status, data)
	case errors.Is(err, apperror.ErrBadRequest):
		utils.ResponseFailed(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, apperror.ErrForbidden):
		utils.ResponseFailed(c, http.StatusForbidden, "FORBIDDEN", "You do not have permission for this group")
	case errors.Is(err, apperror.ErrNotFound):
		utils.ResponseFailed(c, http.StatusNotFound, "NOT_FOUND", "Group or member not found")
	case errors.Is(err, apperror.ErrGroupLimit):
		utils.ResponseFailed(c, http.StatusConflict, "GROUP_LIMIT", "A user can join at most 5 groups")
	case errors.Is(err, apperror.ErrMemberLimit):
		utils.ResponseFailed(c, http.StatusConflict, "MEMBER_LIMIT", "A group can contain at most 10 members")
	case errors.Is(err, apperror.ErrConflict):
		utils.ResponseFailed(c, http.StatusConflict, "CONFLICT", err.Error())
	default:
		utils.ResponseFailed(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to process group request")
	}
}
