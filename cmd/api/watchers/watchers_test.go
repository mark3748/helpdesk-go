package watchers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
)

func TestWatcherHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, nil, nil, nil, nil)
	a.R.GET("/tickets/:id/watchers", authpkg.Middleware(a), List(a))
	a.R.POST("/tickets/:id/watchers", authpkg.Middleware(a), Add(a))
	a.R.DELETE("/tickets/:id/watchers/:uid", authpkg.Middleware(a), Remove(a))

	tests := []struct {
		name   string
		method string
		url    string
		want   int
	}{
		{"list", http.MethodGet, "/tickets/1/watchers", http.StatusOK},
		{"add", http.MethodPost, "/tickets/1/watchers", http.StatusCreated},
		{"remove", http.MethodDelete, "/tickets/1/watchers/1", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.url, nil)
			a.R.ServeHTTP(rr, req)
			if rr.Code != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, rr.Code)
			}
		})
	}
}
