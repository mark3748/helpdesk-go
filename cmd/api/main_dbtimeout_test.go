package main

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
)

// slowDB simulates a DB that blocks longer than the configured timeout.
type slowDB struct{ d time.Duration }

type slowRow struct {
    ctx context.Context
    d   time.Duration
}

func (r slowRow) Scan(dest ...any) error {
    select {
    case <-time.After(r.d):
    case <-r.ctx.Done():
    }
    if err := r.ctx.Err(); err != nil {
        return err
    }
    // if not timed out, set first int if present
    for _, v := range dest {
        if p, ok := v.(*int); ok {
            *p = 1
            break
        }
    }
    return nil
}

func (s slowDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
    select {
    case <-time.After(s.d):
    case <-ctx.Done():
    }
    if err := ctx.Err(); err != nil {
        return nil, err
    }
    return nil, context.DeadlineExceeded
}
func (s slowDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
    return slowRow{ctx: ctx, d: s.d}
}
func (s slowDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
    select {
    case <-time.After(s.d):
    case <-ctx.Done():
    }
    if err := ctx.Err(); err != nil {
        return pgconn.CommandTag{}, err
    }
    return pgconn.CommandTag{}, context.DeadlineExceeded
}

func TestDBTimeout_Readyz(t *testing.T) {
    t.Setenv("ENV", "test")
    // Tight DB timeout to force cancellation
    t.Setenv("DB_TIMEOUT_MS", "5")
    app := NewApp(getConfig(), slowDB{d: 50 * time.Millisecond}, nil, nil, nil)

    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
    app.r.ServeHTTP(rr, req)

    if rr.Code == http.StatusOK {
        t.Fatalf("expected non-200 due to DB timeout, got %d", rr.Code)
    }
    var body map[string]any
    _ = json.Unmarshal(rr.Body.Bytes(), &body)
    if body["error"] != "db" {
        t.Fatalf("expected error=db, got %v", body)
    }
}

