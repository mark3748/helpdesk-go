package main

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Config struct {
	Addr          string
	DatabaseURL   string
	Env           string
	RedisAddr     string
	OIDCIssuer    string
	JWKSURL       string
	MinIOEndpoint string
	MinIOAccess   string
	MinIOSecret   string
	MinIOBucket   string
	MinIOUseSSL   bool
}

func getConfig() Config {
	_ = godotenv.Load()
	cfg := Config{
		Addr:          getEnv("ADDR", ":8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/helpdesk?sslmode=disable"),
		Env:           getEnv("ENV", "dev"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		OIDCIssuer:    getEnv("OIDC_ISSUER", ""),
		JWKSURL:       getEnv("OIDC_JWKS_URL", ""),
		MinIOEndpoint: getEnv("MINIO_ENDPOINT", ""),
		MinIOAccess:   getEnv("MINIO_ACCESS_KEY", ""),
		MinIOSecret:   getEnv("MINIO_SECRET_KEY", ""),
		MinIOBucket:   getEnv("MINIO_BUCKET", ""),
		MinIOUseSSL:   getEnv("MINIO_USE_SSL", "false") == "true",
	}
	return cfg
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type App struct {
	cfg  Config
	db   *pgxpool.Pool
	r    *gin.Engine
	jwks *keyfunc.JWKS
	m    *minio.Client
	q    *redis.Client
}

func main() {
	cfg := getConfig()
	if cfg.Env == "dev" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// DB connect
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("db connect")
	}
	defer pool.Close()

	// Migrate (embedded goose)
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal().Err(err).Msg("goose dialect")
	}
	// goose wants a *sql.DB normally; use pgx stdlib is more complex.
	// Simpler: run migrations via goose with connection string (driver name, DSN).
	if err := goose.UpContext(ctx, nil, "migrations",
		goose.WithAllowMissing(), goose.WithNoVersioning(), goose.WithRequireDB(false),
		goose.WithDriver("postgres", cfg.DatabaseURL)); err != nil {
		log.Fatal().Err(err).Msg("migrate up")
	}

	var jwks *keyfunc.JWKS
	if cfg.JWKSURL != "" {
		jwks, err = keyfunc.Get(cfg.JWKSURL, keyfunc.Options{})
		if err != nil {
			log.Fatal().Err(err).Msg("load jwks")
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

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Error().Err(err).Msg("redis ping")
	}

	a := &App{cfg: cfg, db: pool, r: gin.New(), jwks: jwks, m: mc, q: rdb}
	a.r.Use(gin.Recovery())
	a.r.Use(gin.Logger())
	a.routes()

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

	auth := a.r.Group("/")
	auth.Use(a.authMiddleware())
	auth.GET("/me", a.me)

	// Tickets
	auth.GET("/tickets", a.listTickets)
	auth.POST("/tickets", a.createTicket)
	auth.GET("/tickets/:id", a.getTicket)
	auth.PATCH("/tickets/:id", a.requireRole("agent"), a.updateTicket)
	auth.POST("/tickets/:id/comments", a.addComment)
	auth.GET("/tickets/:id/attachments", a.listAttachments)
	auth.POST("/tickets/:id/attachments", a.uploadAttachment)
	auth.DELETE("/tickets/:id/attachments/:attID", a.deleteAttachment)
	auth.GET("/tickets/:id/watchers", a.listWatchers)
	auth.POST("/tickets/:id/watchers", a.addWatcher)
	auth.DELETE("/tickets/:id/watchers/:userID", a.removeWatcher)
}

type AuthUser struct {
	ID          string   `json:"id"`
	ExternalID  string   `json:"external_id"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	Roles       []string `json:"roles"`
}

func (a *App) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.jwks == nil {
			c.AbortWithStatusJSON(500, gin.H{"error": "jwks not configured"})
			return
		}
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(401, gin.H{"error": "missing bearer token"})
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(tokenStr, a.jwks.Keyfunc)
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
		roles := []string{}
		for rows.Next() {
			var role string
			if err := rows.Scan(&role); err == nil {
				roles = append(roles, role)
			}
		}
		authUser := AuthUser{ID: userID, ExternalID: sub, Email: mail, DisplayName: displayName, Roles: roles}
		c.Set("user", authUser)
		c.Next()
	}
}

func (a *App) requireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := c.Get("user")
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthenticated"})
			return
		}
		user := u.(AuthUser)
		for _, r := range user.Roles {
			if r == role {
				c.Next()
				return
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
}

// ===== Handlers =====
func (a *App) listTickets(c *gin.Context) {
	ctx := c.Request.Context()
	rows, err := a.db.Query(ctx, `
        select id, number, title, coalesce(description,''), requester_id, assignee_id, team_id, priority,
               urgency, category, subcategory, status, scheduled_at, due_at, source, custom_json, created_at, updated_at
        from tickets
        order by created_at desc
        limit 200`)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []Ticket{}
	for rows.Next() {
		var t Ticket
		var customJSON []byte
		if err := rows.Scan(&t.ID, &t.Number, &t.Title, &t.Description, &t.RequesterID, &t.AssigneeID, &t.TeamID,
			&t.Priority, &t.Urgency, &t.Category, &t.Subcategory, &t.Status, &t.ScheduledAt, &t.DueAt, &t.Source, &customJSON, &t.CreatedAt, &t.UpdatedAt); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		t.CustomJSON = jsonRaw(customJSON)
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

type createTicketReq struct {
	Title       string      `json:"title" binding:"required,min=3"`
	Description string      `json:"description"`
	RequesterID string      `json:"requester_id" binding:"required"`
	Priority    int16       `json:"priority" binding:"required"`
	Urgency     *int16      `json:"urgency"`
	Category    *string     `json:"category"`
	Subcategory *string     `json:"subcategory"`
	CustomJSON  interface{} `json:"custom_json"`
}

func (a *App) createTicket(c *gin.Context) {
	var in createTicketReq
	if err := c.ShouldBindJSON(&in); err != nil {
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
	b, _ := json.Marshal(job)
	if err := a.q.RPush(ctx, "jobs", b).Err(); err != nil {
		log.Error().Err(err).Msg("enqueue job")
	}
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
	err := a.db.QueryRow(ctx, `
        select id, number, title, coalesce(description,''), requester_id, assignee_id, team_id, priority,
               urgency, category, subcategory, status, scheduled_at, due_at, source, custom_json, created_at, updated_at
        from tickets where id=$1`, id).
		Scan(&t.ID, &t.Number, &t.Title, &t.Description, &t.RequesterID, &t.AssigneeID, &t.TeamID, &t.Priority, &t.Urgency,
			&t.Category, &t.Subcategory, &t.Status, &t.ScheduledAt, &t.DueAt, &t.Source, &customJSON, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	t.CustomJSON = jsonRaw(customJSON)
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
		a.enqueueEmail(ctx, requesterEmail, "ticket_updated", gin.H{"Number": number})
	}
	c.JSON(200, gin.H{"ok": true})
}

type commentReq struct {
	BodyMD     string `json:"body_md" binding:"required"`
	IsInternal bool   `json:"is_internal"`
	AuthorID   string `json:"author_id" binding:"required"`
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
    `, id, in.AuthorID, in.BodyMD, in.IsInternal).Scan(&cid)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	a.audit(c, "user", u.ID, "ticket", id, "comment_add", gin.H{"comment_id": cid, "author_id": in.AuthorID})
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
