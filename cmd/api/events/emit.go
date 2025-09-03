package events

import (
	"context"
	"encoding/json"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
)

// Emit records a ticket event in the database. Best effort; errors are ignored.
func Emit(ctx context.Context, db apppkg.DB, ticketID, typ string, data interface{}) {
	if db == nil {
		return
	}
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	const q = `insert into ticket_events (ticket_id, event_type, payload) values ($1, $2, $3)`
	_, _ = db.Exec(ctx, q, ticketID, typ, b)
}
