package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
	app "github.com/mark3748/helpdesk-go/cmd/api/app"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
	"golang.org/x/oauth2"
)

// OIDCLogin initiates the OIDC authorization code flow.
func OIDCLogin(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := loadSettings(c.Request.Context())
		if err != nil {
			app.AbortError(c, http.StatusInternalServerError, "settings_error", "failed to load settings", nil)
			return
		}
		if s.OIDC.Issuer == "" || s.OIDC.ClientID == "" || s.OIDC.ClientSecret == "" {
			app.AbortError(c, http.StatusBadRequest, "oidc_not_configured", "OpenID Connect is not fully configured.", nil)
			return
		}

		provider, err := oidc.NewProvider(c.Request.Context(), s.OIDC.Issuer)
		if err != nil {
			app.AbortError(c, http.StatusInternalServerError, "oidc_discovery_failed", "Failed to discover OIDC provider: "+err.Error(), nil)
			return
		}

		redirectURL := s.OIDC.RedirectURL
		if redirectURL == "" {
			// Fallback: infer from request host if not set
			scheme := "http"
			if c.Request.TLS != nil {
				scheme = "https"
			}
			// Use X-Forwarded-Proto if available
			if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
				scheme = proto
			}
			redirectURL = scheme + "://" + c.Request.Host + "/api/auth/oidc/callback"
		}

		config := oauth2.Config{
			ClientID:     s.OIDC.ClientID,
			ClientSecret: s.OIDC.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  redirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		}
		if s.OIDC.Scopes != "" {
			// Split by comma or space
			customScopes := strings.Fields(strings.ReplaceAll(s.OIDC.Scopes, ",", " "))
			config.Scopes = append(config.Scopes, customScopes...)
		}

		state, err := generateRandomString(32)
		if err != nil {
			app.AbortError(c, http.StatusInternalServerError, "random_error", "failed to generate state", nil)
			return
		}

		// Store state in cookie to verify callback
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "hd_oidc_state",
			Value:    state,
			Path:     "/",
			MaxAge:   300, // 5 minutes
			HttpOnly: true,
			Secure:   a.Cfg.Env == "prod",
			SameSite: http.SameSiteLaxMode,
		})

		http.Redirect(c.Writer, c.Request, config.AuthCodeURL(state), http.StatusFound)
	}
}

