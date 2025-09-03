package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DB defines the database methods used by the settings handlers.
type DB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Settings represents persisted configuration values.
type Settings struct {
	Storage  map[string]string `json:"storage"`
	OIDC     map[string]string `json:"oidc"`
	Mail     map[string]string `json:"mail"`
	LogPath  string            `json:"log_path"`
	LastTest string            `json:"last_test"`
}

// InitSettings ensures a row exists and sets initial log path.
func InitSettings(ctx context.Context, db DB, logPath string) {
	if db == nil {
		return
	}
	_, _ = db.Exec(ctx, "insert into settings (id, log_path) values (1, $1) on conflict (id) do nothing", logPath)
	if logPath != "" {
		_, _ = db.Exec(ctx, "update settings set log_path=$1 where id=1", logPath)
	}
}

func loadSettings(ctx context.Context, db DB) (Settings, error) {
	var s Settings
	var storage, oidc, mail []byte
	var lt *time.Time
	row := db.QueryRow(ctx, "select storage, oidc, mail, log_path, last_test from settings where id=1")
	err := row.Scan(&storage, &oidc, &mail, &s.LogPath, &lt)
	if err != nil {
		if err == pgx.ErrNoRows {
			s.Storage = map[string]string{}
			s.OIDC = map[string]string{}
			s.Mail = map[string]string{}
			s.LogPath = "/config/logs"
			return s, nil
		}
		return s, err
	}
	if len(storage) > 0 {
		_ = json.Unmarshal(storage, &s.Storage)
	} else {
		s.Storage = map[string]string{}
	}
	if len(oidc) > 0 {
		_ = json.Unmarshal(oidc, &s.OIDC)
	} else {
		s.OIDC = map[string]string{}
	}
	if len(mail) > 0 {
		_ = json.Unmarshal(mail, &s.Mail)
	} else {
		s.Mail = map[string]string{}
	}
	if lt != nil {
		s.LastTest = lt.Format(time.RFC3339)
	}
	return s, nil
}

// GetSettings returns the current configuration from the database.
func GetSettings(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := loadSettings(c.Request.Context(), db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, s)
	}
}

// SaveStorageSettings stores storage configuration.
func SaveStorageSettings(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data map[string]string
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		b, _ := json.Marshal(data)
		if _, err := db.Exec(c.Request.Context(), "update settings set storage=$1 where id=1", b); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// SaveOIDCSettings stores OIDC configuration.
func SaveOIDCSettings(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data map[string]string
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		b, _ := json.Marshal(data)
		if _, err := db.Exec(c.Request.Context(), "update settings set oidc=$1 where id=1", b); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// SaveMailSettings stores mail configuration.
func SaveMailSettings(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data map[string]string
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		b, _ := json.Marshal(data)
		if _, err := db.Exec(c.Request.Context(), "update settings set mail=$1 where id=1", b); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// TestConnection records a test run and returns log path and last result.
func TestConnection(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now()
		if _, err := db.Exec(c.Request.Context(), "update settings set last_test=$1 where id=1", now); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		s, err := loadSettings(c.Request.Context(), db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "log_path": s.LogPath, "last_test": s.LastTest})
	}
}
