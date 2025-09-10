package roles

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
)

// List returns all role names defined in the roles table.
func List(a *apppkg.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := a.DB.Query(c.Request.Context(), `select name from roles order by name`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out = append(out, name)
		}
		c.JSON(http.StatusOK, out)
	}
}
