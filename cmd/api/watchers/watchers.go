package watchers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	app "github.com/mark3748/helpdesk-go/cmd/api/app"
	auth "github.com/mark3748/helpdesk-go/cmd/api/auth"
	"github.com/mark3748/helpdesk-go/cmd/api/events"
)

type watcherReq struct {
	UserID string `json:"user_id" binding:"required"`
}

// List returns watcher IDs for the specified ticket.
func List(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusOK, []string{})
			return
		}
		ticketID := c.Param("id")
		ctx := c.Request.Context()
		rows, err := a.DB.Query(ctx, `select user_id from ticket_watchers where ticket_id=$1`, ticketID)
		if err != nil {
			app.AbortError(c, http.StatusInternalServerError, "db_query_failed", "database query failed", nil)
			return
		}
		defer rows.Close()
		out := []string{}
		for rows.Next() {
			var uid string
			if err := rows.Scan(&uid); err == nil {
				out = append(out, uid)
			}
		}
		c.JSON(http.StatusOK, out)
	}
}

// Add inserts a watcher for the ticket.
func Add(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in watcherReq
		if err := c.ShouldBindJSON(&in); err != nil {
			app.AbortError(c, http.StatusBadRequest, "invalid_body", "invalid request body", map[string]string{"user_id": "required"})
			return
		}
		if a.DB != nil {
			ctx := c.Request.Context()
			ticketID := c.Param("id")
			if _, err := a.DB.Exec(ctx, `insert into ticket_watchers (ticket_id, user_id) values ($1,$2) on conflict do nothing`, ticketID, in.UserID); err != nil {
				app.AbortError(c, http.StatusInternalServerError, "db_exec_failed", "database exec failed", nil)
				return
			}
			if v, ok := c.Get("user"); ok {
				if u, ok := v.(auth.AuthUser); ok {
					events.Emit(ctx, a.DB, ticketID, "watcher_add", gin.H{"user_id": in.UserID, "actor_id": u.ID})
				}
			}
		}
		c.Status(http.StatusCreated)
	}
}

// Remove deletes a watcher from the ticket.
func Remove(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB != nil {
			ctx := c.Request.Context()
			ticketID := c.Param("id")
			watcherID := c.Param("uid")
			if _, err := a.DB.Exec(ctx, `delete from ticket_watchers where ticket_id=$1 and user_id=$2`, ticketID, watcherID); err != nil {
				app.AbortError(c, http.StatusInternalServerError, "db_exec_failed", "database exec failed", nil)
				return
			}
			if v, ok := c.Get("user"); ok {
				if u, ok := v.(auth.AuthUser); ok {
					events.Emit(ctx, a.DB, ticketID, "watcher_remove", gin.H{"user_id": watcherID, "actor_id": u.ID})
				}
			}
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
