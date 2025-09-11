package webhooks

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
)

type fakeDBInbound struct {
	sql  string
	args []any
}

func (db *fakeDBInbound) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (db *fakeDBInbound) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (db *fakeDBInbound) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	db.sql = sql
	db.args = args
	return pgconn.CommandTag{}, nil
}
func (db *fakeDBInbound) Begin(context.Context) (pgx.Tx, error) { return nil, nil }

func TestEmailInbound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &fakeDBInbound{}
	cfg := apppkg.Config{Env: "test"}
	a := apppkg.NewApp(cfg, db, nil, nil, nil)
	a.R.POST("/webhooks/email-inbound", EmailInbound(a))

	body := bytes.NewBufferString(`{"raw_store_key":"k","parsed_json":{},"message_id":"m"}`)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/email-inbound", body)
	req.Header.Set("Content-Type", "application/json")
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}
	if db.sql == "" || len(db.args) != 3 {
		t.Fatalf("exec not called properly")
	}
	if db.args[0].(string) != "k" {
		t.Fatalf("unexpected arg: %v", db.args)
	}
}
