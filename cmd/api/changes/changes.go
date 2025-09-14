package changes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	app "github.com/mark3748/helpdesk-go/cmd/api/app"
)

// List returns all change requests.
func List(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Mock data
		changes := []Change{
			{
				ID:          "chg_001",
				Title:       "Database Schema Update",
				Description: "Update user table schema to support new features",
				Priority:    "high",
				Status:      "approved",
				RequestedBy: "admin@example.com",
				ScheduledAt: "2024-02-15T14:00:00Z",
				CreatedAt:   "2024-01-20T10:00:00Z",
				UpdatedAt:   "2024-01-22T15:30:00Z",
			},
		}
		c.JSON(http.StatusOK, gin.H{"changes": changes})
	}
}

// CreateChangeRequest represents the request body for creating a change request
type CreateChangeRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Status      string `json:"status"`
	RequestedBy string `json:"requested_by"`
	ScheduledAt string `json:"scheduled_at"`
}

// Change represents a change request
type Change struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Status      string `json:"status"`
	RequestedBy string `json:"requested_by"`
	ScheduledAt string `json:"scheduled_at"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Create adds a new change request.
func Create(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateChangeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Mock implementation
		change := Change{
			ID:          "chg_" + generateID(),
			Title:       req.Title,
			Description: req.Description,
			Priority:    getStatusOrDefault(req.Priority, "medium"),
			Status:      getStatusOrDefault(req.Status, "draft"),
			RequestedBy: req.RequestedBy,
			ScheduledAt: req.ScheduledAt,
			CreatedAt:   getCurrentTimestamp(),
			UpdatedAt:   getCurrentTimestamp(),
		}

		c.JSON(http.StatusCreated, change)
	}
}

func generateID() string {
	return "123456"
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

// Get retrieves a change request by id.
func Get(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		
		// Mock response
		change := Change{
			ID:          id,
			Title:       "Database Schema Update",
			Description: "Update user table schema to support new features",
			Priority:    "high",
			Status:      "approved",
			RequestedBy: "admin@example.com",
			ScheduledAt: "2024-02-15T14:00:00Z",
			CreatedAt:   "2024-01-20T10:00:00Z",
			UpdatedAt:   "2024-01-22T15:30:00Z",
		}
		
		c.JSON(http.StatusOK, change)
	}
}

// Update modifies a change request.
func Update(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		
		var req CreateChangeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Mock response
		change := Change{
			ID:          id,
			Title:       req.Title,
			Description: req.Description,
			Priority:    getStatusOrDefault(req.Priority, "medium"),
			Status:      getStatusOrDefault(req.Status, "draft"),
			RequestedBy: req.RequestedBy,
			ScheduledAt: req.ScheduledAt,
			CreatedAt:   "2024-01-20T10:00:00Z",
			UpdatedAt:   getCurrentTimestamp(),
		}
		
		c.JSON(http.StatusOK, change)
	}
}

// Delete removes a change request.
func Delete(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	}
}
