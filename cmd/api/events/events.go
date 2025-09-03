package events

import (
    "time"

    "github.com/gin-gonic/gin"

    apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
)

// Stream provides a minimal SSE endpoint that maintains a connection
// and sends periodic heartbeats. Specific domain events can be wired in later.
func Stream(a *apppkg.App) gin.HandlerFunc {
    _ = a
    return func(c *gin.Context) {
        // Standard SSE headers
        c.Writer.Header().Set("Content-Type", "text/event-stream")
        c.Writer.Header().Set("Cache-Control", "no-cache")
        c.Writer.Header().Set("Connection", "keep-alive")
        c.Writer.Header().Set("X-Content-Type-Options", "nosniff")

        // Flush initial comment to open the stream
        c.Writer.WriteHeader(200)
        if flusher, ok := c.Writer.(gin.ResponseWriter); ok {
            flusher.Flush()
        } else if f, ok := c.Writer.(interface{ Flush() }); ok {
            f.Flush()
        }

        ticker := time.NewTicker(25 * time.Second)
        defer ticker.Stop()

        // Heartbeat loop
        for {
            select {
            case <-c.Request.Context().Done():
                return
            case t := <-ticker.C:
                // send a lightweight ping to keep the connection alive
                c.SSEvent("ping", gin.H{"t": t.Unix()})
                if f, ok := c.Writer.(interface{ Flush() }); ok {
                    f.Flush()
                }
            }
        }
    }
}

