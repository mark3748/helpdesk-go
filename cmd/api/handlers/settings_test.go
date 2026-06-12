package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
)

type fakeRow struct {
	scan func(dest ...any) error
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.scan != nil {
		return r.scan(dest...)
	}
	return nil
}

type fakeDB struct {
	s Settings
}

func (db *fakeDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	lower := strings.ToLower(strings.TrimSpace(sql))
	if strings.HasPrefix(lower, "select") {
		return &fakeRow{scan: func(dest ...any) error {
			b, _ := json.Marshal(db.s.Storage)
			if p, ok := dest[0].(*[]byte); ok {
				*p = b
			}
			b, _ = json.Marshal(db.s.OIDC)
			if p, ok := dest[1].(*[]byte); ok {
				*p = b
			}
			b, _ = json.Marshal(db.s.Mail)
			if p, ok := dest[2].(*[]byte); ok {
				*p = b
			}
			b, _ = json.Marshal(db.s.Discord)
			if p, ok := dest[3].(*[]byte); ok {
				*p = b
			}
			if p, ok := dest[4].(*string); ok {
				*p = db.s.LogPath
			}
			if p, ok := dest[5].(**time.Time); ok {
				if db.s.LastTest != "" {
					t, _ := time.Parse(time.RFC3339, db.s.LastTest)
					*p = &t
				} else {
					*p = nil
				}
			}
			return nil
		}}
	}
	return &fakeRow{}
}

func (db *fakeDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	s := strings.ToLower(sql)
	switch {
	case strings.Contains(s, "insert into settings"):
		if len(args) > 0 {
			if lp, ok := args[len(args)-1].(string); ok && lp != "" {
				db.s.LogPath = lp
			}
		}
	case strings.Contains(s, "update settings set storage"):
		switch v := args[0].(type) {
		case string:
			_ = json.Unmarshal([]byte(v), &db.s.Storage)
		case []byte:
			_ = json.Unmarshal(v, &db.s.Storage)
		}
	case strings.Contains(s, "update settings set oidc"):
		switch v := args[0].(type) {
		case string:
			_ = json.Unmarshal([]byte(v), &db.s.OIDC)
		case []byte:
			_ = json.Unmarshal(v, &db.s.OIDC)
		}
	case strings.Contains(s, "update settings set mail"):
		switch v := args[0].(type) {
		case string:
			_ = json.Unmarshal([]byte(v), &db.s.Mail)
		case []byte:
			_ = json.Unmarshal(v, &db.s.Mail)
		}
	case strings.Contains(s, "update settings set discord"):
		switch v := args[0].(type) {
		case string:
			_ = json.Unmarshal([]byte(v), &db.s.Discord)
		case []byte:
			_ = json.Unmarshal(v, &db.s.Discord)
		}
	case strings.Contains(s, "update settings set last_test"):
		if t, ok := args[0].(time.Time); ok {
			db.s.LastTest = t.Format(time.RFC3339)
		}
	case strings.Contains(s, "update settings set log_path"):
		if lp, ok := args[0].(string); ok {
			db.s.LogPath = lp
		}
	}
	return pgconn.CommandTag{}, nil
}

func TestSettingsRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &fakeDB{s: Settings{Storage: map[string]string{}, OIDC: OIDCSettings{}, Mail: map[string]string{}}}
	InitSettings(context.Background(), db, "/tmp/logs")

	called := false
	smtpSendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		called = true
		return nil
	}
	defer func() { smtpSendMail = smtp.SendMail }()

	r := gin.New()
	r.GET("/settings/oidc", GetOIDCSettings(db))
	r.PUT("/settings/oidc", PutOIDCSettings(db))
	r.GET("/settings/mail", GetMailSettings(db))
	r.PUT("/settings/mail", PutMailSettings(db))
	r.POST("/settings/mail/test", TestMailSettings(db))

	// OIDC round trip
	body := bytes.NewBufferString(`{"issuer":"https://id"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/settings/oidc", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/settings/oidc", nil)
	r.ServeHTTP(w, req)
	var oidc map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &oidc); err != nil {
		t.Fatal(err)
	}
	if oidc["issuer"] != "https://id" {
		t.Fatalf("expected issuer saved, got %#v", oidc)
	}

	// Mail round trip
	body = bytes.NewBufferString(`{"smtp_host":"smtp","smtp_port":"25","smtp_from":"a@b"}`)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPut, "/settings/mail", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/settings/mail", nil)
	r.ServeHTTP(w, req)
	var mail map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &mail); err != nil {
		t.Fatal(err)
	}
	if mail["smtp_host"] != "smtp" {
		t.Fatalf("expected smtp_host saved, got %#v", mail)
	}

	// send test mail
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/settings/mail/test", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	if !called {
		t.Fatalf("expected smtpSendMail called")
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["last_test"] == "" {
		t.Fatalf("expected last_test in response")
	}
}

func TestSettingsRoleGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &fakeDB{s: Settings{Mail: map[string]string{}}}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user", authpkg.AuthUser{Roles: []string{"agent"}}) })
	r.GET("/settings/mail", authpkg.RequireRole("admin"), GetMailSettings(db))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/settings/mail", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", w.Code)
	}
}

func TestMailTestMissingConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &fakeDB{s: Settings{Mail: map[string]string{}}}
	r := gin.New()
	r.POST("/settings/mail/test", TestMailSettings(db))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/settings/mail/test", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestMailTestDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &fakeDB{s: Settings{Mail: map[string]string{"smtp_host": "smtp"}}}
	called := false
	smtpSendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		called = true
		if addr != "smtp:25" {
			t.Fatalf("expected default port, got %s", addr)
		}
		if from != "test@example.com" {
			t.Fatalf("expected default from, got %s", from)
		}
		return nil
	}
	defer func() { smtpSendMail = smtp.SendMail }()

	r := gin.New()
	r.POST("/settings/mail/test", TestMailSettings(db))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/settings/mail/test", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	if !called {
		t.Fatalf("expected smtpSendMail called")
	}
}

func TestPublicMailSettingsMergesEnvironmentAndRedactsPasswords(t *testing.T) {
	t.Setenv("SMTP_HOST", "env-smtp.example.com")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_PASS", "env-secret")
	t.Setenv("IMAP_PASS", "env-imap-secret")

	got := publicMailSettings(map[string]string{"smtp_host": "db-smtp.example.com"})
	if got["smtp_host"] != "db-smtp.example.com" {
		t.Fatalf("database setting did not override environment: %#v", got)
	}
	if got["smtp_port"] != "587" {
		t.Fatalf("environment fallback missing: %#v", got)
	}
	if got["smtp_pass"] != "" || got["imap_pass"] != "" {
		t.Fatalf("passwords exposed in public settings: %#v", got)
	}
	if got["smtp_pass_configured"] != "true" || got["imap_pass_configured"] != "true" {
		t.Fatalf("password configured flags missing: %#v", got)
	}
}

func TestPrepareMailSettingsUpdatePreservesStoredPasswords(t *testing.T) {
	db := &fakeDB{s: Settings{Mail: map[string]string{
		"smtp_host": "smtp.example.com",
		"smtp_pass": "stored-smtp-secret",
		"imap_pass": "stored-imap-secret",
	}}}
	InitSettings(context.Background(), db, "/tmp/logs")

	got := prepareMailSettingsUpdate(context.Background(), map[string]string{
		"smtp_host": "new-smtp.example.com",
		"smtp_pass": "",
		"imap_pass": "",
	})
	if got["smtp_pass"] != "stored-smtp-secret" || got["imap_pass"] != "stored-imap-secret" {
		t.Fatalf("blank password fields did not preserve stored secrets: %#v", got)
	}
	if got["smtp_host"] != "new-smtp.example.com" {
		t.Fatalf("non-secret update was not applied: %#v", got)
	}
}

func TestPublicDiscordSettingsMergesEnvironmentAndRedactsToken(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "env-token")
	t.Setenv("DISCORD_GUILD_ID", "env-guild")
	t.Setenv("DISCORD_CHANNEL_ID", "env-channel")

	got := publicDiscordSettings(map[string]string{"guild_id": "db-guild"})
	if got["guild_id"] != "db-guild" || got["channel_id"] != "env-channel" {
		t.Fatalf("Discord settings were not merged: %#v", got)
	}
	if got["bot_token"] != "" || got["bot_token_configured"] != "true" {
		t.Fatalf("Discord token was exposed or configured flag missing: %#v", got)
	}
}

func TestSaveDiscordSettingsPreservesStoredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &fakeDB{s: Settings{Discord: map[string]string{
		"bot_token":  "stored-token",
		"guild_id":   "old-guild",
		"channel_id": "old-channel",
	}}}
	InitSettings(context.Background(), db, "/tmp/logs")

	r := gin.New()
	r.POST("/settings/discord", SaveDiscordSettings)
	body := bytes.NewBufferString(`{"bot_token":"","guild_id":"new-guild","channel_id":"new-channel"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/settings/discord", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	if db.s.Discord["bot_token"] != "stored-token" || db.s.Discord["guild_id"] != "new-guild" {
		t.Fatalf("Discord settings were not saved correctly: %#v", db.s.Discord)
	}
}

func TestStorageConnectionHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/settings/storage/test", TestStorageConnection)

	// Test missing fields
	body := bytes.NewBufferString(`{}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/settings/storage/test", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}

	// Test connection failure (dummy endpoint)
	body = bytes.NewBufferString(`{
		"endpoint": "localhost:12345",
		"bucket": "test",
		"access_key_id": "minio",
		"secret_access_key": "minio123",
		"use_ssl": "false"
	}`)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/settings/storage/test", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["ok"] == true {
		t.Fatalf("expected connection failure")
	}
	// Test scheme stripping and SSL auto-detect
	// We provide https:// but use_ssl="false". The handler should switch use_ssl to true.
	// Since we can't easily check the internal bool, we rely on the fact that
	// if it DIDN'T switch, it would try HTTP -> HTTPS and fail with "Client sent an HTTP request..."
	// or effectively succeed if the mockup server handles it.
	// But here we are mocking minio.New? No, we are using the real minio.New but checking for error.
	// To comfortably test this without a real MinIO, we just ensure it doesn't error on the SCHEME.
	body = bytes.NewBufferString(`{
		"endpoint": "https://localhost:12345",
		"bucket": "test",
		"access_key_id": "minio",
		"secret_access_key": "minio123",
		"use_ssl": "false"
	}`)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/settings/storage/test", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	// We expect connection failure (no server), but NOT "fully qualified paths"
	if resp["error"] != nil {
		errStr := resp["error"].(string)
		if strings.Contains(errStr, "fully qualified paths") {
			t.Fatalf("scheme not stripped: %s", errStr)
		}
	}
}
