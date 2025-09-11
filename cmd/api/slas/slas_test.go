package slas

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

type fakeDB struct{ rows []slapolicy }

type slapolicy struct {
	id         string
	name       string
	priority   int
	response   int
	resolution int
	update     *int
}

func (db *fakeDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	r := &fakeRows{rows: db.rows}
	return r, nil
}
func (db *fakeDB) QueryRow(context.Context, string, ...any) pgx.Row { return nil }
func (db *fakeDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (db *fakeDB) Begin(context.Context) (pgx.Tx, error) { return nil, nil }

type fakeRows struct {
	rows []slapolicy
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
	if r.i >= len(r.rows) {
		return false
	}
	r.i++
	return true
}
func (r *fakeRows) Scan(dest ...any) error {
	if r.i == 0 || r.i > len(r.rows) {
		return pgx.ErrNoRows
	}
	row := r.rows[r.i-1]
	if p, ok := dest[0].(*string); ok {
		*p = row.id
	}
	if p, ok := dest[1].(*string); ok {
		*p = row.name
	}
	if p, ok := dest[2].(*int); ok {
		*p = row.priority
	}
	if p, ok := dest[3].(*int); ok {
		*p = row.response
	}
	if p, ok := dest[4].(*int); ok {
		*p = row.resolution
	}
	if p, ok := dest[5].(**int); ok {
		*p = row.update
	}
	return nil
}

func TestList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	up := 30
	db := &fakeDB{rows: []slapolicy{{"1", "P1", 1, 10, 20, &up}}}
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, db, nil, nil, nil)
	a.R.GET("/slas", authpkg.Middleware(a), List(a))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slas", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var out []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0]["name"].(string) != "P1" {
		t.Fatalf("unexpected output: %v", out)
	}
}
