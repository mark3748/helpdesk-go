package emails

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
)

type fakeDB struct{}

func (db *fakeDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return &fakeRows{}, nil
}
func (db *fakeDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return fakeRow{}
}
func (db *fakeDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (db *fakeDB) Begin(ctx context.Context) (pgx.Tx, error) { return nil, nil }

type fakeRows struct{ done bool }

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return nil }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) RawValues() [][]byte                          { return nil }
func (r *fakeRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeRows) Conn() *pgx.Conn                              { return nil }
func (r *fakeRows) Next() bool {
	if r.done {
		return false
	}
	r.done = true
	return true
}
func (r *fakeRows) Scan(dest ...any) error {
	*(dest[0].(*string)) = "1"
	*(dest[1].(*string)) = "to@example.com"
	*(dest[2].(*string)) = "subject"
	*(dest[3].(*string)) = "sent"
	*(dest[4].(*int)) = 1
	if p, ok := dest[5].(**string); ok {
		*p = nil
	}
	*(dest[6].(*time.Time)) = time.Unix(0, 0)
	return nil
}

type fakeRow struct{}

func (fakeRow) Scan(dest ...any) error { return pgx.ErrNoRows }

func TestListOutbound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &fakeDB{}
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, db, nil, nil, nil)
	a.R.GET("/emails/outbound", authpkg.Middleware(a), ListOutbound(a))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/emails/outbound", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var out []Outbound
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Retries != 1 {
		t.Fatalf("unexpected response %v", out)
	}
}