// OIDCCallback handles the return from the OIDC provider.
func OIDCCallback(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Request.Cookie("hd_oidc_state")
		if err != nil {
			app.AbortError(c, http.StatusBadRequest, "state_cookie_missing", "missing state cookie", nil)
			return
		}
		if c.Query("state") != cookie.Value {
			app.AbortError(c, http.StatusBadRequest, "invalid_state", "state mismatch", nil)
			return
		}

		s, err := loadSettings(c.Request.Context())
		if err != nil {
			app.AbortError(c, http.StatusInternalServerError, "settings_error", "failed to load settings", nil)
			return
		}

		provider, err := oidc.NewProvider(c.Request.Context(), s.OIDC.Issuer)
		if err != nil {
			app.AbortError(c, http.StatusInternalServerError, "oidc_error", "provider discovery failed", nil)
			return
		}

		redirectURL := s.OIDC.RedirectURL
		if redirectURL == "" {
			scheme := "http"
			if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
				scheme = proto
			} else if c.Request.TLS != nil {
				scheme = "https"
			}
			redirectURL = scheme + "://" + c.Request.Host + "/api/auth/oidc/callback"
		}

		config := oauth2.Config{
			ClientID:     s.OIDC.ClientID,
			ClientSecret: s.OIDC.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  redirectURL,
		}

		oauth2Token, err := config.Exchange(c.Request.Context(), c.Query("code"))
		if err != nil {
			app.AbortError(c, http.StatusInternalServerError, "token_exchange_failed", "failed to exchange code: "+err.Error(), nil)
			return
		}

		verifier := provider.Verifier(&oidc.Config{ClientID: s.OIDC.ClientID})
		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			app.AbortError(c, http.StatusInternalServerError, "no_id_token", "no id_token in response", nil)
			return
		}

		idToken, err := verifier.Verify(c.Request.Context(), rawIDToken)
		if err != nil {
			app.AbortError(c, http.StatusInternalServerError, "token_verify_failed", "failed to verify ID token: "+err.Error(), nil)
			return
		}

		var claims map[string]interface{}
		if err := idToken.Claims(&claims); err != nil {
			app.AbortError(c, http.StatusInternalServerError, "claims_error", "failed to parse claims", nil)
			return
		}

		// Extract User Details
		// 1. External ID (sub)
		sub, _ := claims["sub"].(string)
		if sub == "" {
			app.AbortError(c, http.StatusInternalServerError, "invalid_claims", "sub claim missing", nil)
			return
		}

		// Prefix with provider name to correspond with existing logic if needed, or just use raw sub?
		// Existing logic in auth.go handles "local:..." for local users.
		// We should prefix with "oidc:" or similar to avoid collisions if we support multiple?
		// The user settings just say "Provider Name". Let's use "oidc:<sub?>"
		// Actually, let's keep it simple: "oidc:" + sub.
		externalID := "oidc:" + sub

		// 2. Email
		email, _ := claims["email"].(string)

		// 3. Name / Username
		// Prefer explicit claim if configured
		var username string
		if s.OIDC.UsernameClaim != "" {
			username, _ = claims[s.OIDC.UsernameClaim].(string)
		}
		if username == "" {
			username, _ = claims["preferred_username"].(string)
		}
		if username == "" {
			username, _ = claims["name"].(string)
		}
		if username == "" {
			username = email
		}

		name, _ := claims["name"].(string)
		if name == "" {
			name = username
		}

		// 4. Groups
		var groups []string
		groupClaim := s.OIDC.GroupClaimName
		if groupClaim == "" {
			groupClaim = "groups"
		}
		if g, ok := claims[groupClaim]; ok {
			switch v := g.(type) {
			case []interface{}:
				for _, item := range v {
					if str, ok := item.(string); ok {
						groups = append(groups, str)
					}
				}
			case []string:
				groups = append(groups, v...)
			}
		}

		// DB Sync (JIT)
		userID, err := syncUser(c, a, externalID, username, email, name, s.OIDC.AutoOnboard)
		if err != nil {
			app.AbortError(c, http.StatusInternalServerError, "db_sync_error", "failed to sync user: "+err.Error(), nil)
			return
		}

		// Sync Roles
		if len(groups) > 0 {
			if err := syncRoles(c, a, userID, groups, s.OIDC); err != nil {
				// log error but continue?
			}
		} else if s.OIDC.AutoOnboard {
			// Assign default role if auto-onboard is on and no groups?
			// Actually auto-onboard usually means allow login at all.
			// If we are here, we are allowing login.
		}

		// Set Session
		secure := a.Cfg.Env == "prod"
		if err := authpkg.SetSessionCookie(c, a.Cfg.AuthLocalSecret, externalID, email, name, secure); err != nil {
			app.AbortError(c, http.StatusInternalServerError, "session_error", "failed to set session", nil)
			return
		}

		http.Redirect(c.Writer, c.Request, "/", http.StatusFound)
	}
}

func generateRandomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func syncUser(c *gin.Context, a *app.App, externalID, username, email, name string, autoOnboard bool) (string, error) {
	if a.DB == nil {
		return "", errors.New("database not available")
	}

	// First check if user exists
	var existingID string
	checkQ := `SELECT id::text FROM users WHERE external_id = $1`
	err := a.DB.QueryRow(c.Request.Context(), checkQ, externalID).Scan(&existingID)
	
	if err == nil {
		// User exists, update their information
		updateQ := `UPDATE users SET email=$2, display_name=$3, username=$4 WHERE external_id=$1 RETURNING id::text`
		var id string
		if err := a.DB.QueryRow(c.Request.Context(), updateQ, externalID, email, name, username).Scan(&id); err != nil {
			return "", err
		}
		return id, nil
	}

	// User doesn't exist - check if auto-onboard is enabled
	if !autoOnboard {
		return "", errors.New("user does not exist and auto-onboard is disabled")
	}

	// Create new user
	if username == "" {
		username = externalID
	}

	const insertQ = `
    INSERT INTO users (external_id, username, email, display_name)
    VALUES ($1, $2, $3, $4)
    RETURNING id::text`

	var id string
	originalUsername := username
	
	// Try up to 5 times with different suffixes if there's a username conflict
	for attempt := 0; attempt < 5; attempt++ {
		err = a.DB.QueryRow(c.Request.Context(), insertQ, externalID, username, email, name).Scan(&id)
		if err == nil {
			// Success
			return id, nil
		}
		
		// Check if it's a username conflict
		if !strings.Contains(err.Error(), "duplicate") && !strings.Contains(err.Error(), "unique") {
			// Not a username conflict, return the error
			return "", err
		}
		
		// Generate a new username with suffix
		if attempt == 0 {
			// First retry: use last 8 chars of externalID
			suffix := externalID
			if len(externalID) > 8 {
				suffix = externalID[len(externalID)-8:]
			}
			username = originalUsername + "_" + suffix
		} else {
			// Subsequent retries: append random suffix
			randomBytes := make([]byte, 4)
			if _, err := rand.Read(randomBytes); err != nil {
				// If crypto/rand fails, this is a serious system issue
				// Return error rather than using a predictable fallback
				return "", errors.New("failed to generate unique username: random generation unavailable")
			}
			username = originalUsername + "_" + base64.RawURLEncoding.EncodeToString(randomBytes)
		}
	}
	
	// All retries failed
	return "", errors.New("failed to create user: username conflict after multiple attempts")
}

func syncRoles(c *gin.Context, a *app.App, userID string, groups []string, settings OIDCSettings) error {
	if a.DB == nil {
		return nil
	}
	// settings.ValueToRoles maps group_name -> [role_name, role_name]
	// Also AdminGroup -> "admin"

	// Gather target roles
	targetRoles := make(map[string]bool)

	// Admin Group Check
	if settings.AdminGroup != "" {
		for _, g := range groups {
			if g == settings.AdminGroup {
				targetRoles["admin"] = true
				targetRoles["agent"] = true // admin implies agent usually?
			}
		}
	}

	// Value To Roles Mapping
	for _, g := range groups {
		if roles, ok := settings.ValueToRoles[g]; ok {
			for _, r := range roles {
				targetRoles[r] = true
			}
		}
	}

	// Get current roles from database
	currentRoles := make(map[string]string) // role_name -> role_id
	rows, err := a.DB.Query(c.Request.Context(), `
		SELECT r.id, r.name 
		FROM user_roles ur 
		JOIN roles r ON ur.role_id = r.id 
		WHERE ur.user_id = $1`, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var rid, rname string
		if err := rows.Scan(&rid, &rname); err != nil {
			continue
		}
		currentRoles[rname] = rid
	}

	// Add new roles
	for r := range targetRoles {
		// Skip if already has role
		if _, exists := currentRoles[r]; exists {
			continue
		}

		// Look up role id
		var rid string
		err := a.DB.QueryRow(c.Request.Context(), "SELECT id FROM roles WHERE name=$1", r).Scan(&rid)
		if err != nil {
			// Log error if role doesn't exist
			if errors.Is(err, pgx.ErrNoRows) {
				log.Warn().Str("role", r).Str("user_id", userID).Msg("role not found in database - check OIDC role mapping configuration")
			} else {
				log.Error().Err(err).Str("role", r).Str("user_id", userID).Msg("failed to lookup role")
			}
			continue
		}
		_, _ = a.DB.Exec(c.Request.Context(), "INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", userID, rid)
	}

	// Remove roles that are no longer in target
	for rname, rid := range currentRoles {
		if !targetRoles[rname] {
			_, err := a.DB.Exec(c.Request.Context(), "DELETE FROM user_roles WHERE user_id=$1 AND role_id=$2", userID, rid)
			if err != nil {
				log.Error().Err(err).Str("role", rname).Str("user_id", userID).Msg("failed to remove role")
			}
		}
	}

	return nil
}
