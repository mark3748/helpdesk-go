package webhooks

import (
	"net/http"

	"github.com/gin-gonic/gin"

	app "github.com/mark3748/helpdesk-go/cmd/api/app"
)

type webhookReq struct {
	TargetURL string `json:"target_url" binding:"required"`
	EventMask int    `json:"event_mask"`
	Secret    string `json:"secret"`
	Active    bool   `json:"active"`
}

// List returns all webhook subscriptions.
func List(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusOK, []map[string]any{})
			return
		}
		ctx := c.Request.Context()
		rows, err := a.DB.Query(ctx, `select id, target_url, event_mask, secret, active from webhooks order by target_url`)
		if err != nil {
			app.AbortError(c, http.StatusInternalServerError, "db_query_failed", "database query failed", nil)
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, url, secret string
			var mask int
			var active bool
			if err := rows.Scan(&id, &url, &mask, &secret, &active); err == nil {
				out = append(out, map[string]any{
					"id":         id,
					"target_url": url,
					"event_mask": mask,
					"secret":     secret,
					"active":     active,
				})
			}
		}
		c.JSON(http.StatusOK, out)
	}
}

// Create inserts a new webhook subscription.
func Create(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in webhookReq
		if err := c.ShouldBindJSON(&in); err != nil {
			app.AbortError(c, http.StatusBadRequest, "invalid_body", "invalid request body", map[string]string{"target_url": "required"})
			return
		}
		if a.DB != nil {
			ctx := c.Request.Context()
			if _, err := a.DB.Exec(ctx, `insert into webhooks (target_url, event_mask, secret, active) values ($1,$2,$3,$4)`, in.TargetURL, in.EventMask, in.Secret, in.Active); err != nil {
				app.AbortError(c, http.StatusInternalServerError, "db_exec_failed", "database exec failed", nil)
				return
			}
		}
		c.Status(http.StatusCreated)
	}
}

// Delete removes a webhook subscription by ID.
func Delete(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB != nil {
			ctx := c.Request.Context()
			id := c.Param("id")
			if _, err := a.DB.Exec(ctx, `delete from webhooks where id=$1`, id); err != nil {
				app.AbortError(c, http.StatusInternalServerError, "db_exec_failed", "database exec failed", nil)
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
