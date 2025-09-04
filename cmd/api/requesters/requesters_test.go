package requesters

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"strings"

	"github.com/gin-gonic/gin"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
)

func TestRequesterHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, nil, nil, nil, nil)
	a.R.POST("/requesters", authpkg.Middleware(a), Create(a))
	a.R.GET("/requesters/:id", authpkg.Middleware(a), Get(a))
	a.R.PATCH("/requesters/:id", authpkg.Middleware(a), Update(a))

	tests := []struct {
		name   string
		method string
		url    string
		body   string
		want   int
	}{
		{"create", http.MethodPost, "/requesters", `{"email":"a@b.com","name":"Ann","phone":"+1234567890"}`, http.StatusCreated},
		{"get", http.MethodGet, "/requesters/1", "", http.StatusOK},
		{"update", http.MethodPatch, "/requesters/1", `{"name":"Bob"}`, http.StatusOK},
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
		})
	}
}
