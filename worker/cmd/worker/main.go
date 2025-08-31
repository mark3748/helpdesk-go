package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"net/smtp"
	"os"
	"text/template"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	DatabaseURL string
	RedisAddr   string
	Env         string
	SMTPHost    string
	SMTPPort    string
	SMTPUser    string
	SMTPPass    string
	SMTPFrom    string
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func cfg() Config {
	_ = godotenv.Load()
	return Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/helpdesk?sslmode=disable"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		Env:         getEnv("ENV", "dev"),
		SMTPHost:    getEnv("SMTP_HOST", ""),
		SMTPPort:    getEnv("SMTP_PORT", "25"),
		SMTPUser:    getEnv("SMTP_USER", ""),
		SMTPPass:    getEnv("SMTP_PASS", ""),
		SMTPFrom:    getEnv("SMTP_FROM", ""),
	}
}

//go:embed templates/*.tmpl
var templatesFS embed.FS

var mailTemplates = template.Must(template.ParseFS(templatesFS, "templates/*.tmpl"))

type Job struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type EmailJob struct {
	To       string      `json:"to"`
	Template string      `json:"template"`
	Data     interface{} `json:"data"`
}

func sendEmail(c Config, j EmailJob) error {
	var subjBuf, bodyBuf bytes.Buffer
	if err := mailTemplates.ExecuteTemplate(&subjBuf, j.Template+"_subject", j.Data); err != nil {
		return err
	}
	if err := mailTemplates.ExecuteTemplate(&bodyBuf, j.Template+"_body", j.Data); err != nil {
		return err
	}
	msg := bytes.Buffer{}
	msg.WriteString("From: " + c.SMTPFrom + "\r\n")
	msg.WriteString("To: " + j.To + "\r\n")
	msg.WriteString("Subject: " + subjBuf.String() + "\r\n\r\n")
	msg.Write(bodyBuf.Bytes())
	addr := c.SMTPHost + ":" + c.SMTPPort
	var auth smtp.Auth
	if c.SMTPUser != "" {
		auth = smtp.PlainAuth("", c.SMTPUser, c.SMTPPass, c.SMTPHost)
	}
	return smtp.SendMail(addr, auth, c.SMTPFrom, []string{j.To}, msg.Bytes())
}

func main() {
	c := cfg()
	if c.Env == "dev" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, c.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("db connect")
	}
	defer db.Close()

	rdb := redis.NewClient(&redis.Options{Addr: c.RedisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Error().Err(err).Msg("redis ping failed (queue not active yet)")
	}

	log.Info().Msg("worker started")
	for {
		res, err := rdb.BLPop(ctx, 0, "jobs").Result()
		if err != nil {
			log.Error().Err(err).Msg("blpop")
			continue
		}
		if len(res) < 2 {
			continue
		}
		var job Job
		if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
			log.Error().Err(err).Msg("unmarshal job")
			continue
		}
		switch job.Type {
		case "send_email":
			var ej EmailJob
			if err := json.Unmarshal(job.Data, &ej); err != nil {
				log.Error().Err(err).Msg("unmarshal email job")
				continue
			}
			if err := sendEmail(c, ej); err != nil {
				log.Error().Err(err).Msg("send email")
			}
		default:
			log.Warn().Str("type", job.Type).Msg("unknown job type")
		}
	}
}
