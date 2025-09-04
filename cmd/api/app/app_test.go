package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// Test that the RequestID middleware sets a header and context value.
func TestRequestID(t *testing.T) {
	cfg := Config{Env: "test"}
	a := NewApp(cfg, nil, nil, nil, nil)
	a.R.GET("/ping", func(c *gin.Context) {
		id, _ := c.Get("request_id")
		if id == "" {
			t.Errorf("missing request_id in context")
		}
		c.JSON(200, gin.H{"ok": true})
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Header().Get("X-Request-ID") == "" {
		t.Fatalf("expected X-Request-ID header")
	}
}

// Test that the rate limiter blocks excessive requests.
func TestRateLimit(t *testing.T) {
	cfg := Config{Env: "test", RateLimitRPS: 1, RateLimitBurst: 1}
	a := NewApp(cfg, nil, nil, nil, nil)
	a.R.GET("/", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}
}

// Test that the rate limiter is disabled when no configuration is provided.
func TestRateLimitDisabledByDefault(t *testing.T) {
	cfg := Config{Env: "test"}
	a := NewApp(cfg, nil, nil, nil, nil)
	a.R.GET("/", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
