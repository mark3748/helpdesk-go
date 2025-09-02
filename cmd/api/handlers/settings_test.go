package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSettingsHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	InitSettings("/tmp/logs")
	r := gin.New()
	r.GET("/settings", GetSettings)
	r.POST("/settings/storage", SaveStorageSettings)
	r.POST("/test-connection", TestConnection)

	// initial log path
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/settings", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var resp Settings
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.LogPath != "/tmp/logs" {
		t.Fatalf("unexpected log path %s", resp.LogPath)
	}

	// save storage config
	body := bytes.NewBufferString(`{"endpoint":"s3","bucket":"b"}`)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/settings/storage", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}

	// ensure persisted
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/settings", nil)
	r.ServeHTTP(w, req)
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Storage["endpoint"] != "s3" {
		t.Fatalf("expected endpoint saved")
	}

	// test connection updates last test
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/test-connection", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/settings", nil)
	r.ServeHTTP(w, req)
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.LastTest == "" {
		t.Fatalf("expected last test set")
	}
}
