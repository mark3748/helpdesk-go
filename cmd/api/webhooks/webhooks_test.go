package webhooks

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

type fakeDB struct{ hooks map[string]hook }

type hook struct {
	ID        string
	TargetURL string
	EventMask int
	Secret    string
	Active    bool
}

func (db *fakeDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	fr := &fakeRows{}
	for _, h := range db.hooks {
		fr.list = append(fr.list, h)
	}
	return fr, nil
}

func (db *fakeDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row { return fakeRow{} }

func (db *fakeDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if db.hooks == nil {
		db.hooks = map[string]hook{}
	}
	s := strings.ToLower(sql)
	if strings.HasPrefix(s, "insert") {
		id := "1"
		db.hooks[id] = hook{ID: id, TargetURL: args[0].(string), EventMask: args[1].(int), Secret: args[2].(string), Active: args[3].(bool)}
	} else if strings.HasPrefix(s, "delete") {
		id := args[0].(string)
		delete(db.hooks, id)
	}
	return pgconn.CommandTag{}, nil
}

func (db *fakeDB) Begin(ctx context.Context) (pgx.Tx, error) { return nil, nil }

type fakeRows struct {
	list []hook
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
	h := r.list[r.i-1]
	if p, ok := dest[0].(*string); ok {
		*p = h.ID
	}
	if p, ok := dest[1].(*string); ok {
		*p = h.TargetURL
	}
	if p, ok := dest[2].(*int); ok {
		*p = h.EventMask
	}
	if p, ok := dest[3].(*string); ok {
		*p = h.Secret
	}
	if p, ok := dest[4].(*bool); ok {
		*p = h.Active
	}
	return nil
}

type fakeRow struct{}

func (fakeRow) Scan(dest ...any) error { return pgx.ErrNoRows }

func TestWebhookHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &fakeDB{}
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, db, nil, nil, nil)
	a.R.GET("/webhooks", authpkg.Middleware(a), List(a))
	a.R.POST("/webhooks", authpkg.Middleware(a), Create(a))
	a.R.DELETE("/webhooks/:id", authpkg.Middleware(a), Delete(a))

	body := bytes.NewBufferString(`{"target_url":"https://example.com","event_mask":1,"secret":"s","active":true}`)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks", body)
	req.Header.Set("Content-Type", "application/json")
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/webhooks", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var out []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0]["target_url"].(string) != "https://example.com" {
		t.Fatalf("unexpected list: %v", out)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/webhooks/1", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
