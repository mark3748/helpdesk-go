package main

import (
    "context"
    "os"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/redis/go-redis/v9"
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
    "github.com/joho/godotenv"
)

type Config struct {
    DatabaseURL string
    RedisAddr   string
    Env         string
}

func getEnv(key, def string) string { if v:=os.Getenv(key); v!="" { return v }; return def }

func cfg() Config {
    _ = godotenv.Load()
    return Config{
        DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/helpdesk?sslmode=disable"),
        RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
        Env:         getEnv("ENV", "dev"),
    }
}

func main() {
    c := cfg()
    if c.Env == "dev" {
        log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
    }
    ctx := context.Background()
    db, err := pgxpool.New(ctx, c.DatabaseURL)
    if err != nil { log.Fatal().Err(err).Msg("db connect") }
    defer db.Close()

    rdb := redis.NewClient(&redis.Options{Addr: c.RedisAddr})
    if err := rdb.Ping(ctx).Err(); err != nil {
        log.Error().Err(err).Msg("redis ping failed (queue not active yet)")
    }

    log.Info().Msg("worker started (no jobs yet)")
    for {
        // Placeholder loop
        time.Sleep(10 * time.Second)
    }
}
