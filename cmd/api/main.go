package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	handlers "github.com/mark3748/helpdesk-go/cmd/api/handlers"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// openapi.yaml is served from disk to avoid cross-package embed limitations.

// Serve Swagger UI locally to avoid external CDN dependency.
var swaggerHTML = `<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>Helpdesk API Docs</title>
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <link rel="stylesheet" type="text/css" href="/swagger/swagger-ui.css" />
    <style>body { margin: 0; padding: 0; }</style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="/swagger/swagger-ui-bundle.js" charset="UTF-8"></script>
    <script src="/swagger/swagger-ui-standalone-preset.js" charset="UTF-8"></script>
    <script>
      window.onload = () => {
        window.ui = SwaggerUIBundle({
          url: '/openapi.yaml',
          dom_id: '#swagger-ui',
          presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
          layout: 'StandaloneLayout'
        });
      };
    </script>
  </body>
</html>`

type Config struct {
	Addr           string
	DatabaseURL    string
	Env            string
	RedisAddr      string
	OIDCIssuer     string
	JWKSURL        string
	OIDCGroupClaim string
	MinIOEndpoint  string
	MinIOAccess    string
	MinIOSecret    string
	MinIOBucket    string
	MinIOUseSSL    bool
	// Testing helpers
	TestBypassAuth bool
	// Local auth
	AuthMode        string // "oidc" or "local"
	AuthLocalSecret string
	// Filesystem object store for dev/local
	FileStorePath   string
	OpenAPISpecPath string
	LogPath         string
}

func getConfig() Config {
	_ = godotenv.Load()
	cfg := Config{
		Addr:            getEnv("ADDR", ":8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/helpdesk?sslmode=disable"),
		Env:             getEnv("ENV", "dev"),
		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),
		OIDCIssuer:      getEnv("OIDC_ISSUER", ""),
		JWKSURL:         getEnv("OIDC_JWKS_URL", ""),
		OIDCGroupClaim:  getEnv("OIDC_GROUP_CLAIM", "groups"),
		MinIOEndpoint:   getEnv("MINIO_ENDPOINT", ""),
		MinIOAccess:     getEnv("MINIO_ACCESS_KEY", ""),
		MinIOSecret:     getEnv("MINIO_SECRET_KEY", ""),
		MinIOBucket:     getEnv("MINIO_BUCKET", "attachments"),
		MinIOUseSSL:     getEnv("MINIO_USE_SSL", "false") == "true",
		TestBypassAuth:  getEnv("TEST_BYPASS_AUTH", "false") == "true",
		AuthMode:        getEnv("AUTH_MODE", "oidc"),
		AuthLocalSecret: getEnv("AUTH_LOCAL_SECRET", ""),
		FileStorePath:   getEnv("FILESTORE_PATH", ""),
		OpenAPISpecPath: getEnv("OPENAPI_SPEC_PATH", ""),
		LogPath:         getEnv("LOG_PATH", "/config/logs"),
	}
	return cfg
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// DB is a minimal interface to allow mocking in tests.
type DB interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

// ObjectStore wraps the subset of MinIO we need for tests.
type ObjectStore interface {
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
}

// fsObjectStore implements ObjectStore on the local filesystem for development/testing.
type fsObjectStore struct {
	base string
}

func (f *fsObjectStore) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	_ = ctx
	dir := f.base
	if bucketName != "" {
		dir = dir + string(os.PathSeparator) + bucketName
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return minio.UploadInfo{}, err
	}
	fp := dir + string(os.PathSeparator) + objectName
	tmp := fp + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return minio.UploadInfo{}, err
	}
	defer out.Close()
	if _, err := io.Copy(out, reader); err != nil {
		_ = os.Remove(tmp)
		return minio.UploadInfo{}, err
	}
	if err := os.Rename(tmp, fp); err != nil {
		return minio.UploadInfo{}, err
	}
	return minio.UploadInfo{Bucket: bucketName, Key: objectName, Size: objectSize}, nil
}

func (f *fsObjectStore) RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error {
	_ = ctx
	_ = opts
	dir := f.base
	if bucketName != "" {
		dir = dir + string(os.PathSeparator) + bucketName
	}
	fp := dir + string(os.PathSeparator) + objectName
	return os.Remove(fp)
}

type App struct {
	cfg  Config
	db   DB
	r    *gin.Engine
	keyf jwt.Keyfunc
	m    ObjectStore
	q    *redis.Client
}

// NewApp constructs an App with injected dependencies and registers routes.
func NewApp(cfg Config, db DB, keyf jwt.Keyfunc, store ObjectStore, q *redis.Client) *App {
	a := &App{cfg: cfg, db: db, r: gin.New(), keyf: keyf, m: store, q: q}
	handlers.InitSettings(cfg.LogPath)
	a.r.Use(gin.Recovery())
	a.r.Use(gin.Logger())
	a.routes()
	return a
}

