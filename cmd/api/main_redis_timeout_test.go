package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test that readyz respects REDIS_TIMEOUT_MS and returns quickly on ping timeouts.
func TestReadyz_RedisTimeoutSoftFail(t *testing.T) {
	t.Setenv("ENV", "test")
	t.Setenv("DB_TIMEOUT_MS", "0")
	t.Setenv("REDIS_TIMEOUT_MS", "20")
	app := NewApp(getConfig(), readyzDB{}, nil, nil, nil)
	// Override pingRedis to simulate a slow Redis.
	app.pingRedis = func(ctx context.Context) error {
		// Sleep longer than REDIS_TIMEOUT_MS so the derived context cancels first.
		select {
		case <-time.After(200 * time.Millisecond):
		case <-ctx.Done():
		}
		return ctx.Err()
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	start := time.Now()
	app.r.ServeHTTP(rr, req)
	dur := time.Since(start)
	if rr.Code == http.StatusOK {
		t.Fatalf("expected failure due to redis timeout, got %d", rr.Code)
	}
	if dur > 500*time.Millisecond {
		t.Fatalf("readyz took too long: %v", dur)
	}
}
