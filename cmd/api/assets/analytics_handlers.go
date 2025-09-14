package assets

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	app "github.com/mark3748/helpdesk-go/cmd/api/app"
)

// GetAssetAnalytics handles GET /assets/analytics
func GetAssetAnalytics(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not available"})
			return
		}

		// Parse query parameters
		filters := AnalyticsFilters{}

		if startDateStr := c.Query("start_date"); startDateStr != "" {
			if startDate, err := time.Parse("2006-01-02", startDateStr); err == nil {
				filters.StartDate = &startDate
			}
		}

		if endDateStr := c.Query("end_date"); endDateStr != "" {
			if endDate, err := time.Parse("2006-01-02", endDateStr); err == nil {
				filters.EndDate = &endDate
			}
		}

		if categoryIDStr := c.Query("category"); categoryIDStr != "" {
			if categoryID, err := uuid.Parse(categoryIDStr); err == nil {
				filters.CategoryID = &categoryID
			}
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		analytics, err := service.GetAssetAnalytics(c.Request.Context(), filters)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": analytics})
	}
}