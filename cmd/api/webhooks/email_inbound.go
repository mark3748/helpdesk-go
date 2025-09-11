package webhooks

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
)

type EmailInboundReq struct {
	RawStoreKey string         `json:"raw_store_key" binding:"required"`
	ParsedJSON  map[string]any `json:"parsed_json" binding:"required"`
	MessageID   string         `json:"message_id"`
}

// EmailInbound accepts inbound email webhooks and stores payloads for later processing.
func EmailInbound(a *apppkg.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in EmailInboundReq
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if a.DB != nil {
			if _, err := a.DB.Exec(c.Request.Context(), `insert into email_inbound (raw_store_key, parsed_json, message_id) values ($1,$2,$3)`, in.RawStoreKey, in.ParsedJSON, in.MessageID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		c.Status(http.StatusAccepted)
	}
}
