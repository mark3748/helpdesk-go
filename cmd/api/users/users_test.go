package users

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
	"github.com/mark3748/helpdesk-go/cmd/api/testutil"
)

func TestList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		query      string
		setupDB    func() *testutil.MockDB
		wantStatus int
		wantLen    int
	}{
		{
			name: "success",
			setupDB: func() *testutil.MockDB {
				return &testutil.MockDB{
					QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
						if !strings.Contains(sql, "from users u") {
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
								*(dest[0].(*string)) = "u1"
								*(dest[1].(*string)) = "ext1"
								*(dest[2].(*string)) = "user1"
								*(dest[3].(*string)) = "u1@example.com"
								*(dest[4].(*string)) = "User One"
								*(dest[5].(*string)) = "admin,agent"
								return nil
							},
						}, nil
					},
				}
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
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
			a.R.GET("/users", authpkg.Middleware(a), List(a))

			rr := httptest.NewRecorder()
			url := "/users"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			a.R.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("List() status = %v, want %v", rr.Code, tt.wantStatus)
			}
			if rr.Code == http.StatusOK {
				var out []interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
					t.Fatal(err)
				}
				if len(out) != tt.wantLen {
					t.Errorf("List() len = %v, want %v", len(out), tt.wantLen)
				}
			}
		})
	}
}

func TestGet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		setupDB    func() *testutil.MockDB
		wantStatus int
		wantID     string
	}{
		{
			name: "success",
			setupDB: func() *testutil.MockDB {
				return &testutil.MockDB{
					QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
						return &testutil.MockRow{
							ScanFunc: func(dest ...interface{}) error {
								*(dest[0].(*string)) = "u1"
								*(dest[1].(*string)) = "ext1"
								*(dest[2].(*string)) = "user1"
								*(dest[3].(*string)) = "u1@example.com"
								*(dest[4].(*string)) = "User One"
								return nil
							},
						}
					},
				}
			},
			wantStatus: http.StatusOK,
			wantID:     "u1",
		},
		{
			name: "not found",
			setupDB: func() *testutil.MockDB {
				return &testutil.MockDB{
					QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
						return &testutil.MockRow{
							ScanFunc: func(dest ...interface{}) error {
								return pgx.ErrNoRows
							},
						}
					},
				}
			},
			wantStatus: http.StatusNotFound,
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
			a.R.GET("/users/:id", authpkg.Middleware(a), Get(a))

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/users/u1", nil)
			a.R.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("Get() status = %v, want %v", rr.Code, tt.wantStatus)
			}
			if tt.wantID != "" {
				var out struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
					t.Fatal(err)
				}
				if out.ID != tt.wantID {
					t.Errorf("Get() id = %v, want %v", out.ID, tt.wantID)
				}
			}
		})
	}
}

func TestCreateLocal(t *testing.T) {
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
			body: `{"username":"newuser", "email":"new@example.com", "display_name":"New User", "password":"password"}`,
			setupDB: func() *testutil.MockDB {
				return &testutil.MockDB{
					QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
						return &testutil.MockRow{
							ScanFunc: func(dest ...interface{}) error {
								*(dest[0].(*string)) = "u-new"
								return nil
							},
						}
					},
				}
			},
			wantStatus: http.StatusCreated,
			wantID:     "u-new",
		},
		{
			name:       "invalid json",
			body:       `{`,
			setupDB:    func() *testutil.MockDB { return &testutil.MockDB{} },
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing password",
			body:       `{"username":"newuser"}`,
			setupDB:    func() *testutil.MockDB { return &testutil.MockDB{} },
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "db error",
			body: `{"username":"newuser", "password":"password"}`,
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
			a.R.POST("/users", authpkg.Middleware(a), CreateLocal(a))

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			a.R.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("CreateLocal() status = %v, want %v", rr.Code, tt.wantStatus)
			}
			if tt.wantID != "" {
				var out struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
					t.Fatal(err)
				}
				if out.ID != tt.wantID {
					t.Errorf("CreateLocal() id = %v, want %v", out.ID, tt.wantID)
				}
			}
		})
	}
}

func TestGetProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		setupDB    func() *testutil.MockDB
		wantStatus int
		wantUser   string
	}{
		{
			name: "success",
			setupDB: func() *testutil.MockDB {
				return &testutil.MockDB{
					QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
						return &testutil.MockRow{
							ScanFunc: func(dest ...interface{}) error {
								*(dest[0].(*string)) = "u-test"
								*(dest[1].(*string)) = "testuser"
								*(dest[2].(*string)) = "test@example.com"
								*(dest[3].(*string)) = "Test User"
								return nil
							},
						}
					},
				}
			},
			wantStatus: http.StatusOK,
			wantUser:   "testuser",
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
			a.R.GET("/me/profile", authpkg.Middleware(a), GetProfile(a))

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/me/profile", nil)
			a.R.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("GetProfile() status = %v, want %v", rr.Code, tt.wantStatus)
			}
			if tt.wantUser != "" {
				var out struct {
					Username string `json:"username"`
				}
				if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
					t.Fatal(err)
				}
				if out.Username != tt.wantUser {
					t.Errorf("GetProfile() username = %v, want %v", out.Username, tt.wantUser)
				}
			}
		})
	}
}
