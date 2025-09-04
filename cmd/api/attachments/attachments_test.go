package attachments

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
)

func TestAttachmentHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, nil, nil, nil, nil)
	a.R.GET("/tickets/:id/attachments", authpkg.Middleware(a), List(a))
	a.R.POST("/tickets/:id/attachments", authpkg.Middleware(a), Upload(a))
	a.R.GET("/tickets/:id/attachments/:att", authpkg.Middleware(a), Get(a))
	a.R.DELETE("/tickets/:id/attachments/:att", authpkg.Middleware(a), Delete(a))
	a.R.POST("/tickets/:id/attachments/presign-upload", authpkg.Middleware(a), PresignUpload(a))
	a.R.GET("/tickets/:id/attachments/:att/presign-download", authpkg.Middleware(a), PresignDownload(a))

	tests := []struct {
		name   string
		method string
		url    string
		want   int
	}{
		{"list", http.MethodGet, "/tickets/1/attachments", http.StatusOK},
		{"upload", http.MethodPost, "/tickets/1/attachments", http.StatusCreated},
		{"get", http.MethodGet, "/tickets/1/attachments/1", http.StatusOK},
		{"delete", http.MethodDelete, "/tickets/1/attachments/1", http.StatusOK},
		{"presign upload", http.MethodPost, "/tickets/1/attachments/presign-upload", http.StatusOK},
		{"presign download", http.MethodGet, "/tickets/1/attachments/1/presign-download", http.StatusOK},
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
