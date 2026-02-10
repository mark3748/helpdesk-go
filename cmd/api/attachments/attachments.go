package attachments

import (
	"bytes"
	"io"
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
	metrics "github.com/mark3748/helpdesk-go/cmd/api/metrics"
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
		metrics.AttachmentsUploadedTotal.Inc()
		store, bucket := a.ResolveStore(c.Request.Context())
		if a.DB == nil || store == nil {
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
		oc, cancel := a.ObjCtx(c.Request.Context())
		defer cancel()
		if _, err := store.PutObject(oc, bucket, key, f, size, minio.PutObjectOptions{ContentType: ct}); err != nil {
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
		store, bucket := a.ResolveStore(c.Request.Context())
		if a.DB == nil {
			c.JSON(http.StatusOK, gin.H{"id": c.Param("attID")})
			return
		}
		const q = `select object_key, filename, mime from attachments where id=$1 and ticket_id=$2`
		var key, fn, mt string
		if err := a.DB.QueryRow(c.Request.Context(), q, c.Param("attID"), c.Param("id")).Scan(&key, &fn, &mt); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		if store == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "object store not configured"})
			return
		}

		// Prefer a presigned URL for MinIO/S3 stores so the client downloads
		// directly from object storage (short TTL) instead of proxying data
		// through the API; filesystem-backed stores are served by reading from
		// disk and streaming the bytes to the client.
		if mc, ok := store.(*minio.Client); ok {
			// Use internal S3 helper for consistent TTL
			svc := s3svc.Service{Client: mc, Bucket: bucket, MaxTTL: time.Minute}
			u, err := svc.PresignGet(c.Request.Context(), key, sanitizeFilename(fn), time.Minute)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.Redirect(http.StatusFound, u)
			return
		}

		// Serve from filesystem store when configured
		if fs, ok := store.(*app.FsObjectStore); ok {
			root := filepath.Join(fs.Base, bucket)
			path := filepath.Clean(filepath.Join(root, key))
			// Ensure the path is within the root (prevent traversal)
			if rel, err := filepath.Rel(root, path); err != nil || strings.HasPrefix(rel, "..") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
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

		// Fallback/Unknown store type
		c.JSON(http.StatusInternalServerError, gin.H{"error": "object store type not supported for download"})
	}
}

func PresignUpload(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		store, bucket := a.ResolveStore(c.Request.Context())
		if store == nil {
			c.JSON(http.StatusOK, gin.H{"url": "", "object_key": "temp"})
			return
		}
		// If it's MinIO, we can use our s3svc helper if we convert to *minio.Client
		// OR we use PresignedPutObject from interface (which we added!)

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

		// Only MinIO-backed stores support presigned uploads; others are not implemented.
		if _, ok := store.(*minio.Client); !ok {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "presign not supported"})
			return
		}

		// Use interface method
		u, err := store.PresignedPutObject(c.Request.Context(), bucket, key, time.Minute)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"url": u.String(), "object_key": key, "content_type": req.ContentType})
	}
}

