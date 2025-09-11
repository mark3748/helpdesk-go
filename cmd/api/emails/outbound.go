package emails

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	app "github.com/mark3748/helpdesk-go/cmd/api/app"
)

type Outbound struct {
	ID       string    `json:"id"`
	To       string    `json:"to"`
	Subject  string    `json:"subject"`
	Status   string    `json:"status"`
	Retries  int       `json:"retries"`
	TicketID *string   `json:"ticket_id,omitempty"`
	Created  time.Time `json:"created_at"`
}

func ListOutbound(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := a.DB.Query(c.Request.Context(), `select id::text, to_addr, coalesce(subject,''), status, retries, ticket_id::text, created_at from email_outbound order by created_at desc limit 100`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []Outbound{}
		for rows.Next() {
			var e Outbound
			var tid *string
			if err := rows.Scan(&e.ID, &e.To, &e.Subject, &e.Status, &e.Retries, &tid, &e.Created); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if tid != nil && *tid != "" {
				e.TicketID = tid
			}
			out = append(out, e)
		}
		c.JSON(http.StatusOK, out)
	}
}
