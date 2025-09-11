package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	ws "github.com/mark3748/helpdesk-go/cmd/api/ws"
)

// RoleUser is the minimal interface required for role-based filtering.
type RoleUser interface {
	GetRoles() []string
}

// Events upgrades the connection to WebSocket and registers the client with the hub.
func Events(h *ws.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		uVal, ok := c.Get("user")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}
		user, ok := uVal.(RoleUser)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return
		}
		conn, err := ws.Upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		ctx, cancel := context.WithCancel(context.Background())

		client := ws.NewClient(h, conn, hasRole(user.GetRoles(), "admin"))
		h.Register(client)
		go client.WritePump(ctx)
		client.ReadPump()
		cancel()
	}
}

func hasRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}
