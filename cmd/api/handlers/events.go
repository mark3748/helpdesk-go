package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RoleUser is the minimal interface required for role-based filtering.
type RoleUser interface {
	GetRoles() []string
}

// Event represents a message broadcast to subscribers.
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// PublishEvent sends an event to the Redis "events" channel.
func PublishEvent(ctx context.Context, rdb *redis.Client, ev Event) {
	if rdb == nil {
		return
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return
	}
	_ = rdb.Publish(ctx, "events", b).Err()
}

// Events streams server-sent events to the client.
func Events(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "events not available"})
			return
		}
		uVal, ok := c.Get("user")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}
		user, ok := uVal.(RoleUser)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return
		}

		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}

		ctx := c.Request.Context()
		sub := rdb.Subscribe(ctx, "events")
		defer sub.Close()
		ch := sub.Channel()

		roles := user.GetRoles()
		isAdmin := hasRole(roles, "admin")

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var ev Event
				if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
					continue
				}
				if ev.Type == "queue_changed" && !isAdmin {
					continue
				}
				fmt.Fprintf(c.Writer, "event: %s\n", ev.Type)
				fmt.Fprintf(c.Writer, "data: %s\n\n", msg.Payload)
				flusher.Flush()
			}
		}
	}
}

func hasRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}
