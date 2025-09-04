package handlers

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// OIDCSettings holds OpenID Connect configuration.
type OIDCSettings struct {
	Issuer       string              `json:"issuer"`
	ClientID     string              `json:"client_id"`
	ClaimPath    string              `json:"claim_path"`
	ValueToRoles map[string][]string `json:"value_to_roles"`
}

// Settings holds configuration values stored in memory.
type Settings struct {
	Storage  map[string]string `json:"storage"`
	OIDC     OIDCSettings      `json:"oidc"`
	Mail     map[string]string `json:"mail"`
	LogPath  string            `json:"log_path"`
	LastTest string            `json:"last_test"`
}

var (
	mu       sync.RWMutex
	cfgStore = Settings{
		Storage:  map[string]string{},
		OIDC:     OIDCSettings{},
		Mail:     map[string]string{},
		LogPath:  "/config/logs",
		LastTest: "",
	}

	// EnqueueEmail is set by the API to enqueue email jobs.
	EnqueueEmail func(ctx context.Context, to, template string, data interface{})
)

// InitSettings sets initial values like log path.
func InitSettings(logPath string) {
	mu.Lock()
	defer mu.Unlock()
	if logPath != "" {
		cfgStore.LogPath = logPath
	}
}

// GetSettings returns the current configuration.
func GetSettings(c *gin.Context) {
	mu.RLock()
	defer mu.RUnlock()
	c.JSON(http.StatusOK, cfgStore)
}

// SaveStorageSettings stores storage configuration.
func SaveStorageSettings(c *gin.Context) {
	var data map[string]string
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mu.Lock()
	cfgStore.Storage = data
	mu.Unlock()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// SaveOIDCSettings stores OIDC configuration.
func SaveOIDCSettings(c *gin.Context) {
	var data OIDCSettings
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mu.Lock()
	cfgStore.OIDC = data
	mu.Unlock()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// SaveMailSettings stores mail configuration.
func SaveMailSettings(c *gin.Context) {
	var data map[string]string
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mu.Lock()
	cfgStore.Mail = data
	mu.Unlock()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// MailSettings returns the current mail configuration.
func MailSettings() map[string]string {
	mu.RLock()
	defer mu.RUnlock()
	out := make(map[string]string, len(cfgStore.Mail))
	for k, v := range cfgStore.Mail {
		out[k] = v
	}
	return out
}

// SendTestMail enqueues a test email via the worker.
func SendTestMail(c *gin.Context) {
	if EnqueueEmail == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mail queue unavailable"})
		return
	}
	to := c.Query("to")
	if to == "" {
		to = MailSettings()["smtp_from"]
	}
	EnqueueEmail(c.Request.Context(), to, "test_email", nil)
	c.JSON(http.StatusOK, gin.H{"queued": true})
}

// TestConnection records a test run and returns log path and last result.
func TestConnection(c *gin.Context) {
	mu.Lock()
	cfgStore.LastTest = time.Now().Format(time.RFC3339)
	resp := gin.H{"ok": true, "log_path": cfgStore.LogPath, "last_test": cfgStore.LastTest}
	mu.Unlock()
	c.JSON(http.StatusOK, resp)
}
