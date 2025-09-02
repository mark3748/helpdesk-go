package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Settings holds configuration values stored in memory.
type Settings struct {
	Storage  map[string]string `json:"storage"`
	OIDC     map[string]string `json:"oidc"`
	Mail     map[string]string `json:"mail"`
	LogPath  string            `json:"log_path"`
	LastTest string            `json:"last_test"`
}

var (
	mu       sync.RWMutex
	cfgStore = Settings{
		Storage:  map[string]string{},
		OIDC:     map[string]string{},
		Mail:     map[string]string{},
		LogPath:  "/config/logs",
		LastTest: "",
	}
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
	var data map[string]string
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

// TestConnection records a test run and returns log path and last result.
func TestConnection(c *gin.Context) {
	mu.Lock()
	cfgStore.LastTest = time.Now().Format(time.RFC3339)
	resp := gin.H{"ok": true, "log_path": cfgStore.LogPath, "last_test": cfgStore.LastTest}
	mu.Unlock()
	c.JSON(http.StatusOK, resp)
}
