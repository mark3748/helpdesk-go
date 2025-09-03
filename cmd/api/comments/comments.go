package comments

import (
	"net/http"

	"github.com/gin-gonic/gin"
	app "github.com/mark3748/helpdesk-go/cmd/api/app"
)

func List(a *app.App) gin.HandlerFunc { return func(c *gin.Context) { c.JSON(http.StatusOK, []any{}) } }
func Add(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"id": ""}) }
}
