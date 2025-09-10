package watchers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
)

type fakeDB struct{ watchers map[string]map[string]bool }

func (db *fakeDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	ticketID := args[0].(string)
	f := &fakeRows{}
	if m, ok := db.watchers[ticketID]; ok {
		for uid := range m {
			f.list = append(f.list, uid)
		}
	}
	return f, nil
}

func (db *fakeDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row { return &fakeRow{} }

func (db *fakeDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	s := strings.ToLower(sql)
	if !strings.Contains(s, "ticket_watchers") {
		return pgconn.CommandTag{}, nil
	}
	ticketID := args[0].(string)
	userID := args[1].(string)
	if db.watchers == nil {
		db.watchers = map[string]map[string]bool{}
	}
	if _, ok := db.watchers[ticketID]; !ok {
		db.watchers[ticketID] = map[string]bool{}
	}
	if strings.HasPrefix(s, "insert") {
		db.watchers[ticketID][userID] = true
	} else {
		delete(db.watchers[ticketID], userID)
	}
	return pgconn.CommandTag{}, nil
}

func (db *fakeDB) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, nil
}

type fakeRows struct {
	list []string
	i    int
}

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return nil }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) RawValues() [][]byte                          { return nil }
func (r *fakeRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeRows) Conn() *pgx.Conn                              { return nil }
func (r *fakeRows) Next() bool {
	if r.i >= len(r.list) {
		return false
	}
	r.i++
	return true
}
func (r *fakeRows) Scan(dest ...any) error {
	if r.i == 0 || r.i > len(r.list) {
		return nil
	}
	if p, ok := dest[0].(*string); ok {
		*p = r.list[r.i-1]
	}
	return nil
}

type fakeRow struct{}

func (r *fakeRow) Scan(dest ...any) error { return pgx.ErrNoRows }

func TestWatcherHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &fakeDB{}
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, db, nil, nil, nil)
	a.R.GET("/tickets/:id/watchers", authpkg.Middleware(a), List(a))
	a.R.POST("/tickets/:id/watchers", authpkg.Middleware(a), Add(a))
	a.R.DELETE("/tickets/:id/watchers/:uid", authpkg.Middleware(a), Remove(a))

	// Add watcher
	body := bytes.NewBufferString(`{"user_id":"u1"}`)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tickets/1/watchers", body)
	req.Header.Set("Content-Type", "application/json")
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	// List should return inserted watcher
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/tickets/1/watchers", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var out []string
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "u1" {
		t.Fatalf("expected [u1], got %v", out)
	}

	// Remove watcher
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/tickets/1/watchers/u1", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// List should now be empty
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/tickets/1/watchers", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	out = nil
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty, got %v", out)
	}
}
