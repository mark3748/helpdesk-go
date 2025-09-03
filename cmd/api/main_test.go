package main_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
	ticketspkg "github.com/mark3748/helpdesk-go/cmd/api/tickets"
)

func TestHealthz(t *testing.T) {
	cfg := apppkg.Config{Env: "test"}
	a := apppkg.NewApp(cfg, nil, nil, nil, nil)
	a.R.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	a.R.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if ok, _ := body["ok"].(bool); !ok {
		t.Fatalf("expected ok=true in body, got: %v", body)
	}
}

func TestMe_BypassAuth(t *testing.T) {
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, nil, nil, nil, nil)
	a.R.GET("/me", authpkg.Middleware(a), authpkg.Me)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	a.R.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var user authpkg.AuthUser
	if err := json.Unmarshal(rr.Body.Bytes(), &user); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if user.ID == "" || user.Email == "" {
		t.Fatalf("expected synthetic user, got: %+v", user)
	}
	found := false
	for _, r := range user.Roles {
		if r == "agent" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected role 'agent' in %+v", user.Roles)
	}
}

func TestMe_NoBypass_NoJWKS(t *testing.T) {
	cfg := apppkg.Config{Env: "test", TestBypassAuth: false}
	a := apppkg.NewApp(cfg, nil, nil, nil, nil)
	a.R.GET("/me", authpkg.Middleware(a), authpkg.Me)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	a.R.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 due to missing JWKS, got %d", rr.Code)
	}
}

func TestRequireRoleMultiple(t *testing.T) {
	handler := authpkg.RequireRole("agent", "manager")

	t.Run("allowed role", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user", authpkg.AuthUser{Roles: []string{"manager"}})
		handler(c)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("forbidden role", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user", authpkg.AuthUser{Roles: []string{"user"}})
		handler(c)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})
}

func TestCreateTicketValidationErrors(t *testing.T) {
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true}
	a := apppkg.NewApp(cfg, nil, nil, nil, nil)
	a.R.POST("/tickets", authpkg.Middleware(a), ticketspkg.Create(a))

	t.Run("invalid urgency", func(t *testing.T) {
		rr := httptest.NewRecorder()
		body := `{"title":"abc","requester_id":"00000000-0000-0000-0000-000000000000","priority":1,"urgency":5,"custom_json":{}}`
		req := httptest.NewRequest(http.MethodPost, "/tickets", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		a.R.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
		var resp struct {
			Errors map[string]string `json:"errors"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("invalid json: %v", err)
		}
		if resp.Errors["urgency"] == "" {
			t.Fatalf("expected urgency error, got %v", resp.Errors)
		}
	})

	t.Run("invalid custom_json", func(t *testing.T) {
		rr := httptest.NewRecorder()
		body := `{"title":"abc","requester_id":"00000000-0000-0000-0000-000000000000","priority":1,"custom_json":[]}`
		req := httptest.NewRequest(http.MethodPost, "/tickets", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		a.R.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
		var resp struct {
			Errors map[string]string `json:"errors"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("invalid json: %v", err)
		}
		if resp.Errors["custom_json"] == "" {
			t.Fatalf("expected custom_json error, got %v", resp.Errors)
		}
	})
}
