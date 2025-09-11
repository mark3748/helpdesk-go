package kb

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

type fakeDB struct{ rows []article }

type article struct {
	id, slug, title, body string
}

func (db *fakeDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return &fakeRows{rows: db.rows}, nil
}
func (db *fakeDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	switch {
	case strings.HasPrefix(sql, "insert"):
		a := article{"1", args[0].(string), args[1].(string), args[2].(string)}
		db.rows = append(db.rows, a)
		return &fakeRow{a: a}
	case strings.HasPrefix(sql, "select"):
		slug := args[0].(string)
		for _, r := range db.rows {
			if r.slug == slug {
				return &fakeRow{a: r}
			}
		}
		return &fakeRow{err: pgx.ErrNoRows}
	case strings.HasPrefix(sql, "update"):
		slug := args[3].(string)
		for i, r := range db.rows {
			if r.slug == slug {
				db.rows[i].slug = args[0].(string)
				db.rows[i].title = args[1].(string)
				db.rows[i].body = args[2].(string)
				return &fakeRow{a: db.rows[i]}
			}
		}
		return &fakeRow{err: pgx.ErrNoRows}
	}
	return &fakeRow{err: pgx.ErrNoRows}
}
func (db *fakeDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if strings.HasPrefix(sql, "delete") {
		slug := args[0].(string)
		for i, r := range db.rows {
			if r.slug == slug {
				db.rows = append(db.rows[:i], db.rows[i+1:]...)
				return pgconn.CommandTag{}, nil
			}
		}
		return pgconn.CommandTag{}, pgx.ErrNoRows
	}
	return pgconn.CommandTag{}, nil
}
func (db *fakeDB) Begin(context.Context) (pgx.Tx, error) { return nil, nil }

type fakeRow struct {
	a   article
	err error
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if p, ok := dest[0].(*string); ok {
		*p = r.a.id
	}
	if len(dest) > 1 {
		if p, ok := dest[1].(*string); ok {
			*p = r.a.slug
		}
	}
	if len(dest) > 2 {
		if p, ok := dest[2].(*string); ok {
			*p = r.a.title
		}
	}
	if len(dest) > 3 {
		if p, ok := dest[3].(*string); ok {
			*p = r.a.body
		}
	}
	return nil
}

type fakeRows struct {
	rows []article
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
		*p = row.slug
	}
	if p, ok := dest[2].(*string); ok {
		*p = row.title
	}
	if p, ok := dest[3].(*string); ok {
		*p = row.body
	}
	return nil
}

func TestSearch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &fakeDB{rows: []article{{"1", "slug", "Title", "Body"}}}
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, db, nil, nil, nil)
	a.R.GET("/kb", authpkg.Middleware(a), Search(a))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/kb?q=x", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var out []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0]["slug"].(string) != "slug" {
		t.Fatalf("unexpected output: %v", out)
	}
}

func TestCRUD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &fakeDB{}
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, db, nil, nil, nil)
	a.R.POST("/kb", authpkg.Middleware(a), Create(a))
	a.R.GET("/kb/:slug", authpkg.Middleware(a), Get(a))
	a.R.PUT("/kb/:slug", authpkg.Middleware(a), Update(a))
	a.R.DELETE("/kb/:slug", authpkg.Middleware(a), Delete(a))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/kb", strings.NewReader(`{"slug":"s1","title":"T1","body_md":"B1"}`))
	req.Header.Set("Content-Type", "application/json")
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/kb/s1", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 get, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/kb/s1", strings.NewReader(`{"slug":"s1","title":"T2","body_md":"B2"}`))
	req.Header.Set("Content-Type", "application/json")
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 update, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/kb/s1", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 delete, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/kb/s1", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 get after delete, got %d", rr.Code)
	}
}
