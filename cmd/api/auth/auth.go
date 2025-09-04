package auth

import (
    "net/http"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

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

		// Local auth: verify cookie JWT signed with HMAC secret
		if strings.ToLower(a.Cfg.AuthMode) == "local" {
			cookie, err := c.Request.Cookie("hd_auth")
			if err != nil || cookie.Value == "" {
				app.AbortError(c, http.StatusUnauthorized, "unauthenticated", "unauthenticated", nil)
				return
			}
			token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrTokenUnverifiable
				}
				return []byte(a.Cfg.AuthLocalSecret), nil
			})
			if err != nil || !token.Valid {
				app.AbortError(c, http.StatusUnauthorized, "invalid_token", "invalid token", nil)
				return
			}
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				app.AbortError(c, http.StatusUnauthorized, "invalid_token", "invalid token", nil)
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
			if roles, ok := claims["roles"]; ok {
				switch g := roles.(type) {
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
			// Populate internal user ID and fill missing fields from DB if available
			if a.DB != nil {
				var id, em, dn string
				row := a.DB.QueryRow(c.Request.Context(), `select id::text, coalesce(email,''), coalesce(display_name,'') from users where external_id=$1 or lower(email)=lower($2) or lower(username)=lower($3) limit 1`, u.ExternalID, u.Email, u.Email)
				_ = row.Scan(&id, &em, &dn)
				if id != "" {
					u.ID = id
				}
				if u.Email == "" {
					u.Email = em
				}
				if u.DisplayName == "" {
					u.DisplayName = dn
				}
				if u.ID == "" && strings.HasPrefix(u.ExternalID, "local:") {
					uname := strings.TrimPrefix(u.ExternalID, "local:")
					_ = a.DB.QueryRow(c.Request.Context(), `select id::text, coalesce(email,''), coalesce(display_name,'') from users where lower(username)=lower($1) limit 1`, uname).Scan(&id, &em, &dn)
					if id != "" {
						u.ID = id
					}
					if u.Email == "" {
						u.Email = em
					}
					if u.DisplayName == "" {
						u.DisplayName = dn
					}
				}
			}
            // Augment roles with stored DB roles (union). Prefer querying by
            // resolved user ID; fall back to external_id, then username/email.
            if a.DB != nil {
                ctx := c.Request.Context()
                var rows pgx.Rows
                var err error
                switch {
                case u.ID != "":
                    rows, err = a.DB.Query(ctx, `
select r.name
from users u
left join user_roles ur on ur.user_id = u.id
left join roles r on r.id = ur.role_id
where u.id::text = $1`, u.ID)
                case u.ExternalID != "":
                    rows, err = a.DB.Query(ctx, `
select r.name
from users u
left join user_roles ur on ur.user_id = u.id
left join roles r on r.id = ur.role_id
where u.external_id = $1`, u.ExternalID)
                case u.Email != "":
                    rows, err = a.DB.Query(ctx, `
select r.name
from users u
left join user_roles ur on ur.user_id = u.id
left join roles r on r.id = ur.role_id
where lower(u.email) = lower($1)`, u.Email)
                default:
                    rows, err = nil, nil
                }
                if err == nil && rows != nil {
                    defer rows.Close()
                    for rows.Next() {
                        var name *string
                        if err := rows.Scan(&name); err == nil {
                            if name != nil && *name != "" && !hasRole(u.Roles, *name) {
                                u.Roles = append(u.Roles, *name)
                            }
                        }
                    }
                }
                // Ensure local admin always retains baseline roles even if DB links are missing
                if strings.HasPrefix(u.ExternalID, "local:") && strings.EqualFold(strings.TrimPrefix(u.ExternalID, "local:"), "admin") {
                    if !hasRole(u.Roles, "admin") { u.Roles = append(u.Roles, "admin") }
                    if !hasRole(u.Roles, "agent") { u.Roles = append(u.Roles, "agent") }
                }
            }
			c.Set("user", u)
			c.Next()
			return
		}

		// OIDC/JWT bearer auth using JWKS
		if a.Keyf == nil {
			app.AbortError(c, http.StatusInternalServerError, "jwks_not_configured", "jwks not configured", nil)
			return
		}
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			app.AbortError(c, http.StatusUnauthorized, "unauthenticated", "missing bearer token", nil)
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
        // Enforce acceptable algorithms and validate standard time-based claims;
        // allow optional leeway when configured.
        opts := []jwt.ParserOption{jwt.WithValidMethods([]string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512", "HS256", "HS384", "HS512"})}
        if a.Cfg.JWTClockSkewSeconds > 0 {
            opts = append(opts, jwt.WithLeeway(time.Duration(a.Cfg.JWTClockSkewSeconds)*time.Second))
        }
        parser := jwt.NewParser(opts...)
        token, err := parser.Parse(tokenStr, a.Keyf)
        if err != nil || !token.Valid {
            app.AbortError(c, http.StatusUnauthorized, "invalid_token", "invalid token", nil)
            return
        }
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        app.AbortError(c, http.StatusUnauthorized, "invalid_token", "invalid token", nil)
        return
    }
        // Optional issuer validation when configured
        if iss := a.Cfg.OIDCIssuer; iss != "" {
            if got := getStringClaim(claims, "iss"); got != iss {
                app.AbortError(c, http.StatusUnauthorized, "invalid_issuer", "invalid issuer", nil)
                return
            }
        }
        // Optional audience validation when configured
        if aud := a.Cfg.OIDCAudience; aud != "" {
            if !audienceContains(claims, aud) {
                app.AbortError(c, http.StatusUnauthorized, "invalid_audience", "invalid audience", nil)
                return
            }
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
		// Populate internal user ID and fill missing fields from DB if available
		if a.DB != nil {
			var id, em, dn string
			row := a.DB.QueryRow(c.Request.Context(), `select id::text, coalesce(email,''), coalesce(display_name,'') from users where external_id=$1 or lower(email)=lower($2) or lower(username)=lower($3) limit 1`, u.ExternalID, u.Email, u.Email)
			_ = row.Scan(&id, &em, &dn)
			if id != "" {
				u.ID = id
			}
			if u.Email == "" {
				u.Email = em
			}
			if u.DisplayName == "" {
				u.DisplayName = dn
			}
		}
		// Augment roles with stored DB roles (union)
		if a.DB != nil && u.ExternalID != "" {
			rows, err := a.DB.Query(c.Request.Context(), `
select r.name from users u
left join user_roles ur on ur.user_id=u.id
left join roles r on r.id=ur.role_id
where u.external_id=$1`, u.ExternalID)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var name *string
					if err := rows.Scan(&name); err == nil {
						if name != nil && *name != "" && !hasRole(u.Roles, *name) {
							u.Roles = append(u.Roles, *name)
						}
					}
				}
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

func audienceContains(c jwt.MapClaims, want string) bool {
    if v, ok := c["aud"]; ok {
        switch t := v.(type) {
        case string:
            return t == want
        case []interface{}:
            for _, e := range t {
                if s, ok := e.(string); ok && s == want {
                    return true
                }
            }
        case []string:
            for _, s := range t {
                if s == want { return true }
            }
        }
    }
    return false
}

// Me returns the authenticated user.
func Me(c *gin.Context) {
	u, ok := c.Get("user")
	if !ok {
		app.AbortError(c, http.StatusUnauthorized, "unauthenticated", "unauthenticated", nil)
		return
	}
	c.JSON(http.StatusOK, u)
}

// RequireRole ensures the user has one of the required roles.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		uVal, ok := c.Get("user")
		if !ok {
			app.AbortError(c, http.StatusUnauthorized, "unauthenticated", "unauthenticated", nil)
			return
		}
		user, ok := uVal.(AuthUser)
		if !ok {
			app.AbortError(c, http.StatusUnauthorized, "invalid_user", "invalid user", nil)
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
		app.AbortError(c, http.StatusForbidden, "forbidden", "forbidden", nil)
	}
}

// The following handlers implement local auth when enabled; otherwise act as no-ops.
func Login(a *app.App) gin.HandlerFunc {
	type creds struct{ Username, Password string }
	return func(c *gin.Context) {
		if strings.ToLower(a.Cfg.AuthMode) != "local" {
			app.AbortError(c, http.StatusNotImplemented, "login_disabled", "login disabled", nil)
			return
		}
		if a.Cfg.AuthLocalSecret == "" {
			app.AbortError(c, http.StatusInternalServerError, "auth_not_configured", "local auth not configured", nil)
			return
		}
		var in creds
		if err := c.ShouldBindJSON(&in); err != nil {
			app.AbortError(c, http.StatusBadRequest, "invalid_json", "invalid json", nil)
			return
		}
		// Check DB for local user or fallback to built-in admin
		var uid, externalID, email, name, hash string
		if a.DB != nil {
			const find = `select id::text, coalesce(external_id,''), coalesce(email,''), coalesce(display_name,''), coalesce(password_hash,'')
from users where lower(username)=lower($1) or lower(email)=lower($1) limit 1`
			_ = a.DB.QueryRow(c.Request.Context(), find, in.Username).Scan(&uid, &externalID, &email, &name, &hash)
		}
		if uid == "" {
			// Fallback: built-in admin via env password
			if in.Username != "admin" || in.Password != a.Cfg.AdminPassword {
				app.AbortError(c, http.StatusUnauthorized, "invalid_credentials", "invalid credentials", nil)
				return
			}
			// Ensure local admin exists in DB with roles
			const upsertUser = `
insert into users (external_id, username, email, display_name)
values ($1, $2, $3, $4)
on conflict (username) do update set email=excluded.email, display_name=excluded.display_name
returning id::text`
			if a.DB != nil {
				_ = a.DB.QueryRow(c.Request.Context(), upsertUser, "local:admin", "admin", "admin@example.com", "Administrator").Scan(&uid)
				// Seed password hash
				if ph, err := bcrypt.GenerateFromPassword([]byte(a.Cfg.AdminPassword), bcrypt.DefaultCost); err == nil {
					_, _ = a.DB.Exec(c.Request.Context(), "update users set password_hash=$1 where id=$2", string(ph), uid)
				}
				// Ensure roles exist and are assigned
				const ensureRole = `insert into roles (id, name) values (gen_random_uuid(), $1) on conflict do nothing`
				_, _ = a.DB.Exec(c.Request.Context(), ensureRole, "admin")
				_, _ = a.DB.Exec(c.Request.Context(), ensureRole, "agent")
				const linkRole = `insert into user_roles (user_id, role_id) select $1, r.id from roles r where r.name=$2 on conflict do nothing`
				_, _ = a.DB.Exec(c.Request.Context(), linkRole, uid, "admin")
				_, _ = a.DB.Exec(c.Request.Context(), linkRole, uid, "agent")
			}
			externalID = "local:admin"
			email = "admin@example.com"
			name = "Administrator"
		} else {
			// Validate password for local DB user
			if hash == "" || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
				app.AbortError(c, http.StatusUnauthorized, "invalid_credentials", "invalid credentials", nil)
				return
			}
			if externalID == "" {
				externalID = "local:" + in.Username
			}
		}
		claims := jwt.MapClaims{
			"sub":   externalID,
			"email": email,
			"name":  name,
			"exp":   time.Now().Add(24 * time.Hour).Unix(),
			"iat":   time.Now().Unix(),
		}
		tk := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		s, err := tk.SignedString([]byte(a.Cfg.AuthLocalSecret))
		if err != nil {
			app.AbortError(c, http.StatusInternalServerError, "sign_token_failed", "failed to sign token", nil)
			return
		}
		// HttpOnly cookie for browser auth; Secure false in dev, true in prod
		secure := a.Cfg.Env == "prod"
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "hd_auth",
			Value:    s,
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(24 * time.Hour),
		})
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Clear cookie
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "hd_auth",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
		})
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// AddUserRole and RemoveUserRole implemented below

