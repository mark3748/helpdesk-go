package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/smtp"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/microcosm-cc/bluemonday"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	handlers "github.com/mark3748/helpdesk-go/cmd/api/handlers"
	"github.com/mark3748/helpdesk-go/internal/sla"
)

type Config struct {
	DatabaseURL   string
	RedisAddr     string
	Env           string
	SMTPHost      string
	SMTPPort      string
	SMTPUser      string
	SMTPPass      string
	SMTPFrom      string
	IMAPHost      string
	IMAPUser      string
	IMAPPass      string
	IMAPFolder    string
	MinIOEndpoint string
	MinIOAccess   string
	MinIOSecret   string
	MinIOBucket   string
	MinIOUseSSL   bool
	LogPath       string
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
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/helpdesk?sslmode=disable"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Env:           getEnv("ENV", "dev"),
		SMTPHost:      getEnv("SMTP_HOST", ""),
		SMTPPort:      getEnv("SMTP_PORT", "25"),
		SMTPUser:      getEnv("SMTP_USER", ""),
		SMTPPass:      getEnv("SMTP_PASS", ""),
		SMTPFrom:      getEnv("SMTP_FROM", ""),
		IMAPHost:      getEnv("IMAP_HOST", ""),
		IMAPUser:      getEnv("IMAP_USER", ""),
		IMAPPass:      getEnv("IMAP_PASS", ""),
		IMAPFolder:    getEnv("IMAP_FOLDER", "INBOX"),
		MinIOEndpoint: getEnv("MINIO_ENDPOINT", ""),
		MinIOAccess:   getEnv("MINIO_ACCESS_KEY", ""),
		MinIOSecret:   getEnv("MINIO_SECRET_KEY", ""),
		MinIOBucket:   getEnv("MINIO_BUCKET", ""),
		MinIOUseSSL:   getEnv("MINIO_USE_SSL", "false") == "true",
		LogPath:       getEnv("LOG_PATH", os.TempDir()),
	}
}

//go:embed templates/*.tmpl
var templatesFS embed.FS

var mailTemplates = template.Must(template.ParseFS(templatesFS, "templates/*.tmpl"))

type Job struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type EmailJob struct {
	To       string      `json:"to"`
	Template string      `json:"template"`
	Data     interface{} `json:"data"`
}

type ExportTicketsJob struct {
	IDs       []string `json:"ids"`
	Requester string   `json:"requester"`
}

type ExportStatus struct {
    Requester string `json:"requester"`
    Status    string `json:"status"`
    URL       string `json:"url,omitempty"`
    ObjectKey string `json:"object_key,omitempty"`
    Error     string `json:"error,omitempty"`
}

type DB interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
}

type ObjectStore interface {
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
}

// Email address validation regex based on RFC 5322 simplified pattern
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// HTML sanitization policy for email bodies
var htmlPolicy = bluemonday.UGCPolicy()

// sanitizeEmailHeader removes CRLF characters that could be used for header injection
func sanitizeEmailHeader(input string) string {
	// Remove carriage return and line feed characters
	sanitized := strings.ReplaceAll(input, "\r", "")
	sanitized = strings.ReplaceAll(sanitized, "\n", "")
	return strings.TrimSpace(sanitized)
}

// validateEmailAddress checks if an email address is valid
func validateEmailAddress(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email address cannot be empty")
	}
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email address format: %s", email)
	}
	return nil
}

// sanitizeAndValidateEmail sanitizes and validates an email address
func sanitizeAndValidateEmail(email string) (string, error) {
	sanitized := sanitizeEmailHeader(email)
	if err := validateEmailAddress(sanitized); err != nil {
		return "", err
	}
	return sanitized, nil
}

// sanitizeEmailBody removes potentially harmful HTML or scripts from an email body
func sanitizeEmailBody(body []byte) string {
	return string(htmlPolicy.SanitizeBytes(body))
}

