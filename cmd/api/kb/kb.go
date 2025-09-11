package kb

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	kbsvc "github.com/mark3748/helpdesk-go/internal/kb"
)

// Search returns knowledge-base articles matching the query parameter `q`.
func Search(a *apppkg.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := c.Query("q")
		arts, err := kbsvc.Search(c.Request.Context(), a.DB, q)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, arts)
	}
}
