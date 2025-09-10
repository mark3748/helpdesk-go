package assets

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mark3748/helpdesk-go/cmd/api/app"
	"github.com/mark3748/helpdesk-go/cmd/api/auth"
)

// RequestStatusChange handles POST /assets/:id/status-change
func RequestStatusChange(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		u, _ := c.Get("user")
		authUser := u.(auth.AuthUser)

		assetID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset ID"})
			return
		}

		var req struct {
			ToStatus AssetStatus `json:"to_status" binding:"required"`
			Comments *string     `json:"comments"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		workflow, err := service.RequestStatusChange(c.Request.Context(), assetID, req.ToStatus, uuid.MustParse(authUser.ID), req.Comments)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, workflow)
	}
}

// ApproveWorkflow handles POST /workflows/:id/approve
func ApproveWorkflow(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		u, _ := c.Get("user")
		authUser := u.(auth.AuthUser)

		workflowID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow ID"})
			return
		}

		var req struct {
			Comments *string `json:"comments"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		err = service.ApproveWorkflow(c.Request.Context(), workflowID, uuid.MustParse(authUser.ID), req.Comments)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "workflow approved"})
	}
}

// RejectWorkflow handles POST /workflows/:id/reject
func RejectWorkflow(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		u, _ := c.Get("user")
		authUser := u.(auth.AuthUser)

		workflowID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow ID"})
			return
		}

		var req struct {
			Comments *string `json:"comments"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		err = service.RejectWorkflow(c.Request.Context(), workflowID, uuid.MustParse(authUser.ID), req.Comments)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "workflow rejected"})
	}
}

// CheckoutAsset handles POST /assets/:id/checkout
func CheckoutAsset(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		u, _ := c.Get("user")
		authUser := u.(auth.AuthUser)

		assetID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset ID"})
			return
		}

		var req CheckoutRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		req.AssetID = assetID

		service := NewService(a.DB.(*pgxpool.Pool))
		checkout, err := service.CheckoutAsset(c.Request.Context(), req, uuid.MustParse(authUser.ID))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, checkout)
	}
}

// CheckinAsset handles POST /assets/checkin
func CheckinAsset(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		u, _ := c.Get("user")
		authUser := u.(auth.AuthUser)

		var req CheckinRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		err := service.CheckinAsset(c.Request.Context(), req, uuid.MustParse(authUser.ID))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "asset checked in successfully"})
	}
}

// GetActiveCheckouts handles GET /assets/checkouts/active
func GetActiveCheckouts(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		checkouts, err := service.GetActiveCheckouts(c.Request.Context(), nil)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, checkouts)
	}
}

// GetOverdueCheckouts handles GET /assets/checkouts/overdue
func GetOverdueCheckouts(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		checkouts, err := service.GetOverdueCheckouts(c.Request.Context())

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, checkouts)
	}
}

// CreateRelationship handles POST /assets/:id/relationships
func CreateRelationship(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		u, _ := c.Get("user")
		authUser := u.(auth.AuthUser)

		parentAssetID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset ID"})
			return
		}

		var req RelationshipRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		req.ParentAssetID = parentAssetID

		service := NewService(a.DB.(*pgxpool.Pool))
		relationship, err := service.CreateRelationship(c.Request.Context(), req, uuid.MustParse(authUser.ID))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, relationship)
	}
}

// GetRelationshipGraph handles GET /assets/:id/relationships/graph
func GetRelationshipGraph(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		assetID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset ID"})
			return
		}

		maxDepth := 3 // Default depth
		if depthStr := c.Query("max_depth"); depthStr != "" {
			if d, err := strconv.Atoi(depthStr); err == nil && d > 0 && d <= 10 {
				maxDepth = d
			}
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		graph, err := service.GetRelationshipGraph(c.Request.Context(), assetID, maxDepth)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, graph)
	}
}

// GetAssetImpactAnalysis handles GET /assets/:id/impact-analysis
func GetAssetImpactAnalysis(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		assetID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset ID"})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		analysis, err := service.GetAssetImpactAnalysis(c.Request.Context(), assetID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, analysis)
	}
}

// BulkUpdateAssets handles POST /assets/bulk/update
func BulkUpdateAssets(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		u, _ := c.Get("user")
		authUser := u.(auth.AuthUser)

		var req BulkUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		operation, err := service.BulkUpdateAssets(c.Request.Context(), req, uuid.MustParse(authUser.ID))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusAccepted, operation)
	}
}

// BulkAssignAssets handles POST /assets/bulk/assign
func BulkAssignAssets(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		u, _ := c.Get("user")
		authUser := u.(auth.AuthUser)

		var req BulkAssignRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		operation, err := service.BulkAssignAssets(c.Request.Context(), req, uuid.MustParse(authUser.ID))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusAccepted, operation)
	}
}

// ImportAssetsPreview handles POST /assets/import/preview
func ImportAssetsPreview(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
			return
		}
		defer file.Close()

		service := NewService(a.DB.(*pgxpool.Pool))
		preview, err := service.PreviewImport(c.Request.Context(), file, header.Filename)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, preview)
	}
}

// ImportAssets handles POST /assets/import
func ImportAssets(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		u, _ := c.Get("user")
		authUser := u.(auth.AuthUser)

		file, _, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
			return
		}
		defer file.Close()

		options := make(map[string]interface{})
		if skipValidation := c.PostForm("skip_validation"); skipValidation == "true" {
			options["skip_validation"] = true
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		operation, err := service.ImportAssets(c.Request.Context(), file, uuid.MustParse(authUser.ID), options)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusAccepted, operation)
	}
}

// ExportAssets handles POST /assets/export
func ExportAssets(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		u, _ := c.Get("user")
		authUser := u.(auth.AuthUser)

		var req ExportRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		operation, err := service.ExportAssets(c.Request.Context(), req, uuid.MustParse(authUser.ID))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusAccepted, operation)
	}
}

// GetBulkOperation handles GET /assets/bulk/operations/:id
func GetBulkOperation(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		operationID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid operation ID"})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		operation, err := service.getBulkOperation(c.Request.Context(), operationID)

		if err != nil {
			if err.Error() == "no rows in result set" { // or use errors.Is(err, pgx.ErrNoRows) if available
				c.JSON(http.StatusNotFound, gin.H{"error": "operation not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, operation)
	}
}

// GetAuditHistory handles GET /assets/:id/audit
func GetAuditHistory(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		assetID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset ID"})
			return
		}

		var filter AuditFilter
		if err := c.ShouldBindQuery(&filter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		filter.AssetID = &assetID

		service := NewService(a.DB.(*pgxpool.Pool))
		result, err := service.GetAuditHistory(c.Request.Context(), filter)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

// GetAuditSummary handles GET /assets/audit/summary
func GetAuditSummary(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		var assetID *uuid.UUID
		if assetIDStr := c.Query("asset_id"); assetIDStr != "" {
			if id, err := uuid.Parse(assetIDStr); err == nil {
				assetID = &id
			}
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		summary, err := service.GetAuditSummary(c.Request.Context(), assetID, nil, nil)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, summary)
	}
}
