package tickets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
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
			if tt.name == "create" {
				var tk Ticket
				if err := json.Unmarshal(rr.Body.Bytes(), &tk); err != nil || tk.Title != "abc" {
					t.Fatalf("unexpected ticket: %v %v", tk, err)
				}
			}
		})
	}
}
