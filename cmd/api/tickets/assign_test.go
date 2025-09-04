package tickets

import (
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

type assignRow struct{ Ticket }

func (r *assignRow) Scan(dest ...any) error {
	*(dest[0].(*string)) = r.ID
	*(dest[1].(*any)) = r.Number
	*(dest[2].(*string)) = r.Title
	*(dest[3].(*string)) = r.Status
	*(dest[4].(**string)) = r.AssigneeID
	*(dest[5].(*int16)) = r.Priority
	return nil
}

type assignDB struct {
	sql  string
	args []any
}

func (db *assignDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return nil, nil
}
func (db *assignDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (db *assignDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	db.sql = sql
	db.args = args
	assignee := "a1"
	t := Ticket{ID: "1", Title: "t", Status: "Open", Priority: 1, AssigneeID: &assignee}
	return &assignRow{t}
}

func TestAssign(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &assignDB{}
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, db, nil, nil, nil)
	a.R.POST("/tickets/:id/assign", authpkg.Middleware(a), Assign(a))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tickets/1/assign", strings.NewReader(`{"assignee_id":"a1"}`))
	req.Header.Set("Content-Type", "application/json")
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var out Ticket
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil || out.AssigneeID == nil || *out.AssigneeID != "a1" {
		t.Fatalf("unexpected ticket: %v %v", out, err)
	}
	if !strings.Contains(db.sql, "update tickets set assignee_id") {
		t.Fatalf("unexpected sql: %s", db.sql)
	}
	if len(db.args) != 2 || db.args[0] != "a1" || db.args[1] != "1" {
		t.Fatalf("unexpected args: %v", db.args)
	}
}

func TestAssignAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &assignDB{}
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, db, nil, nil, nil)
	a.R.POST("/tickets/:id/assign", authpkg.Middleware(a), func(c *gin.Context) { c.Set("user", authpkg.AuthUser{}) }, Assign(a))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tickets/1/assign", strings.NewReader(`{"assignee_id":"a1"}`))
	req.Header.Set("Content-Type", "application/json")
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}
