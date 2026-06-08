package comments

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
	"github.com/mark3748/helpdesk-go/cmd/api/testutil"
	"github.com/redis/go-redis/v9"
)

func TestList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		setupDB    func() *testutil.MockDB
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			setupDB: func() *testutil.MockDB {
				return &testutil.MockDB{
					QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
						if !strings.Contains(sql, "ticket_comments") {
							return nil, errors.New("unexpected sql")
						}
						idx := 0
						return &testutil.MockRows{
							NextFunc: func() bool {
								if idx == 0 {
									idx++
									return true
								}
								return false
							},
							ScanFunc: func(dest ...interface{}) error {
								*(dest[0].(*string)) = "c1"
								*(dest[1].(*string)) = "body"
								return nil
							},
						}, nil
					},
				}
			},
			wantStatus: http.StatusOK,
			wantBody:   `[{"id":"c1","body_md":"body"}]`,
		},
		{
			name: "db error",
			setupDB: func() *testutil.MockDB {
				return &testutil.MockDB{
					QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
						return nil, errors.New("db error")
					},
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":"db error"}`,
		},
		{
			name: "nil db",
			setupDB: func() *testutil.MockDB {
				return nil
			},
			wantStatus: http.StatusOK,
			wantBody:   `[]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var db apppkg.DB
			if tt.setupDB != nil {
				mdb := tt.setupDB()
				if mdb != nil {
					db = mdb
				}
			}

			cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
			a := apppkg.NewApp(cfg, db, nil, nil, nil)
			a.R.GET("/tickets/:id/comments", authpkg.Middleware(a), List(a))

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/tickets/1/comments", nil)
			a.R.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("List() status = %v, want %v", rr.Code, tt.wantStatus)
			}
			if tt.wantBody != "" {
				if strings.TrimSpace(rr.Body.String()) != tt.wantBody {
					t.Errorf("List() body = %v, want %v", rr.Body.String(), tt.wantBody)
				}
			}
		})
	}
}

func TestAdd(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		body       string
		setupDB    func() *testutil.MockDB
		wantStatus int
		wantID     string
	}{
		{
			name: "success",
			body: `{"body_md": "new comment"}`,
			setupDB: func() *testutil.MockDB {
				return &testutil.MockDB{
					QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
						return &testutil.MockRow{
							ScanFunc: func(dest ...interface{}) error {
								*(dest[0].(*string)) = "c-new"
								return nil
							},
						}
					},
					ExecFunc: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
						return pgconn.CommandTag{}, nil
					},
				}
			},
			wantStatus: http.StatusCreated,
			wantID:     "c-new",
		},
		{
			name:       "invalid json",
			body:       `{`,
			setupDB:    func() *testutil.MockDB { return &testutil.MockDB{} },
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty body",
			body:       `{"body_md": ""}`,
			setupDB:    func() *testutil.MockDB { return &testutil.MockDB{} },
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "db error",
			body: `{"body_md": "new comment"}`,
			setupDB: func() *testutil.MockDB {
				return &testutil.MockDB{
					QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
						return &testutil.MockRow{
							ScanFunc: func(dest ...interface{}) error {
								return errors.New("db error")
							},
						}
					},
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "nil db",
			body: `{"body_md": "new comment"}`,
			setupDB: func() *testutil.MockDB {
				return nil
			},
			wantStatus: http.StatusCreated,
			wantID:     "temp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var db apppkg.DB
			if tt.setupDB != nil {
				mdb := tt.setupDB()
				if mdb != nil {
					db = mdb
				}
			}
			cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
			a := apppkg.NewApp(cfg, db, nil, nil, nil)
			a.R.POST("/tickets/:id/comments", authpkg.Middleware(a), Add(a))

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/tickets/1/comments", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			a.R.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("Add() status = %v, want %v", rr.Code, tt.wantStatus)
			}
			if tt.wantID != "" {
				var resp struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Fatal(err)
				}
				if resp.ID != tt.wantID {
					t.Errorf("Add() id = %v, want %v", resp.ID, tt.wantID)
				}
			}
		})
	}
}

func TestAdd_EnqueuesDiscordOutgoingCommentJob(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := &testutil.MockDB{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return &testutil.MockRow{
				ScanFunc: func(dest ...interface{}) error {
					*(dest[0].(*string)) = "c-new"
					return nil
				},
			}
		},
		ExecFunc: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, nil
		},
	}

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rdb.Close()
	})

	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, db, nil, nil, rdb)
	a.R.POST("/tickets/:id/comments", authpkg.Middleware(a), Add(a))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tickets/ticket-1/comments", strings.NewReader(`{"body_md":"new comment"}`))
	req.Header.Set("Content-Type", "application/json")
	a.R.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Add() status = %v, want %v", rr.Code, http.StatusCreated)
	}

	items, err := rdb.LRange(context.Background(), "jobs", 0, -1).Result()
	if err != nil {
		t.Fatalf("LRange jobs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("jobs len = %d, want 1", len(items))
	}

	var job struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(items[0]), &job); err != nil {
		t.Fatalf("unmarshal job: %v", err)
	}
	if job.Type != "discord_outgoing_comment" {
		t.Fatalf("job type = %q, want discord_outgoing_comment", job.Type)
	}

	var payload struct {
		TicketID string `json:"ticket_id"`
		BodyMD   string `json:"body_md"`
	}
	if err := json.Unmarshal(job.Data, &payload); err != nil {
		t.Fatalf("unmarshal job payload: %v", err)
	}
	if payload.TicketID != "ticket-1" {
		t.Fatalf("payload.ticket_id = %q, want ticket-1", payload.TicketID)
	}
	if payload.BodyMD != "new comment" {
		t.Fatalf("payload.body_md = %q, want new comment", payload.BodyMD)
	}
}
