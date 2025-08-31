package main

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "math"
)

func TestHealthz(t *testing.T) {
    cfg := Config{Env: "test"}
    app := NewApp(cfg, nil, nil, nil, nil)

    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
    app.r.ServeHTTP(rr, req)

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
    cfg := Config{Env: "test", TestBypassAuth: true}
    app := NewApp(cfg, nil, nil, nil, nil)

    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/me", nil)
    app.r.ServeHTTP(rr, req)

    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rr.Code)
    }
    var user AuthUser
    if err := json.Unmarshal(rr.Body.Bytes(), &user); err != nil {
        t.Fatalf("invalid json: %v", err)
    }
    if user.ID == "" || user.Email == "" {
        t.Fatalf("expected synthetic user, got: %+v", user)
    }
    // Should include agent role by default for bypass
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
    cfg := Config{Env: "test", TestBypassAuth: false}
    app := NewApp(cfg, nil, nil, nil, nil)

    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/me", nil)
    app.r.ServeHTTP(rr, req)

    if rr.Code != http.StatusInternalServerError {
        t.Fatalf("expected 500 due to missing JWKS, got %d", rr.Code)
    }
}

func TestEnqueueEmail_JSONMarshalError(t *testing.T) {
    // Create a minimal app instance without Redis (enqueueEmail will return early if q is nil)
    app := &App{}
    
    // Test that the function handles marshal errors gracefully
    // Use data that cannot be marshaled to JSON (e.g., a function or channel)
    ctx := context.Background()
    unmarshalableData := map[string]interface{}{
        "invalid": func() {}, // functions cannot be marshaled to JSON
    }
    
    // This should not panic, even with unmarshalable data and nil Redis client
    app.enqueueEmail(ctx, "test@example.com", "test_template", unmarshalableData)
}

func TestEnqueueEmail_InfinityError(t *testing.T) {
    // Another test with data that can't be marshaled (Infinity/NaN)
    app := &App{}
    ctx := context.Background()
    
    unmarshalableData := map[string]interface{}{
        "infinity": math.Inf(1),
        "nan":      math.NaN(),
    }
    
    // This should not panic and should handle the marshal error gracefully
    app.enqueueEmail(ctx, "test@example.com", "test_template", unmarshalableData)
}

