package exports

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/mark3748/helpdesk-go/cmd/api/app"
	"github.com/minio/minio-go/v7"
)

type TicketsReq struct {
	IDs []string `json:"ids" binding:"required"`
}

func Tickets(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.M == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "minio not configured"})
			return
		}
		var in TicketsReq
		if err := c.ShouldBindJSON(&in); err != nil || len(in.IDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
			return
		}
		ctx := c.Request.Context()
		placeholders := make([]string, len(in.IDs))
		args := make([]any, len(in.IDs))
		for i, id := range in.IDs {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args[i] = id
		}
		q := fmt.Sprintf("select id, number, title, status, priority from tickets where id in (%s)", strings.Join(placeholders, ","))
		rows, err := a.DB.Query(ctx, q, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		buf := &bytes.Buffer{}
		w := csv.NewWriter(buf)
		_ = w.Write([]string{"id", "number", "title", "status", "priority"})
		for rows.Next() {
			var id, number, title, status string
			var priority int16
			if err := rows.Scan(&id, &number, &title, &status, &priority); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			_ = w.Write([]string{id, number, title, status, strconv.Itoa(int(priority))})
		}
		w.Flush()
		objectKey := uuid.New().String() + ".csv"
		oc, cancel := a.ObjCtx(ctx)
		defer cancel()
		_, err = a.M.PutObject(oc, a.Cfg.MinIOBucket, objectKey, bytes.NewReader(buf.Bytes()), int64(buf.Len()), minio.PutObjectOptions{ContentType: "text/csv"})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if mc, ok := a.M.(*minio.Client); ok {
			oc, cancel := a.ObjCtx(ctx)
			defer cancel()
			url, err := mc.PresignedGetObject(oc, a.Cfg.MinIOBucket, objectKey, time.Minute, nil)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"url": url.String()})
			return
		}
		scheme := "http"
		if a.Cfg.MinIOUseSSL {
			scheme = "https"
		}
		url := fmt.Sprintf("%s://%s/%s/%s", scheme, a.Cfg.MinIOEndpoint, a.Cfg.MinIOBucket, objectKey)
		c.JSON(http.StatusOK, gin.H{"url": url})
	}
}