func main() {
	cfg := getConfig()
	writer := io.Writer(os.Stdout)
	if cfg.Env == "dev" {
		writer = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	if err := os.MkdirAll(cfg.LogPath, 0o755); err != nil {
		log.Warn().Err(err).Str("dir", cfg.LogPath).Msg("using stdout for logs")
	} else {
		logFile := filepath.Join(cfg.LogPath, "api.log")
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			log.Warn().Err(err).Str("path", logFile).Msg("using stdout for logs")
		} else {
			if cfg.Env == "dev" {
				writer = zerolog.MultiLevelWriter(f, writer)
			} else {
				writer = f
			}
			defer f.Close()
		}
	}
	log.Logger = zerolog.New(writer).With().Timestamp().Logger()

	// DB connect
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("db connect")
	}
	defer pool.Close()

	// Migrate (embedded goose) using pgx stdlib driver
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal().Err(err).Msg("goose dialect")
	}
	sqldb, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("sql open for goose")
	}
	defer sqldb.Close()
	if err := goose.UpContext(ctx, sqldb, "migrations"); err != nil {
		log.Fatal().Err(err).Msg("migrate up")
	}

	// JWKS-backed Keyfunc
	var keyf jwt.Keyfunc
	if cfg.JWKSURL != "" {
		// Fetch JWKS on startup and refresh periodically
		httpClient := &http.Client{Timeout: 10 * time.Second}
		set, err := jwk.Fetch(ctx, cfg.JWKSURL, jwk.WithHTTPClient(httpClient))
		if err != nil {
			log.Fatal().Err(err).Str("jwks_url", cfg.JWKSURL).Msg("fetch jwks")
		}
		// simple periodic refresh
		setPtr := &set
		go func() {
			ticker := time.NewTicker(10 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				if newSet, err := jwk.Fetch(context.Background(), cfg.JWKSURL, jwk.WithHTTPClient(httpClient)); err == nil {
					*setPtr = newSet
				}
			}
		}()
		keyf = func(t *jwt.Token) (interface{}, error) {
			kid, _ := t.Header["kid"].(string)
			if kid != "" {
				if key, ok := (*setPtr).LookupKeyID(kid); ok {
					var pub any
					if err := key.Raw(&pub); err != nil {
						return nil, err
					}
					return pub, nil
				}
			}
			// fallback: use the first key in the set
			it := (*setPtr).Iterate(context.Background())
			if it.Next(context.Background()) {
				pair := it.Pair()
				if key, ok := pair.Value.(jwk.Key); ok {
					var pub any
					if err := key.Raw(&pub); err != nil {
						return nil, err
					}
					return pub, nil
				}
			}
			return nil, fmt.Errorf("no jwk for kid: %s", kid)
		}
	}

	var mc *minio.Client
	if cfg.MinIOEndpoint != "" {
		mc, err = minio.New(cfg.MinIOEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.MinIOAccess, cfg.MinIOSecret, ""),
			Secure: cfg.MinIOUseSSL,
		})
		if err != nil {
			log.Fatal().Err(err).Msg("minio init")
		}
	}

	// Redis client (optional)
	var rdb *redis.Client
	if cfg.RedisAddr != "" {
		rdb = redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Error().Err(err).Msg("redis ping")
		}
		defer rdb.Close()
	}

	var store ObjectStore
	if mc != nil {
		store = mc
	} else if cfg.FileStorePath != "" {
		if err := os.MkdirAll(cfg.FileStorePath, 0o755); err != nil {
			log.Fatal().Err(err).Str("path", cfg.FileStorePath).Msg("create filestore path")
		}
		store = &fsObjectStore{base: cfg.FileStorePath}
	}

	// Seed a dev admin for local auth
	if cfg.AuthMode == "local" && cfg.Env == "dev" {
		if err := seedLocalAdmin(context.Background(), pool); err != nil {
			log.Error().Err(err).Msg("seed local admin")
		}
	}

	a := NewApp(cfg, pool, keyf, store, rdb)

	srv := &http.Server{
		Addr:           cfg.Addr,
		Handler:        a.r,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Info().Str("addr", cfg.Addr).Msg("api listening")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("listen")
	}
}

func (a *App) routes() {
	a.r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	a.r.GET("/csat/:token", a.submitCSAT)
	// API docs UI and spec
	// Serve bundled Swagger UI assets from container image
	a.r.Static("/swagger", "/opt/helpdesk/swagger")
	a.r.GET("/docs", a.docsUI)
	a.r.GET("/openapi.yaml", a.openapiSpec)

	// Local auth endpoints
	if a.cfg.AuthMode == "local" {
		a.r.POST("/login", a.login)
		a.r.POST("/logout", a.logout)
	}

	auth := a.r.Group("/")
	auth.Use(a.authMiddleware())
	auth.GET("/me", a.me)
	auth.GET("/events", handlers.Events(a.q))

	auth.GET("/settings", a.requireRole("admin"), handlers.GetSettings)
	auth.POST("/test-connection", a.requireRole("admin"), handlers.TestConnection)
	auth.POST("/settings/storage", a.requireRole("admin"), handlers.SaveStorageSettings)
	auth.POST("/settings/oidc", a.requireRole("admin"), handlers.SaveOIDCSettings)
	auth.POST("/settings/mail", a.requireRole("admin"), handlers.SaveMailSettings)

	auth.GET("/users/:id/roles", a.requireRole("admin"), a.listUserRoles)
	auth.POST("/users/:id/roles", a.requireRole("admin"), a.addUserRole)
	auth.DELETE("/users/:id/roles/:role", a.requireRole("admin"), a.removeUserRole)

	// Tickets
	auth.GET("/tickets", a.listTickets)
	auth.POST("/tickets", a.createTicket)
	auth.GET("/tickets/:id", a.getTicket)
	auth.PATCH("/tickets/:id", a.requireRole("agent", "manager"), a.updateTicket)
	auth.GET("/tickets/:id/comments", a.listComments)
	auth.POST("/tickets/:id/comments", a.addComment)
	auth.GET("/tickets/:id/attachments", a.listAttachments)
	auth.POST("/tickets/:id/attachments", a.uploadAttachment)
	auth.GET("/tickets/:id/attachments/:attID", a.getAttachment)
	auth.DELETE("/tickets/:id/attachments/:attID", a.deleteAttachment)
	auth.GET("/tickets/:id/watchers", a.listWatchers)
	auth.POST("/tickets/:id/watchers", a.addWatcher)
	auth.DELETE("/tickets/:id/watchers/:userID", a.removeWatcher)
	auth.GET("/metrics/sla", a.requireRole("agent"), a.metricsSLA)
	auth.GET("/metrics/resolution", a.requireRole("agent"), a.metricsResolution)
	auth.GET("/metrics/tickets", a.requireRole("agent"), a.metricsTicketVolume)
	auth.POST("/exports/tickets", a.requireRole("agent"), a.exportTickets)
}

func (a *App) docsUI(c *gin.Context) {
	c.Data(200, "text/html; charset=utf-8", []byte(swaggerHTML))
}

