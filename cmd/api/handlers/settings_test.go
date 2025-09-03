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
			if p, ok := dest[3].(*string); ok {
				*p = db.s.LogPath
			}
			if p, ok := dest[4].(**time.Time); ok {
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
	db := &fakeDB{s: Settings{Storage: map[string]string{}, OIDC: map[string]string{}, Mail: map[string]string{}}}
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
