package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestHealthz(t *testing.T) {
	cfg := Config{Env: "test"}
	app := NewApp(cfg, nil, nil, nil, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	app.r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if ok, _ := body["ok"].(bool); !ok {
		t.Fatalf("expected ok=true in body, got: %v", body)
	}
}

func TestMe_BypassAuth(t *testing.T) {
	cfg := Config{Env: "test", TestBypassAuth: true}
	app := NewApp(cfg, nil, nil, nil, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	app.r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var user AuthUser
	if err := json.Unmarshal(rr.Body.Bytes(), &user); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if user.ID == "" || user.Email == "" {
		t.Fatalf("expected synthetic user, got: %+v", user)
	}
	// Should include agent role by default for bypass
	found := false
	for _, r := range user.Roles {
		if r == "agent" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected role 'agent' in %+v", user.Roles)
	}
}

func TestMe_NoBypass_NoJWKS(t *testing.T) {
	cfg := Config{Env: "test", TestBypassAuth: false}
	app := NewApp(cfg, nil, nil, nil, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	app.r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 due to missing JWKS, got %d", rr.Code)
	}
}
func TestEnqueueEmail_JSONMarshalError(t *testing.T) {
	// Create a minimal app instance without Redis (enqueueEmail will return early if q is nil)
	app := &App{}

	// Test that the function handles marshal errors gracefully
	// Use data that cannot be marshaled to JSON (e.g., a function or channel)
	ctx := context.Background()
	unmarshalableData := map[string]interface{}{
		"invalid": func() {}, // functions cannot be marshaled to JSON
	}

	// This should not panic, even with unmarshalable data and nil Redis client
	app.enqueueEmail(ctx, "test@example.com", "test_template", unmarshalableData)
}

func TestEnqueueEmail_InfinityError(t *testing.T) {
	// Another test with data that can't be marshaled (Infinity/NaN)
	app := &App{}
	ctx := context.Background()

	unmarshalableData := map[string]interface{}{
		"infinity": math.Inf(1),
		"nan":      math.NaN(),
	}

	// This should not panic and should handle the marshal error gracefully
	app.enqueueEmail(ctx, "test@example.com", "test_template", unmarshalableData)
}

type csatTestDB struct {
	lastSQL  string
	lastArgs []any
	rows     int64
}

func (db *csatTestDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return &fakeRows{}, nil
}

func (db *csatTestDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return nil
}

func (db *csatTestDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	db.lastSQL = sql
	db.lastArgs = args
	return pgconn.NewCommandTag(fmt.Sprintf("UPDATE %d", db.rows)), nil
}

func TestSubmitCSAT(t *testing.T) {
	db := &csatTestDB{rows: 1}
	cfg := Config{Env: "test"}
	app := NewApp(cfg, db, nil, nil, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/csat/token123?score=good", nil)
	app.r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if db.lastSQL == "" || len(db.lastArgs) != 2 {
		t.Fatalf("exec not called properly: %s %v", db.lastSQL, db.lastArgs)
	}
	if db.lastArgs[0] != "good" || db.lastArgs[1] != "token123" {
		t.Fatalf("unexpected args: %v", db.lastArgs)
	}
}

type recordDB struct {
	sql  string
	args []any
}

func (db *recordDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	db.sql = sql
	db.args = args
	return &fakeRows{}, nil
}

func (db *recordDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return nil
}

func (db *recordDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func TestListTickets(t *testing.T) {
	cases := []struct {
		name         string
		url          string
		wantSQLParts []string
		wantArgs     []any
	}{
		{
			name: "filtering and search",
			url:  "/tickets?status=open&priority=2&team=team1&assignee=user1&search=foo+++bar",
			wantSQLParts: []string{
				"t.status = $1",
				"t.priority = $2",
				"t.team_id = $3",
				"t.assignee_id = $4",
				"websearch_to_tsquery('english', $5)",
			},
			wantArgs: []any{"open", 2, "team1", "user1", "foo   bar"},
		},
		{
			name:         "search only",
			url:          "/tickets?search=hello+++world",
			wantSQLParts: []string{"websearch_to_tsquery('english', $1)"},
			wantArgs:     []any{"hello   world"},
		},
		{
			name:         "filters only",
			url:          "/tickets?status=open&priority=1",
			wantSQLParts: []string{"t.status = $1", "t.priority = $2"},
			wantArgs:     []any{"open", 1},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := &recordDB{}
			cfg := Config{Env: "test", TestBypassAuth: true}
			app := NewApp(cfg, db, nil, nil, nil)

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			app.r.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rr.Code)
			}
			for _, part := range tc.wantSQLParts {
				if !strings.Contains(db.sql, part) {
					t.Fatalf("missing sql part %q in %s", part, db.sql)
				}
			}
			if len(db.args) != len(tc.wantArgs) {
				t.Fatalf("expected %d args, got %d", len(tc.wantArgs), len(db.args))
			}
			for i, v := range tc.wantArgs {
				if db.args[i] != v {
					t.Fatalf("arg %d = %#v, want %#v", i, db.args[i], v)
				}
			}
		})
	}
}
