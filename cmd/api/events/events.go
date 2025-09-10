package events

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
)

// Envelope is the standardized event payload sent to clients.
type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// Stream broadcasts ticket events from the database using Server-Sent Events.
// It supports resuming from the Last-Event-ID header and emits periodic
// heartbeat comments to keep connections alive.
func Stream(a *apppkg.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.Status(http.StatusOK)
			return
		}
		// Standard SSE headers
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}

		ctx := c.Request.Context()

		// Determine starting point based on Last-Event-ID
		// Use a stable resume cursor (created_at, id) to avoid dropping
		// events that share the same timestamp as the last delivered event.
		last := time.Time{}
		lastID := ""
		if id := c.GetHeader("Last-Event-ID"); id != "" {
			_ = a.DB.QueryRow(ctx, `select created_at from ticket_events where id=$1`, id).Scan(&last)
			lastID = id
		}

		// Helper to send all events newer than the provided cursor.
		send := func(since time.Time, sinceID string) (time.Time, string) {
			rows, err := a.DB.Query(ctx, `
                select id::text, event_type, payload, created_at
                from ticket_events
                where created_at > $1 or (created_at = $1 and id <> $2)
                order by created_at asc, id asc`, since, sinceID)
			if err != nil {
				return since, sinceID
			}
			defer rows.Close()
			for rows.Next() {
				var id, typ string
				var payload []byte
				var ts time.Time
				if err := rows.Scan(&id, &typ, &payload, &ts); err != nil {
					continue
				}
				env := Envelope{Type: typ, Data: payload}
				b, _ := json.Marshal(env)
				fmt.Fprintf(c.Writer, "id: %s\n", id)
				fmt.Fprintf(c.Writer, "event: %s\n", typ)
				fmt.Fprintf(c.Writer, "data: %s\n\n", b)
				flusher.Flush()
				since = ts
				sinceID = id
			}
			return since, sinceID
		}

		last, lastID = send(last, lastID)

		poll := time.NewTicker(time.Second)
		heart := time.NewTicker(25 * time.Second)
		defer poll.Stop()
		defer heart.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-poll.C:
				last, lastID = send(last, lastID)
			case <-heart.C:
				fmt.Fprint(c.Writer, ": heartbeat\n\n")
				flusher.Flush()
			}
		}
	}
}
