package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
	metrics "github.com/mark3748/helpdesk-go/cmd/api/metrics"
)

func TestMetricsHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, nil, nil, nil, nil)
	a.R.GET("/metrics/sla", authpkg.Middleware(a), metrics.SLA(a))
	a.R.GET("/metrics/resolution", authpkg.Middleware(a), metrics.Resolution(a))
	a.R.GET("/metrics/ticket_volume", authpkg.Middleware(a), metrics.TicketVolume(a))

	tests := []struct {
		name string
		url  string
	}{
		{"sla", "/metrics/sla"},
		{"resolution", "/metrics/resolution"},
		{"volume", "/metrics/ticket_volume"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			a.R.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rr.Code)
			}
		})
	}
}