func (a *App) openapiSpec(c *gin.Context) {
	candidates := []string{}
	if a.cfg.OpenAPISpecPath != "" {
		candidates = append(candidates, a.cfg.OpenAPISpecPath)
	}
	// Common defaults for dev and container images
	candidates = append(candidates, "docs/openapi.yaml", "/opt/helpdesk/docs/openapi.yaml")
	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err == nil {
			c.Data(200, "application/yaml", b)
			return
		}
	}
	c.JSON(404, gin.H{"error": "openapi spec not found"})
}

type AuthUser struct {
	ID          string   `json:"id"`
	ExternalID  string   `json:"external_id"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	Roles       []string `json:"roles"`
}

func (u AuthUser) GetRoles() []string { return u.Roles }

func (a *App) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Testing bypass: allow unit tests to inject a user without JWKS/token.
		if a.cfg.TestBypassAuth {
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

		if a.cfg.AuthMode == "local" {
			tokenStr, err := c.Cookie("auth")
			if err != nil || tokenStr == "" {
				c.AbortWithStatusJSON(401, gin.H{"error": "missing auth cookie"})
				return
			}
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return []byte(a.cfg.AuthLocalSecret), nil
			})
			if err != nil || !token.Valid {
				c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
				return
			}
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				c.AbortWithStatusJSON(401, gin.H{"error": "invalid claims"})
				return
			}
			uid, _ := claims["sub"].(string)
			ctx := c.Request.Context()
			var userID, mail, displayName string
			if err := a.db.QueryRow(ctx, "select id, coalesce(email,''), coalesce(display_name,'') from users where id=$1", uid).Scan(&userID, &mail, &displayName); err != nil {
				c.AbortWithStatusJSON(401, gin.H{"error": "unknown user"})
				return
			}
			rows, err := a.db.Query(ctx, "select r.name from user_roles ur join roles r on ur.role_id=r.id where ur.user_id=$1", userID)
			if err != nil {
				c.AbortWithStatusJSON(500, gin.H{"error": "role lookup"})
				return
			}
			defer rows.Close()
			roles := []string{}
			for rows.Next() {
				var role string
				if err := rows.Scan(&role); err == nil {
					roles = append(roles, role)
				}
			}
			authUser := AuthUser{ID: userID, ExternalID: "", Email: mail, DisplayName: displayName, Roles: roles}
			c.Set("user", authUser)
			c.Next()
			return
		}

		if a.keyf == nil {
			c.AbortWithStatusJSON(500, gin.H{"error": "jwks not configured"})
			return
		}
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(401, gin.H{"error": "missing bearer token"})
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(tokenStr, a.keyf)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid claims"})
			return
		}
		if iss, ok := claims["iss"].(string); ok && a.cfg.OIDCIssuer != "" && iss != a.cfg.OIDCIssuer {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid issuer"})
			return
		}
		sub, _ := claims["sub"].(string)
		email, _ := claims["email"].(string)
		name, _ := claims["name"].(string)

		ctx := c.Request.Context()
		var userID, mail, displayName string
		err = a.db.QueryRow(ctx, "select id, coalesce(email,''), coalesce(display_name,'') from users where external_id=$1", sub).Scan(&userID, &mail, &displayName)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				err = a.db.QueryRow(ctx, "insert into users (id, external_id, email, display_name) values (gen_random_uuid(), $1, $2, $3) returning id", sub, email, name).Scan(&userID)
				if err != nil {
					c.AbortWithStatusJSON(500, gin.H{"error": "user create"})
					return
				}
				mail = email
				displayName = name
			} else {
				c.AbortWithStatusJSON(500, gin.H{"error": "user lookup"})
				return
			}
		}
		rows, err := a.db.Query(ctx, "select r.name from user_roles ur join roles r on ur.role_id=r.id where ur.user_id=$1", userID)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": "role lookup"})
			return
		}
		defer rows.Close()
		roleSet := map[string]struct{}{}
		for rows.Next() {
			var role string
			if err := rows.Scan(&role); err == nil {
				roleSet[role] = struct{}{}
			}
		}
		if gc := a.cfg.OIDCGroupClaim; gc != "" {
			if v, ok := claims[gc]; ok {
				switch g := v.(type) {
				case []interface{}:
					for _, it := range g {
						if s, ok := it.(string); ok {
							roleSet[s] = struct{}{}
						}
					}
				case []string:
					for _, s := range g {
						roleSet[s] = struct{}{}
					}
				case string:
					roleSet[g] = struct{}{}
				}
			}
		}
		roles := make([]string, 0, len(roleSet))
		for r := range roleSet {
			roles = append(roles, r)
		}
		authUser := AuthUser{ID: userID, ExternalID: sub, Email: mail, DisplayName: displayName, Roles: roles}
		c.Set("user", authUser)
		c.Next()
	}
}

func seedLocalAdmin(ctx context.Context, db *pgxpool.Pool) error {
	var exists bool
	if err := db.QueryRow(ctx, "select exists(select 1 from users where lower(username)='admin')").Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	pw := os.Getenv("ADMIN_PASSWORD")
	if pw == "" {
		// Generate a secure random password if not set
		const pwLen = 16
		b := make([]byte, pwLen)
		if _, err := rand.Read(b); err != nil {
			return fmt.Errorf("failed to generate random admin password: %w", err)
		}
		pw = hex.EncodeToString(b)
		log.Warn().Str("username", "admin").Str("password", pw).Msg("No ADMIN_PASSWORD set, generated random admin password (dev only)")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	var uid string
	if err := db.QueryRow(ctx, "insert into users (id, username, email, display_name, password_hash) values (gen_random_uuid(), 'admin', 'admin@example.com', 'Admin', $1) returning id", string(hash)).Scan(&uid); err != nil {
		return err
	}
	// Grant admin and agent roles
	_, _ = db.Exec(ctx, `insert into user_roles (user_id, role_id)
select $1, r.id from roles r where r.name in ('agent','admin') on conflict do nothing`, uid)
	log.Info().Str("username", "admin").Msg("seeded local admin user (dev)")
	return nil
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (a *App) login(c *gin.Context) {
	if a.cfg.AuthMode != "local" {
		c.JSON(400, gin.H{"error": "login disabled"})
		return
	}
	var in loginReq
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	var id, hash, email, displayName string
	err := a.db.QueryRow(ctx, "select id, coalesce(password_hash,''), coalesce(email,''), coalesce(display_name,'') from users where lower(username)=lower($1)", in.Username).Scan(&id, &hash, &email, &displayName)
	if err != nil || id == "" || hash == "" {
		c.JSON(401, gin.H{"error": "invalid credentials"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		c.JSON(401, gin.H{"error": "invalid credentials"})
		return
	}
	// issue token
	claims := jwt.MapClaims{
		"sub":   id,
		"name":  displayName,
		"email": email,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
		"mode":  "local",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(a.cfg.AuthLocalSecret))
	if err != nil {
		c.JSON(500, gin.H{"error": "token"})
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("auth", s, 86400, "/", "", false, true)
	c.JSON(200, gin.H{"ok": true})
}

func (a *App) logout(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("auth", "", -1, "/", "", false, true)
	c.JSON(200, gin.H{"ok": true})
}

func (a *App) requireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := c.Get("user")
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthenticated"})
			return
		}
		user := u.(AuthUser)
		for _, r := range user.Roles {
			for _, want := range roles {
				if r == want {
					c.Next()
					return
				}
			}
		}
		c.AbortWithStatusJSON(403, gin.H{"error": "forbidden"})
	}
}

func (a *App) me(c *gin.Context) {
	u, ok := c.Get("user")
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "unauthenticated"})
		return
	}
	c.JSON(200, u)
}

type roleRequest struct {
	Role string `json:"role" binding:"required"`
}

func (a *App) listUserRoles(c *gin.Context) {
	ctx := c.Request.Context()
	uid := c.Param("id")
	rows, err := a.db.Query(ctx, "select r.name from user_roles ur join roles r on ur.role_id=r.id where ur.user_id=$1", uid)
	if err != nil {
		c.JSON(500, gin.H{"error": "role lookup"})
		return
	}
	defer rows.Close()
	roles := []string{}
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err == nil {
			roles = append(roles, role)
		}
	}
	c.JSON(200, roles)
}

func (a *App) addUserRole(c *gin.Context) {
	var in roleRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	uid := c.Param("id")
	_, err := a.db.Exec(ctx, `insert into user_roles (user_id, role_id)
        select $1, r.id from roles r where r.name=$2 on conflict do nothing`, uid, in.Role)
	if err != nil {
		c.JSON(500, gin.H{"error": "role add"})
		return
	}
	c.Status(201)
}

func (a *App) removeUserRole(c *gin.Context) {
	ctx := c.Request.Context()
	uid := c.Param("id")
	role := c.Param("role")
	_, err := a.db.Exec(ctx, `delete from user_roles where user_id=$1 and role_id in (select id from roles where name=$2)`, uid, role)
	if err != nil {
		c.JSON(500, gin.H{"error": "role remove"})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

// ===== Data structs =====
type Ticket struct {
	ID          string     `json:"id"`
	Number      string     `json:"number"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	RequesterID string     `json:"requester_id"`
	AssigneeID  *string    `json:"assignee_id,omitempty"`
	TeamID      *string    `json:"team_id,omitempty"`
	Priority    int16      `json:"priority"`
	Urgency     *int16     `json:"urgency,omitempty"`
	Category    *string    `json:"category,omitempty"`
	Subcategory *string    `json:"subcategory,omitempty"`
	Status      string     `json:"status"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	DueAt       *time.Time `json:"due_at,omitempty"`
	Source      string     `json:"source"`
	CustomJSON  any        `json:"custom_json"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	SLA         *SLAStatus `json:"sla,omitempty"`
}

