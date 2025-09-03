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
	exportspkg "github.com/mark3748/helpdesk-go/cmd/api/exports"
	metricspkg "github.com/mark3748/helpdesk-go/cmd/api/metrics"
	ticketspkg "github.com/mark3748/helpdesk-go/cmd/api/tickets"
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

	a.R.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	auth := a.R.Group("/")
	auth.Use(authpkg.Middleware(a))
	auth.GET("/me", authpkg.Me)
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
	auth.GET("/metrics/sla", authpkg.RequireRole("agent"), metricspkg.SLA(a))
	auth.GET("/metrics/resolution", authpkg.RequireRole("agent"), metricspkg.Resolution(a))
	auth.GET("/metrics/tickets", authpkg.RequireRole("agent"), metricspkg.TicketVolume(a))
	auth.POST("/exports/tickets", authpkg.RequireRole("agent"), exportspkg.Tickets(a))

	a.R.Run(cfg.Addr)
}
