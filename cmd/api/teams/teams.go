package teams

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	teamsvc "github.com/mark3748/helpdesk-go/internal/teams"
)

// List returns all teams.
func List(a *apppkg.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		teams, err := teamsvc.List(c.Request.Context(), a.DB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, teams)
	}
}
