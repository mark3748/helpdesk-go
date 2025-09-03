package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/smtp"
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

var smtpSendMail = smtp.SendMail

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
		if _, err := db.Exec(c.Request.Context(), "update settings set storage=$1::jsonb where id=1", string(b)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// GetOIDCSettings returns OIDC configuration.
func GetOIDCSettings(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := loadSettings(c.Request.Context(), db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, s.OIDC)
	}
}

// PutOIDCSettings stores OIDC configuration.
func PutOIDCSettings(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data map[string]string
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		b, _ := json.Marshal(data)
		if _, err := db.Exec(c.Request.Context(), "update settings set oidc=$1::jsonb where id=1", string(b)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// GetMailSettings returns mail configuration.
func GetMailSettings(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := loadSettings(c.Request.Context(), db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, s.Mail)
	}
}

// PutMailSettings stores mail configuration.
func PutMailSettings(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data map[string]string
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		b, _ := json.Marshal(data)
		if _, err := db.Exec(c.Request.Context(), "update settings set mail=$1::jsonb where id=1", string(b)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// TestMailSettings sends a test email and records the attempt.
func TestMailSettings(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := loadSettings(c.Request.Context(), db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		host := s.Mail["smtp_host"]
		port := s.Mail["smtp_port"]
		from := s.Mail["smtp_from"]
		if host == "" || port == "" || from == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "smtp config incomplete"})
			return
		}
		addr := host + ":" + port
		var auth smtp.Auth
		user := s.Mail["smtp_user"]
		pass := s.Mail["smtp_pass"]
		if user != "" {
			auth = smtp.PlainAuth("", user, pass, host)
		}
		msg := []byte("To: " + from + "\r\nSubject: Test Mail\r\n\r\nThis is a test email.")
		if err := smtpSendMail(addr, auth, from, []string{from}, msg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		now := time.Now()
		_, _ = db.Exec(c.Request.Context(), "update settings set last_test=$1 where id=1", now)
		s.LastTest = now.Format(time.RFC3339)
		c.JSON(http.StatusOK, gin.H{"ok": true, "log_path": s.LogPath, "last_test": s.LastTest})
	}
}
