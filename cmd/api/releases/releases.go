package releases

import (
	"net/http"

	"github.com/gin-gonic/gin"
	app "github.com/mark3748/helpdesk-go/cmd/api/app"
)

// List returns all releases.
func List(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"releases": []any{}})
	}
}

// Create adds a new release.
func Create(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"id": "placeholder"})
	}
}

// Get retrieves a release by id.
func Get(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
	}
}

// Update modifies a release.
func Update(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
	}
}

// Delete removes a release.
func Delete(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	}
}
