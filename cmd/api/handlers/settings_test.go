package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
		if b, ok := args[0].([]byte); ok {
			_ = json.Unmarshal(b, &db.s.Storage)
		}
	case strings.Contains(s, "update settings set oidc"):
		if b, ok := args[0].([]byte); ok {
			_ = json.Unmarshal(b, &db.s.OIDC)
		}
	case strings.Contains(s, "update settings set mail"):
		if b, ok := args[0].([]byte); ok {
			_ = json.Unmarshal(b, &db.s.Mail)
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

func TestSettingsHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &fakeDB{s: Settings{Storage: map[string]string{}, OIDC: map[string]string{}, Mail: map[string]string{}}}
	InitSettings(context.Background(), db, "/tmp/logs")
	r := gin.New()
	r.GET("/settings", GetSettings(db))
	r.POST("/settings/storage", SaveStorageSettings(db))
	r.POST("/test-connection", TestConnection(db))

	// initial log path
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/settings", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var resp Settings
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.LogPath != "/tmp/logs" {
		t.Fatalf("unexpected log path %s", resp.LogPath)
	}

	// save storage config
	body := bytes.NewBufferString(`{"endpoint":"s3","bucket":"b"}`)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/settings/storage", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}

	// ensure persisted
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/settings", nil)
	r.ServeHTTP(w, req)
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Storage["endpoint"] != "s3" {
		t.Fatalf("expected endpoint saved")
	}

	// test connection updates last test
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/test-connection", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/settings", nil)
	r.ServeHTTP(w, req)
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.LastTest == "" {
		t.Fatalf("expected last test set")
	}
}
