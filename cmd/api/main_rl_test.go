package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	metrics "github.com/mark3748/helpdesk-go/cmd/api/metrics"
	rateln "github.com/mark3748/helpdesk-go/internal/ratelimit"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
)

// Test that rlMiddleware rejects over-limit requests and increments the metric.
func TestRLMiddleware_IncrementsCounter(t *testing.T) {
	t.Setenv("ENV", "test")
	reg := prometheus.NewRegistry()
	metrics.RateLimitRejectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_rejections_total",
			Help: "Number of requests rejected by rate limiting.",
		},
		[]string{"route"},
	)
	reg.MustRegister(metrics.RateLimitRejectionsTotal)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	limiter := rateln.New(rdb, 1, time.Minute, "test:")

	app := &App{cfg: Config{Env: "test"}, r: gin.New()}
	app.r.GET("/limited", app.rlMiddleware(limiter, func(c *gin.Context) string { return c.ClientIP() }, "test"), func(c *gin.Context) {
		c.String(200, "ok")
	})

	// First request should pass
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	app.r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// Second request should be rate limited and increment the counter
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/limited", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	app.r.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}

	// Validate the counter incremented for the route label
	got := testutil.ToFloat64(metrics.RateLimitRejectionsTotal.WithLabelValues("test"))
	if got < 1 {
		t.Fatalf("expected rate_limit_rejections_total >= 1, got %v", got)
	}
}
