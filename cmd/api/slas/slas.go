package slas

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	slapkg "github.com/mark3748/helpdesk-go/internal/sla"
)

// List returns SLA policies.
func List(a *apppkg.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		slas, err := slapkg.ListPolicies(c.Request.Context(), a.DB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, slas)
	}
}
