package tickets

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
	handlers "github.com/mark3748/helpdesk-go/cmd/api/handlers"
)

// Assign changes the assignee of a ticket and emits a ticket_updated event.
// Requires agent or manager role.
func Assign(a *apppkg.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		uVal, ok := c.Get("user")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}
		user, ok := uVal.(authpkg.AuthUser)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return
		}
		allowed := false
		for _, r := range user.Roles {
			if r == "agent" || r == "manager" || r == "admin" {
				allowed = true
				break
			}
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		var in struct {
			AssigneeID string `json:"assignee_id"`
		}
		if err := c.ShouldBindJSON(&in); err != nil || in.AssigneeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if a.DB == nil {
			c.JSON(http.StatusOK, Ticket{ID: c.Param("id"), AssigneeID: &in.AssigneeID})
			return
		}
		const q = `update tickets set assignee_id=$1, updated_at=now() where id=$2 returning id::text, number, title, status, assignee_id::text, priority`
		var t Ticket
		var assignee *string
		var number any
		row := a.DB.QueryRow(c.Request.Context(), q, in.AssigneeID, c.Param("id"))
		if err := row.Scan(&t.ID, &number, &t.Title, &t.Status, &assignee, &t.Priority); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		t.Number = number
		t.AssigneeID = assignee
		handlers.PublishEvent(c.Request.Context(), a.Q, handlers.Event{Type: "ticket_updated", Data: map[string]any{"id": t.ID}})
		c.JSON(http.StatusOK, t)
	}
}