type SLAStatus struct {
	PolicyID             string  `json:"policy_id"`
	ResponseElapsedMS    int64   `json:"response_elapsed_ms"`
	ResolutionElapsedMS  int64   `json:"resolution_elapsed_ms"`
	ResponseTargetMins   int     `json:"response_target_mins"`
	ResolutionTargetMins int     `json:"resolution_target_mins"`
	Paused               bool    `json:"paused"`
	Reason               *string `json:"reason,omitempty"`
}

type Comment struct {
	ID         string    `json:"id"`
	TicketID   string    `json:"ticket_id"`
	AuthorID   string    `json:"author_id"`
	BodyMD     string    `json:"body_md"`
	IsInternal bool      `json:"is_internal"`
	CreatedAt  time.Time `json:"created_at"`
}

// ===== Handlers =====
func (a *App) listTickets(c *gin.Context) {
	ctx := c.Request.Context()

	base := `
       select t.id, t.number, t.title, coalesce(t.description,''), t.requester_id, t.assignee_id, t.team_id, t.priority,
              t.urgency, t.category, t.subcategory, t.status, t.scheduled_at, t.due_at, t.source, t.custom_json, t.created_at, t.updated_at,
              sc.policy_id, sc.response_elapsed_ms, sc.resolution_elapsed_ms, sc.paused, sc.reason,
              sp.response_target_mins, sp.resolution_target_mins
       from tickets t
       left join ticket_sla_clocks sc on sc.ticket_id = t.id
       left join sla_policies sp on sp.id = sc.policy_id`

	where := []string{}
	args := []any{}
	i := 1

	if v := c.Query("status"); v != "" {
		where = append(where, fmt.Sprintf("t.status = $%d", i))
		args = append(args, v)
		i++
	}
	if v := c.Query("priority"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			where = append(where, fmt.Sprintf("t.priority = $%d", i))
			args = append(args, p)
			i++
		}
	}
	if v := c.Query("team"); v != "" {
		where = append(where, fmt.Sprintf("t.team_id = $%d", i))
		args = append(args, v)
		i++
	}
	if v := c.Query("assignee"); v != "" {
		where = append(where, fmt.Sprintf("t.assignee_id = $%d", i))
		args = append(args, v)
		i++
	}
	if v := strings.TrimSpace(c.Query("search")); v != "" {
		where = append(where, fmt.Sprintf("to_tsvector('english', coalesce(t.title,'') || ' ' || coalesce(t.description,'')) @@ websearch_to_tsquery('english', $%d)", i))
		args = append(args, v)
		i++
	}

	if len(where) > 0 {
		base += "\n       where " + strings.Join(where, " and ")
	}

	base += "\n       order by t.created_at desc\n       limit 200"

	rows, err := a.db.Query(ctx, base, args...)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []Ticket{}
	for rows.Next() {
		var t Ticket
		var customJSON []byte
		var policyID *string
		var respMS, resMS *int64
		var paused *bool
		var reason *string
		var respTarget, resTarget *int32
		if err := rows.Scan(&t.ID, &t.Number, &t.Title, &t.Description, &t.RequesterID, &t.AssigneeID, &t.TeamID,
			&t.Priority, &t.Urgency, &t.Category, &t.Subcategory, &t.Status, &t.ScheduledAt, &t.DueAt, &t.Source, &customJSON, &t.CreatedAt, &t.UpdatedAt,
			&policyID, &respMS, &resMS, &paused, &reason, &respTarget, &resTarget); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		t.CustomJSON = jsonRaw(customJSON)
		if policyID != nil {
			t.SLA = &SLAStatus{
				PolicyID:             *policyID,
				ResponseElapsedMS:    derefInt64(respMS),
				ResolutionElapsedMS:  derefInt64(resMS),
				ResponseTargetMins:   int(derefInt32(respTarget)),
				ResolutionTargetMins: int(derefInt32(resTarget)),
				Paused:               paused != nil && *paused,
				Reason:               reason,
			}
		}
		out = append(out, t)
	}
	c.JSON(200, out)
}

