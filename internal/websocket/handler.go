package websocket

import (
	"context"
	"net/http"

	"findme/backend/internal/middlewares"
	"findme/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gorilla "github.com/gorilla/websocket"
)

type Handler struct {
	hub       *Hub
	groups    groupRepository
	jwtSecret string
	origins   map[string]bool
}

type groupRepository interface {
	IsMember(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}

func NewHandler(hub *Hub, groupRepo groupRepository, jwtSecret string, allowedOrigins []string) *Handler {
	origins := make(map[string]bool, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origins[origin] = true
	}
	return &Handler{hub: hub, groups: groupRepo, jwtSecret: jwtSecret, origins: origins}
}

func (h *Handler) Connect(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("groupId"))
	if err != nil {
		utils.ResponseFailed(c, http.StatusBadRequest, "INVALID_ID", "Invalid group ID")
		return
	}
	userID, err := middlewares.ValidateToken(c.Query("token"), h.jwtSecret)
	if err != nil {
		utils.ResponseFailed(c, http.StatusUnauthorized, "INVALID_TOKEN", "A valid token query parameter is required")
		return
	}
	member, err := h.groups.IsMember(c.Request.Context(), groupID, userID)
	if err != nil || !member {
		utils.ResponseFailed(c, http.StatusForbidden, "FORBIDDEN", "Group membership is required")
		return
	}
	upgrader := gorilla.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return origin == "" || h.origins[origin]
		},
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := &Client{hub: h.hub, conn: conn, groupID: groupID.String(), send: make(chan []byte, 32)}
	h.hub.register <- client
	go client.writePump()
	go client.readPump()
}
