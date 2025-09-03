package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	app "github.com/mark3748/helpdesk-go/cmd/api/app"
)

// AuthUser represents the authenticated user.
type AuthUser struct {
	ID          string   `json:"id"`
	ExternalID  string   `json:"external_id"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	Roles       []string `json:"roles"`
}

func (u AuthUser) GetRoles() []string { return u.Roles }

// Middleware performs JWT validation or bypass during tests.
func Middleware(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.Cfg.TestBypassAuth {
			c.Set("user", AuthUser{
				ID:          "test-user",
				ExternalID:  "test",
				Email:       "test@example.com",
				DisplayName: "Test User",
				Roles:       []string{"agent"},
			})
			c.Next()
			return
		}
		if a.Keyf == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "jwks not configured"})
			return
		}
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(tokenStr, a.Keyf)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		u := AuthUser{
			ExternalID:  getStringClaim(claims, "sub"),
			Email:       getStringClaim(claims, "email"),
			DisplayName: getStringClaim(claims, "name"),
		}
		if u.DisplayName == "" {
			u.DisplayName = getStringClaim(claims, "preferred_username")
		}
		if groups, ok := claims[a.Cfg.OIDCGroupClaim]; ok {
			switch g := groups.(type) {
			case []interface{}:
				for _, v := range g {
					if s, ok := v.(string); ok {
						u.Roles = append(u.Roles, s)
					}
				}
			case []string:
				u.Roles = append(u.Roles, g...)
			case string:
				u.Roles = append(u.Roles, g)
			}
		}
		c.Set("user", u)
		c.Next()
	}
}

func getStringClaim(c jwt.MapClaims, key string) string {
	if v, ok := c[key].(string); ok {
		return v
	}
	return ""
}

// Me returns the authenticated user.
func Me(c *gin.Context) {
	u, ok := c.Get("user")
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	c.JSON(http.StatusOK, u)
}

// RequireRole ensures the user has one of the required roles.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		uVal, ok := c.Get("user")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}
		user, ok := uVal.(AuthUser)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return
		}
		for _, r := range user.Roles {
			if r == "admin" {
				c.Next()
				return
			}
		}
		for _, r := range user.Roles {
			for _, want := range roles {
				if r == want {
					c.Next()
					return
				}
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	}
}

// The following handlers are placeholders for future implementation.
func Login(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) }
}

func Logout() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) }
}

func ListUserRoles(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, []string{}) }
}

func AddUserRole(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.Status(http.StatusCreated) }
}

func RemoveUserRole(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) }
}