type jsonRaw []byte

func (j jsonRaw) MarshalJSON() ([]byte, error) {
	if j == nil || len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

func derefInt64(p *int64) int64 {
	if p != nil {
		return *p
	}
	return 0
}

func derefInt32(p *int32) int32 {
	if p != nil {
		return *p
	}
	return 0
}

type createTicketReq struct {
	Title       string         `json:"title" binding:"required,min=3"`
	Description string         `json:"description"`
	RequesterID string         `json:"requester_id" binding:"required"`
	Priority    int16          `json:"priority" binding:"required,min=1,max=4"`
	Urgency     *int16         `json:"urgency" binding:"omitempty,min=1,max=4"`
	Category    *string        `json:"category"`
	Subcategory *string        `json:"subcategory" binding:"omitempty,min=1"`
	CustomJSON  map[string]any `json:"custom_json"`
}

func (a *App) createTicket(c *gin.Context) {
	var in createTicketReq
	if err := c.ShouldBindJSON(&in); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			errs := make(map[string]string)
			typ := reflect.TypeOf(in)
			for _, fe := range ve {
				field, _ := typ.FieldByName(fe.StructField())
				name := strings.Split(field.Tag.Get("json"), ",")[0]
				if name == "" {
					name = strings.ToLower(fe.StructField())
				}
				errs[name] = fe.Error()
			}
			c.JSON(400, gin.H{"errors": errs})
			return
		}
		var ute *json.UnmarshalTypeError
		if errors.As(err, &ute) {
			c.JSON(400, gin.H{"errors": gin.H{ute.Field: ute.Error()}})
			return
		}
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	u := c.MustGet("user").(AuthUser)
	ctx := c.Request.Context()
	var id, number, status string
	status = "New"
	err := a.db.QueryRow(ctx, `
        insert into tickets (id, number, title, description, requester_id, priority, urgency, category, subcategory, status, source, custom_json)
        values (gen_random_uuid(), 'TKT-' || to_char(nextval('ticket_seq'), 'FM000000'), $1, $2, $3, $4, $5, $6, $7, $8, 'web', coalesce($9::jsonb,'{}'::jsonb))
        returning id, number, status`,
		in.Title, in.Description, in.RequesterID, in.Priority, in.Urgency, in.Category, in.Subcategory, status, toJSON(in.CustomJSON)).Scan(&id, &number, &status)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	a.addStatusHistory(ctx, id, "", status, u.ID)
	a.audit(c, "user", u.ID, "ticket", id, "create", gin.H{"title": in.Title, "requester_id": in.RequesterID})
	var requesterEmail string
	_ = a.db.QueryRow(ctx, "select coalesce(email,'') from users where id=$1", in.RequesterID).Scan(&requesterEmail)
	if requesterEmail != "" {
		a.enqueueEmail(ctx, requesterEmail, "ticket_created", gin.H{"Number": number})
	}
	handlers.PublishEvent(ctx, a.q, handlers.Event{Type: "ticket_created", Data: map[string]interface{}{"id": id}})
	c.JSON(201, gin.H{"id": id, "number": number, "status": status})
}

func toJSON(v interface{}) *string {
	if v == nil {
		return nil
	}
	b, _ := json.Marshal(v)
	s := string(b)
	return &s
}

