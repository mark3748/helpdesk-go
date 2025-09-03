package main

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	attachmentspkg "github.com/mark3748/helpdesk-go/cmd/api/attachments"
	authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
	commentspkg "github.com/mark3748/helpdesk-go/cmd/api/comments"
	eventspkg "github.com/mark3748/helpdesk-go/cmd/api/events"
	exportspkg "github.com/mark3748/helpdesk-go/cmd/api/exports"
	handlers "github.com/mark3748/helpdesk-go/cmd/api/handlers"
	metricspkg "github.com/mark3748/helpdesk-go/cmd/api/metrics"
	migratepkg "github.com/mark3748/helpdesk-go/cmd/api/migrations"
	requesterspkg "github.com/mark3748/helpdesk-go/cmd/api/requesters"
	rolespkg "github.com/mark3748/helpdesk-go/cmd/api/roles"
	ticketspkg "github.com/mark3748/helpdesk-go/cmd/api/tickets"
	userspkg "github.com/mark3748/helpdesk-go/cmd/api/users"
	watcherspkg "github.com/mark3748/helpdesk-go/cmd/api/watchers"
)

func main() {
	cfg := apppkg.GetConfig()
	writer := io.Writer(os.Stdout)
	if cfg.Env == "dev" {
		writer = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}
	log.Logger = zerolog.New(writer).With().Timestamp().Logger()

	ctx := context.Background()

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("db connect")
	}
	defer db.Close()

	// Apply embedded migrations then init settings row
	if err := migratepkg.Apply(ctx, db); err != nil {
		log.Fatal().Err(err).Msg("migrate")
	}
	handlers.InitSettings(ctx, db, cfg.LogPath)

	var keyf jwt.Keyfunc
	if cfg.JWKSURL != "" {
		set, err := jwk.Fetch(ctx, cfg.JWKSURL)
		if err != nil {
			log.Fatal().Err(err).Msg("jwks fetch")
		}
		keyf = func(tk *jwt.Token) (any, error) {
			kid, _ := tk.Header["kid"].(string)
			if kid != "" {
				if key, ok := set.LookupKeyID(kid); ok {
					var pub any
					if err := key.Raw(&pub); err != nil {
						return nil, err
					}
					return pub, nil
				}
			}
			it := set.Iterate(ctx)
			if it.Next(ctx) {
				pair := it.Pair()
				if key, ok := pair.Value.(jwk.Key); ok {
					var pub any
					if err := key.Raw(&pub); err != nil {
						return nil, err
					}
					return pub, nil
				}
			}
			return nil, jwt.ErrTokenSignatureInvalid
		}
	}

	var store apppkg.ObjectStore
	if cfg.FileStorePath != "" {
		store = &apppkg.FsObjectStore{Base: cfg.FileStorePath}
	} else if cfg.MinIOEndpoint != "" {
		mc, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.MinIOAccess, cfg.MinIOSecret, ""),
			Secure: cfg.MinIOUseSSL,
		})
		if err != nil {
			log.Error().Err(err).Msg("minio init")
		} else {
			store = mc
		}
	}

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Error().Err(err).Msg("redis ping failed")
	}
	defer rdb.Close()

	a := apppkg.NewApp(cfg, db, keyf, store, rdb)

	// Serve OpenAPI spec and Swagger UI
	a.R.GET("/openapi.yaml", func(c *gin.Context) {
		// Resolve spec path: env override, repo-local, then image path
		candidates := []string{}
		if cfg.OpenAPISpecPath != "" {
			candidates = append(candidates, cfg.OpenAPISpecPath)
		}
		candidates = append(candidates, "docs/openapi.yaml", "/opt/helpdesk/docs/openapi.yaml")
		var content []byte
		for _, p := range candidates {
			if p == "" {
				continue
			}
			if b, err := os.ReadFile(p); err == nil {
				content = b
				break
			}
		}
		if len(content) == 0 {
			c.AbortWithStatusJSON(404, gin.H{"error": "openapi not found"})
			return
		}
		c.Data(200, "application/yaml", content)
	})
	// Conditionally serve local swagger assets if present; fall back to CDN via /docs page
	if _, err := os.Stat("/opt/helpdesk/swagger"); err == nil {
		a.R.Static("/swagger", "/opt/helpdesk/swagger")
	}
	a.R.GET("/docs", func(c *gin.Context) {
		html := `<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>Helpdesk API Docs</title>
    <link rel="stylesheet" href="/swagger/swagger-ui.css" />
    <style>body{margin:0} .swagger-ui .topbar{display:none}</style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="/swagger/swagger-ui-bundle.js"></script>
    <script src="/swagger/swagger-ui-standalone-preset.js"></script>
    <script>
      (function(){
        function boot(){
          if (!window.SwaggerUIBundle){ setTimeout(boot, 50); return }
          window.ui = SwaggerUIBundle({
            url: '/openapi.yaml',
            dom_id: '#swagger-ui',
            presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
            layout: 'StandaloneLayout'
          });
        }
        boot();
      })();
    </script>
  </body>
</html>`
		c.Data(200, "text/html; charset=utf-8", []byte(html))
	})

	// Public and authenticated API endpoints are mounted under /api
	api := a.R.Group("/api")
	api.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	// Basic auth endpoints
	api.POST("/login", authpkg.Login(a))
	api.POST("/logout", authpkg.Logout())

	// Public SSE (requires auth cookie; still under auth group)
	// NOTE: Expose under authenticated group to include auth middleware
	// so the cookie is validated before streaming.
	// We'll attach it below under auth.

	auth := api.Group("/")
	auth.Use(authpkg.Middleware(a))

	// Settings/admin endpoints
	// Keep GetSettings visible to authenticated users; restrict writes to admin
	// to align with internal UI expectations.
	auth.GET("/settings", handlers.GetSettings(db))
	auth.POST("/settings/oidc", authpkg.RequireRole("admin"), handlers.SaveOIDCSettings(db))
	auth.POST("/settings/storage", authpkg.RequireRole("admin"), handlers.SaveStorageSettings(db))
	auth.POST("/settings/mail", authpkg.RequireRole("admin"), handlers.SaveMailSettings(db))
	auth.POST("/test-connection", authpkg.RequireRole("admin"), handlers.TestConnection(db))
	auth.GET("/events", eventspkg.Stream(a))
	auth.GET("/me", authpkg.Me)
	auth.POST("/requesters", requesterspkg.Create(a))
	auth.GET("/requesters/:id", requesterspkg.Get(a))
	auth.PATCH("/requesters/:id", requesterspkg.Update(a))
	auth.GET("/tickets", ticketspkg.List(a))
	auth.POST("/tickets", ticketspkg.Create(a))
	auth.GET("/tickets/:id", ticketspkg.Get(a))
	auth.PATCH("/tickets/:id", authpkg.RequireRole("agent", "manager"), ticketspkg.Update(a))
	auth.GET("/tickets/:id/comments", commentspkg.List(a))
	auth.POST("/tickets/:id/comments", commentspkg.Add(a))
	auth.GET("/tickets/:id/attachments", attachmentspkg.List(a))
	auth.POST("/tickets/:id/attachments", attachmentspkg.Upload(a))
	auth.GET("/tickets/:id/attachments/:attID", attachmentspkg.Get(a))
	auth.DELETE("/tickets/:id/attachments/:attID", attachmentspkg.Delete(a))
	auth.GET("/tickets/:id/watchers", watcherspkg.List(a))
	auth.POST("/tickets/:id/watchers", watcherspkg.Add(a))
	auth.DELETE("/tickets/:id/watchers/:userID", watcherspkg.Remove(a))
	// Metrics and analytics
	auth.GET("/metrics/agent", authpkg.RequireRole("agent"), metricspkg.Agent(a))
	auth.GET("/metrics/manager", authpkg.RequireRole("manager"), metricspkg.Manager(a))
	auth.GET("/metrics/sla", authpkg.RequireRole("agent"), metricspkg.SLA(a))
	auth.GET("/metrics/resolution", authpkg.RequireRole("agent"), metricspkg.Resolution(a))
	auth.GET("/metrics/tickets", authpkg.RequireRole("agent"), metricspkg.TicketVolume(a))
	auth.POST("/exports/tickets", authpkg.RequireRole("agent"), exportspkg.Tickets(a))

	// Users listing (agent, manager, admin) and role management (admin only)
	auth.GET("/users", authpkg.RequireRole("agent", "manager", "admin"), userspkg.List(a))
	auth.GET("/users/:id", authpkg.RequireRole("agent", "manager", "admin"), userspkg.Get(a))
	auth.POST("/users", authpkg.RequireRole("admin"), userspkg.CreateLocal(a))
	auth.GET("/users/:id/roles", authpkg.RequireRole("admin"), authpkg.ListUserRoles(a))
	auth.POST("/users/:id/roles", authpkg.RequireRole("admin"), authpkg.AddUserRole(a))
	auth.DELETE("/users/:id/roles/:role", authpkg.RequireRole("admin"), authpkg.RemoveUserRole(a))
	auth.GET("/roles", authpkg.RequireRole("admin"), rolespkg.List(a))

	// Current user settings
	auth.GET("/me/profile", userspkg.GetProfile(a))
	auth.PATCH("/me/profile", userspkg.UpdateProfile(a))
	auth.POST("/me/password", userspkg.ChangePassword(a))

	a.R.Run(cfg.Addr)
}