func sendEmail(c Config, j EmailJob) error {
	// Sanitize and validate email addresses
	sanitizedTo, err := sanitizeAndValidateEmail(j.To)
	if err != nil {
		return fmt.Errorf("invalid To address: %w", err)
	}

	sanitizedFrom, err := sanitizeAndValidateEmail(c.SMTPFrom)
	if err != nil {
		return fmt.Errorf("invalid From address: %w", err)
	}

	var subjBuf, bodyBuf bytes.Buffer
	if err := mailTemplates.ExecuteTemplate(&subjBuf, j.Template+"_subject", j.Data); err != nil {
		return err
	}
	if err := mailTemplates.ExecuteTemplate(&bodyBuf, j.Template+"_body", j.Data); err != nil {
		return err
	}

	// Sanitize the subject to prevent header injection
	sanitizedSubject := sanitizeEmailHeader(subjBuf.String())

	msg := bytes.Buffer{}
	msg.WriteString("From: " + sanitizedFrom + "\r\n")
	msg.WriteString("To: " + sanitizedTo + "\r\n")
	msg.WriteString("Subject: " + sanitizedSubject + "\r\n\r\n")
	msg.Write(bodyBuf.Bytes())
	addr := c.SMTPHost + ":" + c.SMTPPort
	var auth smtp.Auth
	if c.SMTPUser != "" {
		auth = smtp.PlainAuth("", c.SMTPUser, c.SMTPPass, c.SMTPHost)
	}
	return smtp.SendMail(addr, auth, sanitizedFrom, []string{sanitizedTo}, msg.Bytes())
}

// exportTickets generates the CSV and uploads it to the object store, returning the object key.
func exportTickets(ctx context.Context, c Config, db DB, store ObjectStore, ids []string) (string, error) {
	if store == nil {
		return "", fmt.Errorf("object store not configured")
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	q := fmt.Sprintf("select id, number, title, status, priority from tickets where id in (%s)", strings.Join(placeholders, ","))
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	buf := &bytes.Buffer{}
	w := csv.NewWriter(buf)
	_ = w.Write([]string{"id", "number", "title", "status", "priority"})
	for rows.Next() {
		var id, number, title, status string
		var priority int16
		if err := rows.Scan(&id, &number, &title, &status, &priority); err != nil {
			return "", err
		}
		_ = w.Write([]string{id, number, title, status, strconv.Itoa(int(priority))})
	}
	w.Flush()
    objectKey := uuid.New().String() + ".csv"
    _, err = store.PutObject(ctx, c.MinIOBucket, objectKey, bytes.NewReader(buf.Bytes()), int64(buf.Len()), minio.PutObjectOptions{ContentType: "text/csv"})
    if err != nil {
        return "", err
    }
    // Return the object key; URL will be generated by API when client polls status.
    return objectKey, nil
}

func handleExportTicketsJob(ctx context.Context, c Config, db DB, store ObjectStore, rdb *redis.Client, jobID string, ej ExportTicketsJob) {
    objectKey, err := exportTickets(ctx, c, db, store, ej.IDs)
    st := ExportStatus{Requester: ej.Requester}
    if err != nil {
        st.Status = "error"
        st.Error = err.Error()
    } else {
        st.Status = "done"
        st.ObjectKey = objectKey
    }
    b, _ := json.Marshal(st)
    if err := rdb.Set(ctx, "export_tickets:"+jobID, b, 0).Err(); err != nil {
        log.Error().Err(err).Msg("store export result")
    }
}

func main() {
	c := cfg()
	writer := io.Writer(os.Stdout)
	if c.Env == "dev" {
		writer = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}
	if err := os.MkdirAll(c.LogPath, 0o755); err != nil {
		log.Warn().Err(err).Str("dir", c.LogPath).Msg("using stdout for logs")
	} else {
		logFile := filepath.Join(c.LogPath, "worker.log")
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			log.Warn().Err(err).Str("path", logFile).Msg("using stdout for logs")
		} else {
			if c.Env == "dev" {
				writer = zerolog.MultiLevelWriter(f, writer)
			} else {
				writer = f
			}
			defer f.Close()
		}
	}
	log.Logger = zerolog.New(writer).With().Timestamp().Logger()
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
	defer rdb.Close()

	var mc *minio.Client
	var store ObjectStore
	if c.MinIOEndpoint != "" {
		mc, err = minio.New(c.MinIOEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(c.MinIOAccess, c.MinIOSecret, ""),
			Secure: c.MinIOUseSSL,
		})
		if err != nil {
			log.Error().Err(err).Msg("minio init")
		} else {
			store = mc
		}
	}

	if c.IMAPHost != "" {
		go func() {
			for {
				if err := pollIMAP(ctx, c, db, mc, rdb); err != nil {
					log.Error().Err(err).Msg("poll imap")
				}
				time.Sleep(time.Minute)
			}
		}()
	}

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := updateSLAClocks(ctx, db); err != nil {
				log.Error().Err(err).Msg("sla update")
			}
		}
	}()

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
		size, _ := rdb.LLen(ctx, "jobs").Result()
		handlers.PublishEvent(ctx, rdb, handlers.Event{Type: "queue_changed", Data: map[string]interface{}{"size": size}})
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
		case "export_tickets":
			var ej ExportTicketsJob
			if err := json.Unmarshal(job.Data, &ej); err != nil {
				log.Error().Err(err).Msg("unmarshal export job")
				continue
			}
			handleExportTicketsJob(ctx, c, db, store, rdb, job.ID, ej)
		default:
			log.Warn().Str("type", job.Type).Msg("unknown job type")
		}
	}
}

