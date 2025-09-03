package attachments

import (
	"net/http"

	"github.com/gin-gonic/gin"
	app "github.com/mark3748/helpdesk-go/cmd/api/app"
)

func List(a *app.App) gin.HandlerFunc { return func(c *gin.Context) { c.JSON(http.StatusOK, []any{}) } }
func Upload(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"id": ""}) }
}
func Get(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"id": ""}) }
}
func Delete(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) }
}