// ListUserRoles queries roles for a user by UUID or external_id.
// Path: GET /api/users/:id/roles
func ListUserRoles(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			app.AbortError(c, http.StatusBadRequest, "invalid_request", "missing id", nil)
			return
		}
		// Query roles where users.id::text=$1 OR users.external_id=$1
		const q = `
select r.name
from users u
left join user_roles ur on ur.user_id = u.id
left join roles r on r.id = ur.role_id
where (u.id::text = $1 or u.external_id = $1)
order by r.name nulls last`
		rows, err := a.DB.Query(c.Request.Context(), q, id)
		if err != nil {
			app.AbortError(c, http.StatusInternalServerError, "db_error", err.Error(), nil)
			return
		}
		defer rows.Close()
		roles := make([]string, 0, 4)
		for rows.Next() {
			var name *string
			if err := rows.Scan(&name); err != nil {
				app.AbortError(c, http.StatusInternalServerError, "db_error", err.Error(), nil)
				return
			}
			if name != nil && *name != "" {
				roles = append(roles, *name)
			}
		}
		// If user not found, still return empty list with 200 to keep UI simple
		c.JSON(http.StatusOK, roles)
	}
}

// AddUserRole assigns a named role to a user by id|external_id.
// Body: {"role":"agent"}
func AddUserRole(a *app.App) gin.HandlerFunc {
	type body struct {
		Role string `json:"role"`
	}
	return func(c *gin.Context) {
		id := c.Param("id")
		var b body
		if err := c.ShouldBindJSON(&b); err != nil || b.Role == "" {
			app.AbortError(c, http.StatusBadRequest, "invalid_json", "invalid json", nil)
			return
		}
		const findUser = `select id from users where id::text=$1 or external_id=$1 or username=$1 or email=$1`
		var uid string
		if err := a.DB.QueryRow(c.Request.Context(), findUser, id).Scan(&uid); err != nil {
			app.AbortError(c, http.StatusNotFound, "not_found", "user not found", nil)
			return
		}
		// Ensure role exists
		_, _ = a.DB.Exec(c.Request.Context(), `insert into roles (id, name) values (gen_random_uuid(), $1) on conflict do nothing`, b.Role)
		// Link
		if _, err := a.DB.Exec(c.Request.Context(), `insert into user_roles (user_id, role_id) select $1, r.id from roles r where r.name=$2 on conflict do nothing`, uid, b.Role); err != nil {
			app.AbortError(c, http.StatusInternalServerError, "db_error", err.Error(), nil)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	}
}

// RemoveUserRole unassigns a role from a user
func RemoveUserRole(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		role := c.Param("role")
		const findUser = `select id from users where id::text=$1 or external_id=$1 or username=$1 or email=$1`
		var uid string
		if err := a.DB.QueryRow(c.Request.Context(), findUser, id).Scan(&uid); err != nil {
			app.AbortError(c, http.StatusNotFound, "not_found", "user not found", nil)
			return
		}
		if _, err := a.DB.Exec(c.Request.Context(), `delete from user_roles using roles r where user_roles.user_id=$1 and user_roles.role_id=r.id and r.name=$2`, uid, role); err != nil {
			app.AbortError(c, http.StatusInternalServerError, "db_error", err.Error(), nil)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func hasRole(list []string, r string) bool {
	for _, v := range list {
		if v == r {
			return true
		}
	}
	return false
}
