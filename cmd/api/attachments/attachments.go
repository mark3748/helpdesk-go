package attachments

import (
    "os"
    "mime"
    "net/http"
    "path/filepath"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/minio/minio-go/v7"
    app "github.com/mark3748/helpdesk-go/cmd/api/app"
    authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
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
        type att struct{ ID string `json:"id"`; Filename string `json:"filename"`; Bytes int64 `json:"bytes"` }
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
        key := uuid.New().String() + "-" + header.Filename
        size := header.Size
        ct := header.Header.Get("Content-Type")
        if ct == "" { ct = mime.TypeByExtension(filepath.Ext(header.Filename)) }
        if _, err := a.M.PutObject(c.Request.Context(), a.Cfg.MinIOBucket, key, f, size, minio.PutObjectOptions{ContentType: ct}); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        const q = `insert into attachments (ticket_id, uploader_id, object_key, filename, bytes) values ($1, $2, $3, $4, $5) returning id::text`
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
        if err := a.DB.QueryRow(c.Request.Context(), q, c.Param("id"), uploader, key, header.Filename, size).Scan(&id); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusCreated, gin.H{"id": id})
    }
}

func Get(a *app.App) gin.HandlerFunc {
    return func(c *gin.Context) {
        if a.DB == nil {
            c.JSON(http.StatusOK, gin.H{"id": c.Param("attID")})
            return
        }
        const q = `select object_key, filename, bytes from attachments where id=$1 and ticket_id=$2`
        var key, fn string
        var size int64
        if err := a.DB.QueryRow(c.Request.Context(), q, c.Param("attID"), c.Param("id")).Scan(&key, &fn, &size); err != nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
            return
        }
        // Serve from filesystem store when configured
        if fs, ok := a.M.(*app.FsObjectStore); ok {
            path := fs.Base + string(filepath.Separator) + a.Cfg.MinIOBucket + string(filepath.Separator) + key
            f, err := os.ReadFile(path)
            if err != nil {
                c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
                return
            }
            c.Writer.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(fn)))
            c.Writer.Header().Set("Content-Disposition", "attachment; filename=\""+strings.ReplaceAll(fn, "\"", "")+"\"")
            _, _ = c.Writer.Write(f)
            return
        }
        // Otherwise unimplemented (e.g., MinIO); client may handle 501
        c.JSON(http.StatusNotImplemented, gin.H{"error": "download not implemented"})
    }
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