func PresignDownload(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		store, bucket := a.ResolveStore(c.Request.Context())
		if a.DB == nil || store == nil {
			c.JSON(http.StatusOK, gin.H{"url": ""})
			return
		}
		// Logic similar to Get but returns JSON
		// Again, relying on type assertion for MinIO-specific s3svc helper for now
		// unless we add PresignedGet to interface.

		mc, ok := store.(*minio.Client)
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
		svc := s3svc.Service{Client: mc, Bucket: bucket, MaxTTL: time.Minute}
		oc, cancel := a.ObjCtx(c.Request.Context())
		defer cancel()
		u, err := svc.PresignGet(oc, key, sanitizeFilename(fn), time.Minute)
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
		// Remove object first when possible
		var key string
		_ = a.DB.QueryRow(c.Request.Context(), `select object_key from attachments where id=$1 and ticket_id=$2`, c.Param("attID"), c.Param("id")).Scan(&key)

		store, bucket := a.ResolveStore(c.Request.Context())
		if key != "" && store != nil {
			_ = store.RemoveObject(c.Request.Context(), bucket, key, minio.RemoveObjectOptions{})
		}
		const q = `delete from attachments where id=$1 and ticket_id=$2`
		if _, err := a.DB.Exec(c.Request.Context(), q, c.Param("attID"), c.Param("id")); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// Presign matches the main.go presignAttachment behavior: for MinIO, returns a
// real presigned PUT URL; for filesystem, returns an internal upload endpoint.
func Presign(a *app.App) gin.HandlerFunc {
	type presignReq struct {
		Filename string `json:"filename" binding:"required"`
		Bytes    int64  `json:"bytes" binding:"required"`
		Mime     string `json:"mime"`
	}
	return func(c *gin.Context) {
		store, bucket := a.ResolveStore(c.Request.Context())
		if store == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "object store not configured"})
			return
		}
		var in presignReq
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		objectKey := uuid.New().String()

		// Try using interface PresignedPutObject first
		u, err := store.PresignedPutObject(c.Request.Context(), bucket, objectKey, time.Minute)
		if err == nil && u != nil {
			headers := map[string]string{}
			if in.Mime != "" {
				headers["Content-Type"] = in.Mime
			}
			c.JSON(http.StatusCreated, gin.H{"upload_url": u.String(), "headers": headers, "attachment_id": objectKey})
			return
		}

		// Filesystem store fallback (PresignedPutObject returns error/nil)
		// We only support this fallback if we know it is FS store.
		if _, ok := store.(*app.FsObjectStore); ok {
			headers := map[string]string{}
			if in.Mime != "" {
				headers["Content-Type"] = in.Mime
			}
			c.JSON(http.StatusCreated, gin.H{"upload_url": "/api/attachments/upload/" + objectKey, "headers": headers, "attachment_id": objectKey})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "presign failed"})
	}
}

// UploadObject handles PUT uploads when using the filesystem store.
func UploadObject(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		store, bucket := a.ResolveStore(c.Request.Context())
		if store == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "object store not configured"})
			return
		}
		// Disallow when using MinIO client; must use presigned URL
		if _, ok := store.(*minio.Client); ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid upload target"})
			return
		}
		objectKey := strings.TrimSpace(c.Param("objectKey"))
		if _, err := uuid.Parse(objectKey); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid object key"})
			return
		}
		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "read body"})
			return
		}
		ct := c.GetHeader("Content-Type")
		if ct == "" {
			ct = "application/octet-stream"
		}
		oc, cancel := a.ObjCtx(c.Request.Context())
		defer cancel()
		if _, err := store.PutObject(oc, bucket, objectKey, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{ContentType: ct}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	}
}

// Finalize records the attachment metadata after a successful upload.
func Finalize(a *app.App) gin.HandlerFunc {
	type finalizeReq struct {
		AttachmentID string `json:"attachment_id" binding:"required"`
		Filename     string `json:"filename" binding:"required"`
		Bytes        int64  `json:"bytes" binding:"required"`
		Mime         string `json:"mime"`
	}
	return func(c *gin.Context) {
		metrics.AttachmentsUploadedTotal.Inc()
		store, bucket := a.ResolveStore(c.Request.Context())
		if store == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "object store not configured"})
			return
		}
		uVal, ok := c.Get("user")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}
		au, _ := uVal.(authpkg.AuthUser)
		ticketID := c.Param("id")
		var in finalizeReq
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if _, err := uuid.Parse(strings.TrimSpace(in.AttachmentID)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid attachment_id"})
			return
		}
		var size int64

		// Use StatObject interface method
		oc, cancel := a.ObjCtx(c.Request.Context())
		defer cancel()
		info, err := store.StatObject(oc, bucket, in.AttachmentID, minio.StatObjectOptions{})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "upload incomplete"})
			return
		}
		size = info.Size

		if size != in.Bytes {
			c.JSON(http.StatusBadRequest, gin.H{"error": "upload incomplete"})
			return
		}
		if _, err := a.DB.Exec(c.Request.Context(), `insert into attachments (id, ticket_id, uploader_id, object_key, filename, bytes, mime) values ($1,$2,$3,$4,$5,$6,$7)`,
			in.AttachmentID, ticketID, au.ID, in.AttachmentID, in.Filename, in.Bytes, in.Mime); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		eventspkg.Emit(c.Request.Context(), a.DB, ticketID, "ticket_updated", map[string]any{"id": ticketID})
		c.JSON(http.StatusCreated, gin.H{"id": in.AttachmentID})
	}
}
