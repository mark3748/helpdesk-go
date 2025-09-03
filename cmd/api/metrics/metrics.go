package metrics

import (
	"net/http"

	"github.com/gin-gonic/gin"
	app "github.com/mark3748/helpdesk-go/cmd/api/app"
)

func SLA(a *app.App) gin.HandlerFunc { return func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) } }
func Resolution(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) }
}
func TicketVolume(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) }
}
