package queues

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
)

type qrow struct{ Queue }

type qrows struct {
	data []qrow
	idx  int
}

func (r *qrows) Close()                                       {}
func (r *qrows) Err() error                                   { return nil }
func (r *qrows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *qrows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *qrows) Next() bool                                   { return r.idx < len(r.data) }
func (r *qrows) Values() ([]any, error)                       { return nil, nil }
func (r *qrows) RawValues() [][]byte                          { return nil }
func (r *qrows) Conn() *pgx.Conn                              { return nil }
func (r *qrows) Scan(dest ...any) error {
	row := r.data[r.idx]
	r.idx++
	*(dest[0].(*string)) = row.ID
	*(dest[1].(*string)) = row.Name
	return nil
}

type qdb struct{ rows []qrow }

func (db *qdb) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return &qrows{data: db.rows}, nil
}
func (db *qdb) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row { return nil }
func (db *qdb) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func TestQueueList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &qdb{rows: []qrow{{Queue{ID: "1", Name: "Alpha"}}, {Queue{ID: "2", Name: "Beta"}}}}
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, db, nil, nil, nil)
	a.R.GET("/queues", authpkg.Middleware(a), List(a))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/queues", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var out []Queue
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil || len(out) != 2 {
		t.Fatalf("unexpected body: %v %v", out, err)
	}
}

func TestQueueListAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &qdb{}
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, db, nil, nil, nil)
	a.R.GET("/queues", authpkg.Middleware(a), func(c *gin.Context) { c.Set("user", authpkg.AuthUser{}) }, List(a))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/queues", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}
