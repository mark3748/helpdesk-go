package handlers

import (
	"github.com/gin-gonic/gin"
	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	"net/http"
)

// Features reports simple capability flags the UI can use to toggle features.
func Features(a *apppkg.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		attachments := a.M != nil
		c.JSON(http.StatusOK, gin.H{
			"attachments": attachments,
		})
	}
}
