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

// ListAssets handles GET /assets
func ListAssets(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		// Parse query parameters
		var filters AssetSearchFilters
		if err := c.ShouldBindQuery(&filters); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		result, err := service.ListAssets(c.Request.Context(), filters)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

// CreateAsset handles POST /assets
func CreateAsset(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		u, _ := c.Get("user")
		authUser := u.(auth.AuthUser)
		if u == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		var req CreateAssetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		asset, err := service.CreateAsset(c.Request.Context(), req, uuid.MustParse(authUser.ID))
		if err != nil {
			if err.Error() == "asset tag already exists" {
				c.JSON(http.StatusConflict, gin.H{"error": "asset tag already exists"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, asset)
	}
}

// GetAsset handles GET /assets/:id
func GetAsset(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset ID"})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		asset, err := service.GetAsset(c.Request.Context(), id)
		if err != nil {
			if err.Error() == "asset not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, asset)
	}
}

// UpdateAsset handles PATCH /assets/:id
func UpdateAsset(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		u, _ := c.Get("user")
		authUser := u.(auth.AuthUser)
		if u == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset ID"})
			return
		}

		var req UpdateAssetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		asset, err := service.UpdateAsset(c.Request.Context(), id, req, uuid.MustParse(authUser.ID))
		if err != nil {
			if err.Error() == "asset not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, asset)
	}
}

// DeleteAsset handles DELETE /assets/:id
func DeleteAsset(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		u, _ := c.Get("user")
		authUser := u.(auth.AuthUser)
		if u == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset ID"})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		err = service.DeleteAsset(c.Request.Context(), id, uuid.MustParse(authUser.ID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "asset deleted successfully"})
	}
}

// AssignAsset handles POST /assets/:id/assign
func AssignAsset(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		u, _ := c.Get("user")
		authUser := u.(auth.AuthUser)
		if u == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset ID"})
			return
		}

		var req AssignAssetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		err = service.AssignAsset(c.Request.Context(), id, req, uuid.MustParse(authUser.ID))
		if err != nil {
			if err.Error() == "asset not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "asset assigned successfully"})
	}
}

// ListCategories handles GET /asset-categories
func ListCategories(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		categories, err := service.ListCategories(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, categories)
	}
}

// CreateCategory handles POST /asset-categories
func CreateCategory(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		var req CreateCategoryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		category, err := service.CreateCategory(c.Request.Context(), req)
		if err != nil {
			if err.Error() == "category already exists" {
				c.JSON(http.StatusConflict, gin.H{"error": "category name already exists"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, category)
	}
}

// GetCategory handles GET /asset-categories/:id
func GetCategory(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category ID"})
			return
		}

		service := NewService(a.DB.(*pgxpool.Pool))
		category, err := service.GetCategory(c.Request.Context(), id)
		if err != nil {
			if err.Error() == "category not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, category)
	}
}

// GetAssetHistory handles GET /assets/:id/history
func GetAssetHistory(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset ID"})
			return
		}

		// Parse pagination parameters
		page := 1
		limit := 20
		if pageStr := c.Query("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}
		if limitStr := c.Query("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
				limit = l
			}
		}

		offset := (page - 1) * limit

		query := `
			SELECT 
				h.id, h.asset_id, h.action, h.actor_id, h.old_values, h.new_values, h.notes, h.created_at,
				u.id, u.email, u.display_name
			FROM asset_history h
			LEFT JOIN users u ON h.actor_id = u.id
			WHERE h.asset_id = $1
			ORDER BY h.created_at DESC
			LIMIT $2 OFFSET $3`

		rows, err := a.DB.Query(c.Request.Context(), query, id, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var history []AssetHistory
		for rows.Next() {
			var h AssetHistory
			var actor AssetUser
			var actorID *uuid.UUID
			var actorEmail, actorDisplayName *string

			err := rows.Scan(&h.ID, &h.AssetID, &h.Action, &actorID,
				&h.OldValues, &h.NewValues, &h.Notes, &h.CreatedAt,
				&actorID, &actorEmail, &actorDisplayName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			if actorID != nil {
				actor.ID = *actorID
				if actorEmail != nil {
					actor.Email = *actorEmail
				}
				if actorDisplayName != nil {
					actor.DisplayName = actorDisplayName
				}
				h.Actor = &actor
			}

			history = append(history, h)
		}

		// Get total count
		var total int
		err = a.DB.QueryRow(c.Request.Context(),
			"SELECT COUNT(*) FROM asset_history WHERE asset_id = $1", id).Scan(&total)
		if err != nil {
			total = len(history)
		}

		c.JSON(http.StatusOK, gin.H{
			"history": history,
			"total":   total,
			"page":    page,
			"limit":   limit,
		})
	}
}

// GetAssetAssignments handles GET /assets/:id/assignments
func GetAssetAssignments(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset ID"})
			return
		}

		query := `
			SELECT 
				aa.id, aa.asset_id, aa.assigned_to_user_id, aa.assigned_by_user_id,
				aa.assigned_at, aa.unassigned_at, aa.notes, aa.status,
				assigned_user.id, assigned_user.email, assigned_user.display_name,
				assigned_by.id, assigned_by.email, assigned_by.display_name
			FROM asset_assignments aa
			LEFT JOIN users assigned_user ON aa.assigned_to_user_id = assigned_user.id
			LEFT JOIN users assigned_by ON aa.assigned_by_user_id = assigned_by.id
			WHERE aa.asset_id = $1
			ORDER BY aa.assigned_at DESC`

		rows, err := a.DB.Query(c.Request.Context(), query, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var assignments []AssetAssignment
		for rows.Next() {
			var assignment AssetAssignment
			var assignedUser, assignedByUser AssetUser
			var assignedUserID, assignedByUserID *uuid.UUID
			var assignedEmail, assignedDisplayName, assignedByEmail, assignedByDisplayName *string

			err := rows.Scan(
				&assignment.ID, &assignment.AssetID, &assignedUserID, &assignedByUserID,
				&assignment.AssignedAt, &assignment.UnassignedAt, &assignment.Notes, &assignment.Status,
				&assignedUserID, &assignedEmail, &assignedDisplayName,
				&assignedByUserID, &assignedByEmail, &assignedByDisplayName,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			if assignedUserID != nil {
				assignedUser.ID = *assignedUserID
				if assignedEmail != nil {
					assignedUser.Email = *assignedEmail
				}
				if assignedDisplayName != nil {
					assignedUser.DisplayName = assignedDisplayName
				}
				assignment.AssignedUser = &assignedUser
			}

			if assignedByUserID != nil {
				assignedByUser.ID = *assignedByUserID
				if assignedByEmail != nil {
					assignedByUser.Email = *assignedByEmail
				}
				if assignedByDisplayName != nil {
					assignedByUser.DisplayName = assignedByDisplayName
				}
				assignment.AssignedBy = &assignedByUser
			}

			assignments = append(assignments, assignment)
		}

		c.JSON(http.StatusOK, assignments)
	}
}
