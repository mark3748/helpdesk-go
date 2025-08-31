package main

import (
	"context"
	"embed"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Config struct {
	Addr        string
	DatabaseURL string
	Env         string
	OIDCIssuer  string
	JWKSURL     string
}

func getConfig() Config {
	_ = godotenv.Load()
	cfg := Config{
		Addr:        getEnv("ADDR", ":8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/helpdesk?sslmode=disable"),
		Env:         getEnv("ENV", "dev"),
		OIDCIssuer:  getEnv("OIDC_ISSUER", ""),
		JWKSURL:     getEnv("OIDC_JWKS_URL", ""),
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

	a := &App{cfg: cfg, db: pool, r: gin.New(), jwks: jwks}
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
	ctx := c.Request.Context()
	// Simple patch (set coalesce to current values)
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
	ctx := c.Request.Context()
	_, err := a.db.Exec(ctx, `
        insert into ticket_comments (id, ticket_id, author_id, body_md, is_internal)
        values (gen_random_uuid(), $1, $2, $3, $4)
    `, id, in.AuthorID, in.BodyMD, in.IsInternal)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"ok": true})
}
