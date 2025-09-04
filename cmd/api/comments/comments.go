package comments

import (
	"net/http"

	"github.com/gin-gonic/gin"
	app "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
	eventspkg "github.com/mark3748/helpdesk-go/cmd/api/events"
)

func List(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusOK, []any{})
			return
		}
		const q = `select id::text, body_md from ticket_comments where ticket_id=$1 order by created_at asc`
		rows, err := a.DB.Query(c.Request.Context(), q, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		type resp struct {
			ID     string `json:"id"`
			BodyMD string `json:"body_md"`
		}
		var out []resp
		for rows.Next() {
			var r resp
			if err := rows.Scan(&r.ID, &r.BodyMD); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}

func Add(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusCreated, gin.H{"id": "temp"})
			return
		}
		var in struct {
			BodyMD string `json:"body_md"`
		}
		if err := c.ShouldBindJSON(&in); err != nil || in.BodyMD == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		uVal, _ := c.Get("user")
		au, _ := uVal.(authpkg.AuthUser)
		const q = `insert into ticket_comments (ticket_id, author_id, body_md) values ($1, $2, $3) returning id::text`
		var id string
		if err := a.DB.QueryRow(c.Request.Context(), q, c.Param("id"), au.ID, in.BodyMD).Scan(&id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		eventspkg.Emit(c.Request.Context(), a.DB, c.Param("id"), "ticket_updated", map[string]any{"id": c.Param("id")})
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}
