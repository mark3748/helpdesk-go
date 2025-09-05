package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
	metrics "github.com/mark3748/helpdesk-go/cmd/api/metrics"
)

func TestMiddlewarePopulatesUserFromClaims(t *testing.T) {
	cfg := apppkg.Config{Env: "test", OIDCGroupClaim: "roles"}
	key := []byte("secret")
	keyf := func(t *jwt.Token) (any, error) { return key, nil }

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "user-123",
		"email": "user@example.com",
		"name":  "User Name",
		"roles": []string{"agent", "manager"},
	})
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	a := apppkg.NewApp(cfg, nil, keyf, nil, nil)
	a.R.GET("/me", authpkg.Middleware(a), authpkg.Me)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	a.R.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var u authpkg.AuthUser
	if err := json.Unmarshal(rr.Body.Bytes(), &u); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if u.Email != "user@example.com" || u.DisplayName != "User Name" {
		t.Fatalf("unexpected user: %+v", u)
	}
	if len(u.Roles) != 2 || u.Roles[0] != "agent" || u.Roles[1] != "manager" {
		t.Fatalf("roles not populated: %+v", u.Roles)
	}
}

// Test that unauthorized requests increment the auth failure counter.
func TestAuthFailureCounter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg := prometheus.NewRegistry()
	metrics.AuthFailuresTotal = prometheus.NewCounter(prometheus.CounterOpts{Name: "auth_failures_total"})
	reg.MustRegister(metrics.AuthFailuresTotal)

	cfg := apppkg.Config{Env: "test", AuthMode: "local", AuthLocalSecret: "secret"}
	a := apppkg.NewApp(cfg, nil, nil, nil, nil)
	a.R.GET("/protected", authpkg.Middleware(a), func(c *gin.Context) { c.Status(http.StatusOK) })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	a.R.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if v := testutil.ToFloat64(metrics.AuthFailuresTotal); v != 1 {
		t.Fatalf("auth_failures_total = %v, want 1", v)
	}
}
