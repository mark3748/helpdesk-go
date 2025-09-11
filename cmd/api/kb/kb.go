package kb

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	kbsvc "github.com/mark3748/helpdesk-go/internal/kb"
)

// Search returns knowledge-base articles matching the query parameter `q`.
func Search(a *apppkg.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := c.Query("q")
		arts, err := kbsvc.Search(c.Request.Context(), a.DB, q)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, arts)
	}
}

// Get returns a single article by slug.
func Get(a *apppkg.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		art, err := kbsvc.Get(c.Request.Context(), a.DB, c.Param("slug"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusOK, art)
	}
}

// Create inserts a new knowledge base article.
func Create(a *apppkg.App) gin.HandlerFunc {
	type in struct {
		Slug   string `json:"slug"`
		Title  string `json:"title"`
		BodyMD string `json:"body_md"`
	}
	return func(c *gin.Context) {
		var req in
		if err := c.ShouldBindJSON(&req); err != nil || req.Slug == "" || req.Title == "" || req.BodyMD == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		art, err := kbsvc.Create(c.Request.Context(), a.DB, kbsvc.Article{Slug: req.Slug, Title: req.Title, BodyMD: req.BodyMD})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, art)
	}
}

// Update modifies an existing article identified by slug.
func Update(a *apppkg.App) gin.HandlerFunc {
	type in struct {
		Slug   string `json:"slug"`
		Title  string `json:"title"`
		BodyMD string `json:"body_md"`
	}
	return func(c *gin.Context) {
		var req in
		if err := c.ShouldBindJSON(&req); err != nil || req.Slug == "" || req.Title == "" || req.BodyMD == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		art, err := kbsvc.Update(c.Request.Context(), a.DB, c.Param("slug"), kbsvc.Article{Slug: req.Slug, Title: req.Title, BodyMD: req.BodyMD})
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusOK, art)
	}
}

// Delete removes an article by slug.
func Delete(a *apppkg.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := kbsvc.Delete(c.Request.Context(), a.DB, c.Param("slug")); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
