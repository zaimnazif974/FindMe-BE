package handlers

import (
	"net/http"

	"findme/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func parseID(c *gin.Context, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "INVALID_ID", "Invalid resource ID")
		return uuid.Nil, false
	}
	return id, true
}

func groupID(c *gin.Context) (uuid.UUID, bool) {
	id, ok := parseID(c, "groupId")
	if !ok {
		return uuid.Nil, false
	}
	return id, true
}
