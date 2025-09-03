package attachments

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
	eventspkg "github.com/mark3748/helpdesk-go/cmd/api/events"
	s3svc "github.com/mark3748/helpdesk-go/internal/s3"
	"github.com/minio/minio-go/v7"
)

func List(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusOK, []any{})
			return
		}
		const q = `select id::text, filename, bytes from attachments where ticket_id=$1 order by created_at asc`
		rows, err := a.DB.Query(c.Request.Context(), q, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		type att struct {
			ID       string `json:"id"`
			Filename string `json:"filename"`
			Bytes    int64  `json:"bytes"`
		}
		var out []att
		for rows.Next() {
			var a1 att
			if err := rows.Scan(&a1.ID, &a1.Filename, &a1.Bytes); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out = append(out, a1)
		}
		c.JSON(http.StatusOK, out)
	}
}

func Upload(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil || a.M == nil {
			c.JSON(http.StatusCreated, gin.H{"id": "temp"})
			return
		}
		f, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
			return
		}
		defer f.Close()
		safeName := sanitizeFilename(header.Filename)
		if safeName == "" {
			safeName = "file"
		}
		key := uuid.New().String() + "-" + safeName
		size := header.Size
		ct := header.Header.Get("Content-Type")
		if ct == "" {
			ct = mime.TypeByExtension(filepath.Ext(header.Filename))
		}
		if _, err := a.M.PutObject(c.Request.Context(), a.Cfg.MinIOBucket, key, f, size, minio.PutObjectOptions{ContentType: ct}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		const q = `insert into attachments (ticket_id, uploader_id, object_key, filename, bytes, mime) values ($1, $2, $3, $4, $5, $6) returning id::text`
		var id string
		// Use current authenticated user's ID as uploader
		var uploader string
		if v, ok := c.Get("user"); ok {
			if u, ok := v.(authpkg.AuthUser); ok {
				uploader = u.ID
			}
		}
		if uploader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}
		if err := a.DB.QueryRow(c.Request.Context(), q, c.Param("id"), uploader, key, header.Filename, size, ct).Scan(&id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		eventspkg.Emit(c.Request.Context(), a.DB, c.Param("id"), "ticket_updated", map[string]any{"id": c.Param("id")})
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

func Get(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusOK, gin.H{"id": c.Param("attID")})
			return
		}
		const q = `select object_key, filename, bytes, mime from attachments where id=$1 and ticket_id=$2`
		var key, fn, mt string
		var size int64
		if err := a.DB.QueryRow(c.Request.Context(), q, c.Param("attID"), c.Param("id")).Scan(&key, &fn, &size, &mt); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		// Serve from filesystem store when configured
		if fs, ok := a.M.(*app.FsObjectStore); ok {
			root := filepath.Join(fs.Base, a.Cfg.MinIOBucket)
			path := filepath.Clean(filepath.Join(root, key))
			// Ensure the path is within the root (prevent traversal)
			if rel, err := filepath.Rel(root, path); err != nil || strings.HasPrefix(rel, "..") {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
				return
			}
			f, err := os.ReadFile(path)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			if mt != "" {
				c.Writer.Header().Set("Content-Type", mt)
			} else {
				c.Writer.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(fn)))
			}
			c.Writer.Header().Set("Content-Disposition", "attachment; filename=\""+strings.ReplaceAll(fn, "\"", "")+"\"")
			_, _ = c.Writer.Write(f)
			return
		}
		// Otherwise unimplemented (e.g., MinIO); client may handle 501
		c.JSON(http.StatusNotImplemented, gin.H{"error": "download not implemented"})
	}
}

func PresignUpload(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.M == nil {
			c.JSON(http.StatusOK, gin.H{"url": "", "object_key": "temp"})
			return
		}
		mc, ok := a.M.(*minio.Client)
		if !ok {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "presign not supported"})
			return
		}
		var req struct {
			Filename    string `json:"filename"`
			ContentType string `json:"content_type"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.Filename == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "filename required"})
			return
		}
		key := uuid.New().String()
		if sn := sanitizeFilename(req.Filename); sn != "" {
			key += "-" + sn
		}
		svc := s3svc.Service{Client: mc, Bucket: a.Cfg.MinIOBucket, MaxTTL: time.Minute}
		u, err := svc.PresignPut(c.Request.Context(), key, req.ContentType, time.Minute)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"url": u, "object_key": key, "content_type": req.ContentType})
	}
}

func PresignDownload(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil || a.M == nil {
			c.JSON(http.StatusOK, gin.H{"url": ""})
			return
		}
		mc, ok := a.M.(*minio.Client)
		if !ok {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "presign not supported"})
			return
		}
		const q = `select object_key, filename, mime from attachments where id=$1 and ticket_id=$2`
		var key, fn, mt string
		if err := a.DB.QueryRow(c.Request.Context(), q, c.Param("attID"), c.Param("id")).Scan(&key, &fn, &mt); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		svc := s3svc.Service{Client: mc, Bucket: a.Cfg.MinIOBucket, MaxTTL: time.Minute}
		u, err := svc.PresignGet(c.Request.Context(), key, sanitizeFilename(fn), time.Minute)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"url": u, "content_type": mt})
	}
}

// sanitizeFilename removes path separators and dot segments and restricts to a
// conservative character set, preserving the extension when possible.
func sanitizeFilename(name string) string {
	// Drop any path components
	name = filepath.Base(name)
	// Replace Windows separators too
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	// Remove dot-dot sequences
	name = strings.ReplaceAll(name, "..", "")
	// Allow only letters, digits, space, dash, underscore, and dot
	b := strings.Builder{}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := strings.TrimSpace(b.String())
	// Avoid empty or hidden names
	out = strings.TrimLeft(out, ".")
	return out
}

func Delete(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.DB == nil {
			c.JSON(http.StatusOK, gin.H{"ok": true})
			return
		}
		const q = `delete from attachments where id=$1 and ticket_id=$2`
		if _, err := a.DB.Exec(c.Request.Context(), q, c.Param("attID"), c.Param("id")); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
