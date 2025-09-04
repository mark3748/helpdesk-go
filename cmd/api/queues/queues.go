package queues

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
)

type Queue struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// List returns all queues sorted by name. Requires agent or manager role.
func List(a *apppkg.App) gin.HandlerFunc {
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
		rows, err := a.DB.Query(c.Request.Context(), `select id::text, name from queues order by name`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []Queue{}
		for rows.Next() {
			var q Queue
			if err := rows.Scan(&q.ID, &q.Name); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out = append(out, q)
		}
		c.JSON(http.StatusOK, out)
	}
}
