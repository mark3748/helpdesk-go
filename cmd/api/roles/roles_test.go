package roles

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	"github.com/mark3748/helpdesk-go/cmd/api/testutil"
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
						if !strings.Contains(sql, "roles") {
							return nil, errors.New("unexpected sql")
						}
						idx := 0
						roles := []string{"admin", "agent"}
						return &testutil.MockRows{
							NextFunc: func() bool {
								return idx < len(roles)
							},
							ScanFunc: func(dest ...interface{}) error {
								*(dest[0].(*string)) = roles[idx]
								idx++
								return nil
							},
						}, nil
					},
				}
			},
			wantStatus: http.StatusOK,
			wantBody:   `["admin","agent"]`,
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
			name: "scan error",
			setupDB: func() *testutil.MockDB {
				return &testutil.MockDB{
					QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
						return &testutil.MockRows{
							NextFunc: func() bool { return true },
							ScanFunc: func(dest ...interface{}) error {
								return errors.New("scan fail")
							},
						}, nil
					},
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":"scan fail"}`,
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
			cfg := apppkg.Config{Env: "test"}
			a := apppkg.NewApp(cfg, db, nil, nil, nil)
			a.R.GET("/roles", List(a))

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/roles", nil)
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