func updateSLAClocks(ctx context.Context, db *pgxpool.Pool) error {
	rows, err := db.Query(ctx, `
       select t.id, coalesce(tm.calendar_id, r.calendar_id), sc.response_elapsed_ms,
              sc.resolution_elapsed_ms, sc.last_started_at,
              sp.response_target_mins, sp.resolution_target_mins
       from ticket_sla_clocks sc
       join tickets t on t.id = sc.ticket_id
       left join teams tm on t.team_id = tm.id
       left join regions r on tm.region_id = r.id
       join sla_policies sp on sp.id = sc.policy_id
       where not sc.paused and sc.last_started_at is not null`)
	if err != nil {
		return err
	}
	defer rows.Close()
	now := time.Now()
	calendars := map[string]*sla.Calendar{}
	for rows.Next() {
		var ticketID, calID string
		var respMS, resMS int64
		var lastStarted time.Time
		var respTarget, resTarget int
		if err := rows.Scan(&ticketID, &calID, &respMS, &resMS, &lastStarted, &respTarget, &resTarget); err != nil {
			log.Error().Err(err).Msg("failed to scan row in updateSLAClocks")
			continue
		}
		if calID == "" {
			continue
		}
		cal, ok := calendars[calID]
		if !ok {
			cal, err = sla.LoadCalendar(ctx, db, calID)
			if err != nil {
				log.Error().Err(err).Str("calendar", calID).Msg("load calendar")
				continue
			}
			calendars[calID] = cal
		}
		dur := cal.BusinessDuration(lastStarted, now)
		respMS += int64(dur / time.Millisecond)
		resMS += int64(dur / time.Millisecond)
		_, err = db.Exec(ctx, `update ticket_sla_clocks set response_elapsed_ms=$1, resolution_elapsed_ms=$2, last_started_at=$3 where ticket_id=$4`, respMS, resMS, now, ticketID)
		if err != nil {
			log.Error().Err(err).Str("ticket", ticketID).Msg("update sla")
		}
		if respTarget > 0 && respMS > int64(respTarget)*60*1000 {
			log.Warn().Str("ticket", ticketID).Msg("response SLA breached")
		}
		if resTarget > 0 && resMS > int64(resTarget)*60*1000 {
			log.Warn().Str("ticket", ticketID).Msg("resolution SLA breached")
		}
	}
	return rows.Err()
}
