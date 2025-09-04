package app

import (
    "context"
    "io"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "net/url"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
    "github.com/minio/minio-go/v7"
    "github.com/redis/go-redis/v9"
    "golang.org/x/time/rate"
)

// Config holds API configuration values.
type Config struct {
	Addr           string
	DatabaseURL    string
	Env            string
	RedisAddr      string
	OIDCIssuer     string
	JWKSURL        string
	OIDCGroupClaim string
	// Optional audience validation for OIDC tokens
	OIDCAudience       string
	// Optional leeway for JWT time-based claims validation
	JWTClockSkewSeconds int
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
	AdminPassword   string
	// Filesystem object store for dev/local
	FileStorePath   string
	OpenAPISpecPath string
	LogPath         string
	RateLimitRPS    float64
	RateLimitBurst  int
}

// GetEnv returns the environment variable value or default.
func GetEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// GetConfig builds Config from environment.
func GetConfig() Config {
	cfg := Config{
		Addr:            GetEnv("ADDR", ":8080"),
		DatabaseURL:     GetEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/helpdesk?sslmode=disable"),
		Env:             GetEnv("ENV", "dev"),
		RedisAddr:       GetEnv("REDIS_ADDR", "localhost:6379"),
		OIDCIssuer:      GetEnv("OIDC_ISSUER", ""),
		JWKSURL:         GetEnv("OIDC_JWKS_URL", ""),
		OIDCGroupClaim:  GetEnv("OIDC_GROUP_CLAIM", "groups"),
		OIDCAudience:    GetEnv("OIDC_AUDIENCE", ""),
		MinIOEndpoint:   GetEnv("MINIO_ENDPOINT", ""),
		MinIOAccess:     GetEnv("MINIO_ACCESS_KEY", ""),
		MinIOSecret:     GetEnv("MINIO_SECRET_KEY", ""),
		MinIOBucket:     GetEnv("MINIO_BUCKET", "attachments"),
		MinIOUseSSL:     GetEnv("MINIO_USE_SSL", "false") == "true",
		TestBypassAuth:  GetEnv("TEST_BYPASS_AUTH", "false") == "true",
		AuthMode:        GetEnv("AUTH_MODE", "oidc"),
		AuthLocalSecret: GetEnv("AUTH_LOCAL_SECRET", ""),
		AdminPassword:   GetEnv("ADMIN_PASSWORD", "admin"),
		FileStorePath:   GetEnv("FILESTORE_PATH", ""),
		OpenAPISpecPath: GetEnv("OPENAPI_SPEC_PATH", ""),
		LogPath:         GetEnv("LOG_PATH", "/config/logs"),
	}
	if v, err := strconv.ParseFloat(GetEnv("RATE_LIMIT_RPS", "0"), 64); err == nil {
		cfg.RateLimitRPS = v
	}
	if v, err := strconv.Atoi(GetEnv("RATE_LIMIT_BURST", "0")); err == nil {
		cfg.RateLimitBurst = v
	}
	if v, err := strconv.Atoi(GetEnv("JWT_CLOCK_SKEW_SECONDS", "0")); err == nil {
		cfg.JWTClockSkewSeconds = v
	}
	return cfg
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
type FsObjectStore struct {
    Base string
}

func (f *FsObjectStore) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
    _ = ctx
    // Clean and constrain paths within base to prevent traversal
    base := filepath.Clean(f.Base)
    dir := base
    if bucketName != "" {
        dir = filepath.Join(base, bucketName)
    }
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return minio.UploadInfo{}, err
    }
    fp := filepath.Join(dir, objectName)
    clean := filepath.Clean(fp)
    // Ensure the final path stays within the base directory
    if !strings.HasPrefix(clean, dir+string(os.PathSeparator)) && clean != dir {
        return minio.UploadInfo{}, os.ErrPermission
    }
    tmp := clean + ".tmp"
    out, err := os.Create(tmp)
    if err != nil {
        return minio.UploadInfo{}, err
    }
    defer out.Close()
    if _, err := io.Copy(out, reader); err != nil {
        _ = os.Remove(tmp)
        return minio.UploadInfo{}, err
    }
    if err := os.Rename(tmp, clean); err != nil {
        return minio.UploadInfo{}, err
    }
    return minio.UploadInfo{Bucket: bucketName, Key: objectName, Size: objectSize}, nil
}

func (f *FsObjectStore) RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error {
    _ = ctx
    _ = opts
    base := filepath.Clean(f.Base)
    dir := base
    if bucketName != "" {
        dir = filepath.Join(base, bucketName)
    }
    fp := filepath.Join(dir, objectName)
    clean := filepath.Clean(fp)
    if !strings.HasPrefix(clean, dir+string(os.PathSeparator)) && clean != dir {
        return os.ErrPermission
    }
    return os.Remove(clean)
}

// PresignedPutObject is not supported for the filesystem store.
func (f *FsObjectStore) PresignedPutObject(ctx context.Context, bucketName, objectName string, expiry time.Duration) (*url.URL, error) {
    _ = ctx
    return nil, os.ErrPermission
}

// StatObject returns basic info for a stored object.
func (f *FsObjectStore) StatObject(ctx context.Context, bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error) {
    _ = ctx
    _ = opts
    base := filepath.Clean(f.Base)
    dir := base
    if bucketName != "" {
        dir = filepath.Join(base, bucketName)
    }
    fp := filepath.Join(dir, objectName)
    clean := filepath.Clean(fp)
    if !strings.HasPrefix(clean, dir+string(os.PathSeparator)) && clean != dir {
        return minio.ObjectInfo{}, os.ErrPermission
    }
    fi, err := os.Stat(clean)
    if err != nil {
        return minio.ObjectInfo{}, err
    }
    return minio.ObjectInfo{Key: objectName, Size: fi.Size()}, nil
}

// App wires dependencies and the Gin router.
type App struct {
	Cfg  Config
	DB   DB
	R    *gin.Engine
	Keyf jwt.Keyfunc
	M    ObjectStore
	Q    *redis.Client
}

// NewApp constructs an App with injected dependencies.
func NewApp(cfg Config, db DB, keyf jwt.Keyfunc, store ObjectStore, q *redis.Client) *App {
	a := &App{Cfg: cfg, DB: db, R: gin.New(), Keyf: keyf, M: store, Q: q}
	a.R.Use(gin.Recovery())
	a.R.Use(RequestID())
	if cfg.RateLimitRPS > 0 && cfg.RateLimitBurst > 0 {
		rl := rate.NewLimiter(rate.Limit(cfg.RateLimitRPS), cfg.RateLimitBurst)
		a.R.Use(RateLimit(rl))
	}
	a.R.Use(Logger())
	a.R.Use(Errors())
	return a
}
