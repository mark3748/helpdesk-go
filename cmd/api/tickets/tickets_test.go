package tickets

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
	metrics "github.com/mark3748/helpdesk-go/cmd/api/metrics"
)

func TestTicketHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, nil, nil, nil, nil)
	a.R.POST("/tickets", authpkg.Middleware(a), Create(a))
	a.R.GET("/tickets", authpkg.Middleware(a), List(a))
	a.R.GET("/tickets/:id", authpkg.Middleware(a), Get(a))
	a.R.PUT("/tickets/:id", authpkg.Middleware(a), Update(a))

	tests := []struct {
		name   string
		method string
		url    string
		body   string
		want   int
	}{
		{"list", http.MethodGet, "/tickets", "", http.StatusOK},
		{"create", http.MethodPost, "/tickets", `{"title":"abc","requester_id":"00000000-0000-0000-0000-000000000000","priority":1}`, http.StatusCreated},
		{"create_inline", http.MethodPost, "/tickets", `{"title":"abc","requester":{"email":"a@b.com"},"priority":1}`, http.StatusCreated},
		{"get", http.MethodGet, "/tickets/1", "", http.StatusOK},
		{"update", http.MethodPut, "/tickets/1", `{}`, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.url, strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			a.R.ServeHTTP(rr, req)
			if rr.Code != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, rr.Code)
			}
			if tt.name == "create" || tt.name == "create_inline" {
				var tk Ticket
				if err := json.Unmarshal(rr.Body.Bytes(), &tk); err != nil || tk.Title != "abc" {
					t.Fatalf("unexpected ticket: %v %v", tk, err)
				}
			}
		})
	}
}

// Test that create and update handlers increment their counters.
func TestTicketCounters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg := prometheus.NewRegistry()
	metrics.TicketsCreatedTotal = prometheus.NewCounter(prometheus.CounterOpts{Name: "tickets_created_total"})
	metrics.TicketsUpdatedTotal = prometheus.NewCounter(prometheus.CounterOpts{Name: "tickets_updated_total"})
	reg.MustRegister(metrics.TicketsCreatedTotal, metrics.TicketsUpdatedTotal)

	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, nil, nil, nil, nil)
	a.R.POST("/tickets", authpkg.Middleware(a), Create(a))
	a.R.PUT("/tickets/:id", authpkg.Middleware(a), Update(a))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tickets", strings.NewReader(`{"title":"abc","requester_id":"00000000-0000-0000-0000-000000000000","priority":1}`))
	req.Header.Set("Content-Type", "application/json")
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
	if v := testutil.ToFloat64(metrics.TicketsCreatedTotal); v != 1 {
		t.Fatalf("tickets_created_total = %v, want 1", v)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/tickets/1", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if v := testutil.ToFloat64(metrics.TicketsUpdatedTotal); v != 1 {
		t.Fatalf("tickets_updated_total = %v, want 1", v)
	}
}

type listRow struct {
	Ticket
	Updated time.Time
}

type listRows struct {
	data []listRow
	idx  int
}

func (r *listRows) Close()                                       {}
func (r *listRows) Err() error                                   { return nil }
func (r *listRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *listRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *listRows) Next() bool                                   { return r.idx < len(r.data) }
func (r *listRows) Values() ([]any, error)                       { return nil, nil }
func (r *listRows) RawValues() [][]byte                          { return nil }
func (r *listRows) Conn() *pgx.Conn                              { return nil }
func (r *listRows) Scan(dest ...any) error {
	row := r.data[r.idx]
	r.idx++
	*(dest[0].(*string)) = row.ID
	*(dest[1].(*any)) = row.Number
	*(dest[2].(*string)) = row.Title
	*(dest[3].(*string)) = row.Status
	*(dest[4].(**string)) = row.AssigneeID
	*(dest[5].(*int16)) = row.Priority
	*(dest[6].(*string)) = row.RequesterID
	*(dest[7].(*string)) = row.Requester
	*(dest[8].(*time.Time)) = row.Updated
	return nil
}

type listDB struct {
	rows []listRow
	sql  string
	args []any
}

func (db *listDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	db.sql = sql
	db.args = args
	return &listRows{data: db.rows}, nil
}
func (db *listDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row { return nil }
func (db *listDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func TestTicketListPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now().UTC()
	db := &listDB{rows: []listRow{
		{Ticket: Ticket{ID: "1", Title: "t1", Status: "Open", Priority: 1, RequesterID: "r1"}, Updated: now},
		{Ticket: Ticket{ID: "2", Title: "t2", Status: "Open", Priority: 1, RequesterID: "r2"}, Updated: now.Add(-time.Minute)},
	}}
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, db, nil, nil, nil)
	a.R.GET("/tickets", authpkg.Middleware(a), List(a))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tickets?limit=1", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp struct {
		Tickets    []Ticket `json:"tickets"`
		NextCursor string   `json:"next_cursor"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(resp.Tickets) != 1 || resp.Tickets[0].ID != "1" {
		t.Fatalf("unexpected tickets: %+v", resp.Tickets)
	}
	if resp.NextCursor == "" {
		t.Fatalf("expected next cursor")
	}
	// Cursor should reference the last returned ticket (rows[0])
	want := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s,%s", db.rows[0].Updated.Format(time.RFC3339Nano), db.rows[0].ID)))
	if resp.NextCursor != want {
		t.Fatalf("unexpected cursor %q want %q", resp.NextCursor, want)
	}
	if !strings.Contains(db.sql, "order by t.updated_at desc, t.id desc") {
		t.Fatalf("missing order clause: %s", db.sql)
	}
}

func TestTicketListFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &listDB{}
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, db, nil, nil, nil)
	a.R.GET("/tickets", authpkg.Middleware(a), List(a))
	rr := httptest.NewRecorder()
	url := "/tickets?status=New,Open&priority=1,2&team=t1,t2&assignee=a1,a2&requester=r1,r2&queue=q1,q2"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if strings.Contains(db.sql, "status <> 'Closed'") {
		t.Fatalf("unexpected closed filter: %s", db.sql)
	}
	if len(db.args) < 7 {
		t.Fatalf("expected args, got %v", db.args)
	}
	if !reflect.DeepEqual(db.args[0], []string{"New", "Open"}) {
		t.Fatalf("status args: %v", db.args[0])
	}
	if !reflect.DeepEqual(db.args[1], []int{1, 2}) {
		t.Fatalf("priority args: %v", db.args[1])
	}
	if !reflect.DeepEqual(db.args[2], []string{"t1", "t2"}) {
		t.Fatalf("team args: %v", db.args[2])
	}
	if !reflect.DeepEqual(db.args[3], []string{"a1", "a2"}) {
		t.Fatalf("assignee args: %v", db.args[3])
	}
	if !reflect.DeepEqual(db.args[4], []string{"r1", "r2"}) {
		t.Fatalf("requester args: %v", db.args[4])
	}
	if !reflect.DeepEqual(db.args[5], []string{"q1", "q2"}) {
		t.Fatalf("queue args: %v", db.args[5])
	}
}
