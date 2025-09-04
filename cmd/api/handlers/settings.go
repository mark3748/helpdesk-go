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

// OIDCSettings holds OpenID Connect configuration.
type OIDCSettings struct {
	Issuer       string              `json:"issuer"`
	ClientID     string              `json:"client_id"`
	ClaimPath    string              `json:"claim_path"`
	ValueToRoles map[string][]string `json:"value_to_roles"`
}

// Settings represents persisted configuration values.
type Settings struct {
	Storage  map[string]string `json:"storage"`
	OIDC     OIDCSettings      `json:"oidc"`
	Mail     map[string]string `json:"mail"`
	LogPath  string            `json:"log_path"`
	LastTest string            `json:"last_test"`
}

// Package-level state wired from main at startup
var (
	dbStore    DB
	startupLog string
	// EnqueueEmail is set by the API to enqueue email jobs.
	EnqueueEmail func(ctx context.Context, to, template string, data interface{})
)

// InitSettings ensures a row exists, sets initial log path, and stores DB handle.
func InitSettings(ctx context.Context, db DB, logPath string) {
	dbStore = db
	startupLog = logPath
	if db == nil {
		return
	}
	_, _ = db.Exec(ctx, "insert into settings (id, log_path) values (1, $1) on conflict (id) do nothing", logPath)
	if logPath != "" {
		_, _ = db.Exec(ctx, "update settings set log_path=$1 where id=1", logPath)
	}
}

func loadSettings(ctx context.Context) (Settings, error) {
	var s Settings
	if dbStore == nil {
		s.Storage = map[string]string{}
		s.OIDC = OIDCSettings{}
		s.Mail = map[string]string{}
		s.LogPath = startupLog
		return s, nil
	}
	var storage, oidc, mail []byte
	var lt *time.Time
	row := dbStore.QueryRow(ctx, "select storage, oidc, mail, log_path, last_test from settings where id=1")
	err := row.Scan(&storage, &oidc, &mail, &s.LogPath, &lt)
	if err != nil {
		if err == pgx.ErrNoRows {
			s.Storage = map[string]string{}
			s.OIDC = OIDCSettings{}
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
func GetSettings(c *gin.Context) {
	s, err := loadSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s)
}

// SaveStorageSettings stores storage configuration.
func SaveStorageSettings(c *gin.Context) {
	if dbStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	var data map[string]string
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	b, _ := json.Marshal(data)
	if _, err := dbStore.Exec(c.Request.Context(), "update settings set storage=$1::jsonb where id=1", string(b)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// SaveOIDCSettings stores OIDC configuration.
func SaveOIDCSettings(c *gin.Context) {
	if dbStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	var data OIDCSettings
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	b, _ := json.Marshal(data)
	if _, err := dbStore.Exec(c.Request.Context(), "update settings set oidc=$1::jsonb where id=1", string(b)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// SaveMailSettings stores mail configuration.
func SaveMailSettings(c *gin.Context) {
	if dbStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	var data map[string]string
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	b, _ := json.Marshal(data)
	if _, err := dbStore.Exec(c.Request.Context(), "update settings set mail=$1::jsonb where id=1", string(b)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// MailSettings returns the current mail settings (from DB).
func MailSettings() map[string]string {
	s, err := loadSettings(context.Background())
	if err != nil {
		return map[string]string{}
	}
	return s.Mail
}

// SendTestMail enqueues a test email via the worker.
func SendTestMail(c *gin.Context) {
	if EnqueueEmail == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mail queue unavailable"})
		return
	}
	to := c.Query("to")
	if to == "" {
		to = MailSettings()["smtp_from"]
	}
	EnqueueEmail(c.Request.Context(), to, "test_email", nil)
	c.JSON(http.StatusOK, gin.H{"queued": true})
}

// TestConnection records a test timestamp and returns log path and last result.
func TestConnection(c *gin.Context) {
	if dbStore == nil {
		c.JSON(http.StatusOK, gin.H{"ok": true, "log_path": startupLog, "last_test": time.Now().Format(time.RFC3339)})
		return
	}
	now := time.Now()
	_, _ = dbStore.Exec(c.Request.Context(), "update settings set last_test=$1 where id=1", now)
	s, _ := loadSettings(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"ok": true, "log_path": s.LogPath, "last_test": now.Format(time.RFC3339)})
}
