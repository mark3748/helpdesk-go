package problems

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/mark3748/helpdesk-go/cmd/api/app"
)

// List returns all problem tickets.
func List(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Mock data
		problems := []Problem{
			{
				ID:          "prb_001",
				Title:       "Server Performance Issues",
				Description: "Database queries are running slowly during peak hours",
				Priority:    "high",
				Status:      "investigating",
				ReportedBy:  "user@example.com",
				AssignedTo:  "admin@example.com",
				CreatedAt:   "2024-01-25T08:30:00Z",
				UpdatedAt:   "2024-01-25T14:15:00Z",
			},
		}
		c.JSON(http.StatusOK, gin.H{"problems": problems})
	}
}

// CreateProblemRequest represents the request body for creating a problem ticket
type CreateProblemRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Status      string `json:"status"`
	ReportedBy  string `json:"reported_by"`
	AssignedTo  string `json:"assigned_to"`
}

// Problem represents a problem ticket
type Problem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Status      string `json:"status"`
	ReportedBy  string `json:"reported_by"`
	AssignedTo  string `json:"assigned_to"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Create adds a new problem ticket.
func Create(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateProblemRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Mock implementation
		problem := Problem{
			ID:          "prb_" + generateID(),
			Title:       req.Title,
			Description: req.Description,
			Priority:    getStatusOrDefault(req.Priority, "medium"),
			Status:      getStatusOrDefault(req.Status, "open"),
			ReportedBy:  req.ReportedBy,
			AssignedTo:  req.AssignedTo,
			CreatedAt:   getCurrentTimestamp(),
			UpdatedAt:   getCurrentTimestamp(),
		}

		c.JSON(http.StatusCreated, problem)
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

// Get retrieves a problem ticket by id.
func Get(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Mock response
		problem := Problem{
			ID:          id,
			Title:       "Server Performance Issues",
			Description: "Database queries are running slowly during peak hours",
			Priority:    "high",
			Status:      "investigating",
			ReportedBy:  "user@example.com",
			AssignedTo:  "admin@example.com",
			CreatedAt:   "2024-01-25T08:30:00Z",
			UpdatedAt:   "2024-01-25T14:15:00Z",
		}

		c.JSON(http.StatusOK, problem)
	}
}

// Update modifies a problem ticket.
func Update(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req CreateProblemRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Mock response
		problem := Problem{
			ID:          id,
			Title:       req.Title,
			Description: req.Description,
			Priority:    getStatusOrDefault(req.Priority, "medium"),
			Status:      getStatusOrDefault(req.Status, "open"),
			ReportedBy:  req.ReportedBy,
			AssignedTo:  req.AssignedTo,
			CreatedAt:   "2024-01-25T08:30:00Z",
			UpdatedAt:   getCurrentTimestamp(),
		}

		c.JSON(http.StatusOK, problem)
	}
}

// Delete removes a problem ticket.
func Delete(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	}
}
