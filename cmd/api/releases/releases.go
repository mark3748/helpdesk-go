package releases

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/mark3748/helpdesk-go/cmd/api/app"
)

// List returns all releases.
func List(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Mock data for demonstration
		releases := []Release{
			{
				ID:          "rel_001",
				Title:       "System Update v2.1.0",
				Description: "Major system update with new features",
				Version:     "2.1.0",
				Status:      "planned",
				ScheduledAt: "2024-02-01T10:00:00Z",
				CreatedAt:   "2024-01-15T09:00:00Z",
				UpdatedAt:   "2024-01-15T09:00:00Z",
			},
		}
		c.JSON(http.StatusOK, gin.H{"releases": releases})
	}
}

// CreateReleaseRequest represents the request body for creating a release
type CreateReleaseRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Version     string `json:"version" binding:"required"`
	Status      string `json:"status"`
	ScheduledAt string `json:"scheduled_at"`
}

// Release represents a software release
type Release struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	ScheduledAt string `json:"scheduled_at"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Create adds a new release.
func Create(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateReleaseRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// For now, return a mock response since this is a basic implementation
		// In a full implementation, this would save to database
		release := Release{
			ID:          "rel_" + generateID(),
			Title:       req.Title,
			Description: req.Description,
			Version:     req.Version,
			Status:      getStatusOrDefault(req.Status, "planned"),
			ScheduledAt: req.ScheduledAt,
			CreatedAt:   getCurrentTimestamp(),
			UpdatedAt:   getCurrentTimestamp(),
		}

		c.JSON(http.StatusCreated, release)
	}
}

func generateID() string {
	return uuid.New().String()
}

func getStatusOrDefault(status, defaultStatus string) string {
	if status == "" {
		return defaultStatus
	}
	return status
}

func getCurrentTimestamp() string {
	return "2024-01-01T00:00:00Z"
}

// Get retrieves a release by id.
func Get(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Mock response - in production this would query the database
		release := Release{
			ID:          id,
			Title:       "System Update v2.1.0",
			Description: "Major system update with new features",
			Version:     "2.1.0",
			Status:      "planned",
			ScheduledAt: "2024-02-01T10:00:00Z",
			CreatedAt:   "2024-01-15T09:00:00Z",
			UpdatedAt:   "2024-01-15T09:00:00Z",
		}

		c.JSON(http.StatusOK, release)
	}
}

// Update modifies a release.
func Update(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req CreateReleaseRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Mock response - in production this would update the database
		release := Release{
			ID:          id,
			Title:       req.Title,
			Description: req.Description,
			Version:     req.Version,
			Status:      getStatusOrDefault(req.Status, "planned"),
			ScheduledAt: req.ScheduledAt,
			CreatedAt:   "2024-01-15T09:00:00Z",
			UpdatedAt:   getCurrentTimestamp(),
		}

		c.JSON(http.StatusOK, release)
	}
}

// Delete removes a release.
func Delete(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	}
}