func (a *App) audit(c *gin.Context, actorType, actorID, entityType, entityID, action string, diff interface{}) {
	ctx := c.Request.Context()
	var prevHash *string
	_ = a.db.QueryRow(ctx, "select hash from audit_events order by at desc limit 1").Scan(&prevHash)
	diffJSON, _ := json.Marshal(diff)
	data := append([]byte{}, diffJSON...)
	if prevHash != nil {
		data = append(data, []byte(*prevHash)...)
	}
	h := sha256.Sum256(data)
	hash := fmt.Sprintf("%x", h[:])
	_, _ = a.db.Exec(ctx, `insert into audit_events (actor_type, actor_id, entity_type, entity_id, action, diff_json, ip, ua, hash, prev_hash)
        values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		actorType, actorID, entityType, entityID, action, diffJSON, c.ClientIP(), c.Request.UserAgent(), hash, prevHash)
}

func (a *App) enqueueEmail(ctx context.Context, to, template string, data interface{}) {
	if a.q == nil {
		return
	}
	job := struct {
		Type string      `json:"type"`
		Data interface{} `json:"data"`
	}{
		Type: "send_email",
		Data: struct {
			To       string      `json:"to"`
			Template string      `json:"template"`
			Data     interface{} `json:"data"`
		}{to, template, data},
	}
	b, err := json.Marshal(job)
	if err != nil {
		log.Error().Err(err).Msg("marshal email job")
		return
	}
	if err := a.q.RPush(ctx, "jobs", b).Err(); err != nil {
		log.Error().Err(err).Msg("enqueue job")
	}
	size, _ := a.q.LLen(ctx, "jobs").Result()
	handlers.PublishEvent(ctx, a.q, handlers.Event{Type: "queue_changed", Data: map[string]interface{}{"size": size}})
}

func (a *App) addStatusHistory(ctx context.Context, ticketID, from, to, actorID string) {
	_, _ = a.db.Exec(ctx, `insert into ticket_status_history (ticket_id, from_status, to_status, actor_id) values ($1,$2,$3,$4)`, ticketID, nullable(from), to, nullable(actorID))
}

func nullable(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (a *App) getTicket(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()
	var t Ticket
	var customJSON []byte
	var policyID *string
	var respMS, resMS *int64
	var paused *bool
	var reason *string
	var respTarget, resTarget *int32
	err := a.db.QueryRow(ctx, `
       select t.id, t.number, t.title, coalesce(t.description,''), t.requester_id, t.assignee_id, t.team_id, t.priority,
              t.urgency, t.category, t.subcategory, t.status, t.scheduled_at, t.due_at, t.source, t.custom_json, t.created_at, t.updated_at,
              sc.policy_id, sc.response_elapsed_ms, sc.resolution_elapsed_ms, sc.paused, sc.reason,
              sp.response_target_mins, sp.resolution_target_mins
       from tickets t
       left join ticket_sla_clocks sc on sc.ticket_id = t.id
       left join sla_policies sp on sp.id = sc.policy_id
       where t.id=$1`, id).
		Scan(&t.ID, &t.Number, &t.Title, &t.Description, &t.RequesterID, &t.AssigneeID, &t.TeamID, &t.Priority, &t.Urgency,
			&t.Category, &t.Subcategory, &t.Status, &t.ScheduledAt, &t.DueAt, &t.Source, &customJSON, &t.CreatedAt, &t.UpdatedAt,
			&policyID, &respMS, &resMS, &paused, &reason, &respTarget, &resTarget)
	if err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	t.CustomJSON = jsonRaw(customJSON)
	if policyID != nil {
		t.SLA = &SLAStatus{
			PolicyID:             *policyID,
			ResponseElapsedMS:    derefInt64(respMS),
			ResolutionElapsedMS:  derefInt64(resMS),
			ResponseTargetMins:   int(derefInt32(resTarget)),
			ResolutionTargetMins: int(derefInt32(respTarget)),
			Paused:               paused != nil && *paused,
			Reason:               reason,
		}
	}
	c.JSON(200, t)
}

type patchTicketReq struct {
	Status      *string     `json:"status"`
	AssigneeID  *string     `json:"assignee_id"`
	Priority    *int16      `json:"priority"`
	Urgency     *int16      `json:"urgency"`
	ScheduledAt *time.Time  `json:"scheduled_at"`
	DueAt       *time.Time  `json:"due_at"`
	CustomJSON  interface{} `json:"custom_json"`
}

func (a *App) updateTicket(c *gin.Context) {
	id := c.Param("id")
	var in patchTicketReq
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	u := c.MustGet("user").(AuthUser)
	ctx := c.Request.Context()
	var oldStatus, number, requesterEmail string
	_ = a.db.QueryRow(ctx, "select status, number, (select coalesce(email,'') from users where id=requester_id) from tickets where id=$1", id).Scan(&oldStatus, &number, &requesterEmail)

	_, err := a.db.Exec(ctx, `
        update tickets set
            status = coalesce($1, status),
            assignee_id = coalesce($2::uuid, assignee_id),
            priority = coalesce($3, priority),
            urgency = coalesce($4, urgency),
            scheduled_at = coalesce($5, scheduled_at),
            due_at = coalesce($6, due_at),
            custom_json = coalesce($7::jsonb, custom_json),
            updated_at = now()
        where id=$8
    `, in.Status, in.AssigneeID, in.Priority, in.Urgency, in.ScheduledAt, in.DueAt, toJSON(in.CustomJSON), id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if in.Status != nil && oldStatus != *in.Status {
		a.addStatusHistory(ctx, id, oldStatus, *in.Status, u.ID)
	}
	diff := gin.H{}
	if in.Status != nil {
		diff["status"] = *in.Status
	}
	if in.AssigneeID != nil {
		diff["assignee_id"] = *in.AssigneeID
	}
	if in.Priority != nil {
		diff["priority"] = *in.Priority
	}
	if in.Urgency != nil {
		diff["urgency"] = *in.Urgency
	}
	if in.ScheduledAt != nil {
		diff["scheduled_at"] = in.ScheduledAt
	}
	if in.DueAt != nil {
		diff["due_at"] = in.DueAt
	}
	if in.CustomJSON != nil {
		diff["custom_json"] = in.CustomJSON
	}
	a.audit(c, "user", u.ID, "ticket", id, "update", diff)
	if requesterEmail != "" {
		if in.Status != nil && *in.Status == "Resolved" && oldStatus != *in.Status {
			b := make([]byte, 16)
			if _, err := rand.Read(b); err == nil {
				token := hex.EncodeToString(b)
				_, _ = a.db.Exec(ctx, `update tickets set csat_token=$1, csat_score=null where id=$2`, token, id)
				data := gin.H{
					"Number":  number,
					"GoodURL": fmt.Sprintf("/csat/%s?score=good", token),
					"BadURL":  fmt.Sprintf("/csat/%s?score=bad", token),
				}
				a.enqueueEmail(ctx, requesterEmail, "ticket_resolved", data)
			}
		} else {
			a.enqueueEmail(ctx, requesterEmail, "ticket_updated", gin.H{"Number": number})
		}
	}
	handlers.PublishEvent(ctx, a.q, handlers.Event{Type: "ticket_updated", Data: map[string]interface{}{"id": id}})
	c.JSON(200, gin.H{"ok": true})
}

func (a *App) submitCSAT(c *gin.Context) {
	token := c.Param("token")
	score := c.Query("score")
	if score != "good" && score != "bad" {
		c.JSON(400, gin.H{"error": "invalid score"})
		return
	}
	ctx := c.Request.Context()
	res, err := a.db.Exec(ctx, `update tickets set csat_score=$1, csat_token=null where csat_token=$2 and csat_score is null`, score, token)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(404, gin.H{"error": "invalid token"})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

type commentReq struct {
	BodyMD     string `json:"body_md" binding:"required"`
	IsInternal bool   `json:"is_internal"`
	// AuthorID is ignored server-side; author is derived from authenticated user
	AuthorID string `json:"author_id,omitempty"`
}

func (a *App) listComments(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()
	rows, err := a.db.Query(ctx, `
       select id, ticket_id, author_id, body_md, is_internal, created_at
       from ticket_comments
       where ticket_id=$1 and is_internal=false
       order by created_at
    `, id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var cs []Comment
	for rows.Next() {
		var cm Comment
		if err := rows.Scan(&cm.ID, &cm.TicketID, &cm.AuthorID, &cm.BodyMD, &cm.IsInternal, &cm.CreatedAt); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		cs = append(cs, cm)
	}
	if err := rows.Err(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, cs)
}

func (a *App) addComment(c *gin.Context) {
	id := c.Param("id")
	var in commentReq
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	u := c.MustGet("user").(AuthUser)
	ctx := c.Request.Context()
	var cid string
	err := a.db.QueryRow(ctx, `
        insert into ticket_comments (id, ticket_id, author_id, body_md, is_internal)
        values (gen_random_uuid(), $1, $2, $3, $4) returning id
    `, id, u.ID, in.BodyMD, in.IsInternal).Scan(&cid)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	a.audit(c, "user", u.ID, "ticket", id, "comment_add", gin.H{"comment_id": cid, "author_id": u.ID})
	c.JSON(201, gin.H{"id": cid})
}

// ===== Attachments =====
func (a *App) uploadAttachment(c *gin.Context) {
	if a.m == nil {
		c.JSON(500, gin.H{"error": "minio not configured"})
		return
	}
	ticketID := c.Param("id")
	u := c.MustGet("user").(AuthUser)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()
	objectKey := uuid.New().String()
	_, err = a.m.PutObject(c.Request.Context(), a.cfg.MinIOBucket, objectKey, file, header.Size, minio.PutObjectOptions{ContentType: header.Header.Get("Content-Type")})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	var id string
	err = a.db.QueryRow(ctx, `insert into attachments (id, ticket_id, uploader_id, object_key, filename, bytes, mime) values ($1,$2,$3,$4,$5,$6,$7) returning id`,
		objectKey, ticketID, u.ID, objectKey, header.Filename, header.Size, header.Header.Get("Content-Type")).Scan(&id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	a.audit(c, "user", u.ID, "ticket", ticketID, "attachment_add", gin.H{"attachment_id": id})
	c.JSON(201, gin.H{"id": id})
}

func (a *App) listAttachments(c *gin.Context) {
	ticketID := c.Param("id")
	ctx := c.Request.Context()
	rows, err := a.db.Query(ctx, "select id, filename, bytes, mime, created_at from attachments where ticket_id=$1", ticketID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	type Att struct {
		ID       string    `json:"id"`
		Filename string    `json:"filename"`
		Bytes    int64     `json:"bytes"`
		Mime     *string   `json:"mime"`
		Created  time.Time `json:"created_at"`
	}
	out := []Att{}
	for rows.Next() {
		var a Att
		if err := rows.Scan(&a.ID, &a.Filename, &a.Bytes, &a.Mime, &a.Created); err == nil {
			out = append(out, a)
		}
	}
	c.JSON(200, out)
}

// getAttachment streams the attachment file or redirects to object storage if configured.
func (a *App) getAttachment(c *gin.Context) {
	ticketID := c.Param("id")
	attID := c.Param("attID")
	ctx := c.Request.Context()
	var objectKey, filename string
	var mime *string
	err := a.db.QueryRow(ctx, "select object_key, filename, mime from attachments where id=$1 and ticket_id=$2", attID, ticketID).Scan(&objectKey, &filename, &mime)
	if err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	// If MinIO endpoint is configured, generate a presigned URL and redirect.
	if a.cfg.MinIOEndpoint != "" {
		if mc, ok := a.m.(*minio.Client); ok {
			url, err := mc.PresignedGetObject(ctx, a.cfg.MinIOBucket, objectKey, time.Minute, nil)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.Redirect(http.StatusFound, url.String())
			return
		}
	}
	// Serve from filesystem store when configured
	if a.cfg.FileStorePath != "" {
		dir := a.cfg.FileStorePath
		if a.cfg.MinIOBucket != "" {
			dir = dir + string(os.PathSeparator) + a.cfg.MinIOBucket
		}
		fp := dir + string(os.PathSeparator) + objectKey
		if mime != nil && *mime != "" {
			c.Header("Content-Type", *mime)
		} else {
			c.Header("Content-Type", "application/octet-stream")
		}
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.File(fp)
		return
	}
	c.JSON(500, gin.H{"error": "object store not configured"})
}

func (a *App) deleteAttachment(c *gin.Context) {
	if a.m == nil {
		c.JSON(500, gin.H{"error": "minio not configured"})
		return
	}
	ticketID := c.Param("id")
	attID := c.Param("attID")
	u := c.MustGet("user").(AuthUser)
	ctx := c.Request.Context()
	var objectKey string
	err := a.db.QueryRow(ctx, "select object_key from attachments where id=$1 and ticket_id=$2", attID, ticketID).Scan(&objectKey)
	if err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	_ = a.m.RemoveObject(ctx, a.cfg.MinIOBucket, objectKey, minio.RemoveObjectOptions{})
	_, err = a.db.Exec(ctx, "delete from attachments where id=$1", attID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	a.audit(c, "user", u.ID, "ticket", ticketID, "attachment_delete", gin.H{"attachment_id": attID})
	c.JSON(200, gin.H{"ok": true})
}

// ===== Watchers =====
type watcherReq struct {
	UserID string `json:"user_id" binding:"required"`
}

func (a *App) listWatchers(c *gin.Context) {
	ticketID := c.Param("id")
	ctx := c.Request.Context()
	rows, err := a.db.Query(ctx, "select user_id from ticket_watchers where ticket_id=$1", ticketID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err == nil {
			out = append(out, uid)
		}
	}
	c.JSON(200, out)
}

func (a *App) addWatcher(c *gin.Context) {
	ticketID := c.Param("id")
	var in watcherReq
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	u := c.MustGet("user").(AuthUser)
	ctx := c.Request.Context()
	_, err := a.db.Exec(ctx, "insert into ticket_watchers (ticket_id, user_id) values ($1,$2) on conflict do nothing", ticketID, in.UserID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	a.audit(c, "user", u.ID, "ticket", ticketID, "watcher_add", gin.H{"user_id": in.UserID})
	c.JSON(201, gin.H{"ok": true})
}

func (a *App) removeWatcher(c *gin.Context) {
	ticketID := c.Param("id")
	watcherID := c.Param("userID")
	u := c.MustGet("user").(AuthUser)
	ctx := c.Request.Context()
	_, err := a.db.Exec(ctx, "delete from ticket_watchers where ticket_id=$1 and user_id=$2", ticketID, watcherID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	a.audit(c, "user", u.ID, "ticket", ticketID, "watcher_remove", gin.H{"user_id": watcherID})
	c.JSON(200, gin.H{"ok": true})
}

// ===== Metrics =====

// metricsSLA returns SLA attainment statistics.
func (a *App) metricsSLA(c *gin.Context) {
	ctx := c.Request.Context()
	var met, total int
	err := a.db.QueryRow(ctx, `
               select
                       count(*) filter (where tsc.resolution_elapsed_ms <= sp.resolution_target_mins * 60000) as met,
                       count(*) as total
               from ticket_sla_clocks tsc
               join tickets t on t.id = tsc.ticket_id
               join sla_policies sp on sp.id = tsc.policy_id
               where t.status = 'Resolved'
       `).Scan(&met, &total)
	if err != nil {
		c.JSON(500, gin.H{"error": "sla query"})
		return
	}
	attainment := 0.0
	if total > 0 {
		attainment = float64(met) / float64(total)
	}
	c.JSON(200, gin.H{"total": total, "met": met, "sla_attainment": attainment})
}

// metricsResolution returns average resolution time in milliseconds.
func (a *App) metricsResolution(c *gin.Context) {
	ctx := c.Request.Context()
	var avg sql.NullFloat64
	err := a.db.QueryRow(ctx, `
               select avg(tsc.resolution_elapsed_ms)
               from ticket_sla_clocks tsc
               join tickets t on t.id = tsc.ticket_id
               where t.status = 'Resolved' and tsc.resolution_elapsed_ms > 0
       `).Scan(&avg)
	if err != nil {
		c.JSON(500, gin.H{"error": "resolution query"})
		return
	}
	c.JSON(200, gin.H{"avg_resolution_ms": avg.Float64})
}

// metricsTicketVolume returns ticket counts per day for the last 30 days.
func (a *App) metricsTicketVolume(c *gin.Context) {
	ctx := c.Request.Context()
	rows, err := a.db.Query(ctx, `
               select date_trunc('day', created_at)::date as day, count(*)
               from tickets
               group by day
               order by day desc
               limit 30
       `)
	if err != nil {
		c.JSON(500, gin.H{"error": "volume query"})
		return
	}
	defer rows.Close()
	type dayCount struct {
		Day   time.Time `json:"day"`
		Count int       `json:"count"`
	}
	out := []dayCount{}
	for rows.Next() {
		var dc dayCount
		if err := rows.Scan(&dc.Day, &dc.Count); err == nil {
			out = append(out, dc)
		}
	}
	c.JSON(200, gin.H{"daily": out})
}

// ===== Exports =====
type exportTicketsReq struct {
	IDs []string `json:"ids" binding:"required"`
}

func (a *App) exportTickets(c *gin.Context) {
	if a.m == nil {
		c.JSON(500, gin.H{"error": "minio not configured"})
		return
	}
	var in exportTicketsReq
	if err := c.ShouldBindJSON(&in); err != nil || len(in.IDs) == 0 {
		c.JSON(400, gin.H{"error": "ids required"})
		return
	}
	ctx := c.Request.Context()
	placeholders := make([]string, len(in.IDs))
	args := make([]any, len(in.IDs))
	for i, id := range in.IDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	q := fmt.Sprintf("select id, number, title, status, priority from tickets where id in (%s)", strings.Join(placeholders, ","))
	rows, err := a.db.Query(ctx, q, args...)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	buf := &bytes.Buffer{}
	w := csv.NewWriter(buf)
	_ = w.Write([]string{"id", "number", "title", "status", "priority"})
	for rows.Next() {
		var id, number, title, status string
		var priority int16
		if err := rows.Scan(&id, &number, &title, &status, &priority); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		_ = w.Write([]string{id, number, title, status, strconv.Itoa(int(priority))})
	}
	w.Flush()
	objectKey := uuid.New().String() + ".csv"
	_, err = a.m.PutObject(ctx, a.cfg.MinIOBucket, objectKey, bytes.NewReader(buf.Bytes()), int64(buf.Len()), minio.PutObjectOptions{ContentType: "text/csv"})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if mc, ok := a.m.(*minio.Client); ok {
		url, err := mc.PresignedGetObject(ctx, a.cfg.MinIOBucket, objectKey, time.Minute, nil)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"url": url.String()})
		return
	}
	scheme := "http"
	if a.cfg.MinIOUseSSL {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s/%s/%s", scheme, a.cfg.MinIOEndpoint, a.cfg.MinIOBucket, objectKey)
	c.JSON(200, gin.H{"url": url})
}
