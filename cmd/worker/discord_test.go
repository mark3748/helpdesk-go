package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
)

type discordTestRow struct {
	scan func(dest ...any) error
}

func (r discordTestRow) Scan(dest ...any) error {
	if r.scan == nil {
		return pgx.ErrNoRows
	}
	return r.scan(dest...)
}

type discordTestDB struct {
	lastSQL  string
	lastArgs []any
	execs    []string
	queryRow func(sql string, args ...any) pgx.Row
}

func (db *discordTestDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func (db *discordTestDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	db.lastSQL = sql
	db.lastArgs = args
	if db.queryRow != nil {
		return db.queryRow(sql, args...)
	}
	return discordTestRow{}
}

func (db *discordTestDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	db.execs = append(db.execs, sql)
	db.lastSQL = sql
	db.lastArgs = args
	return pgconn.CommandTag{}, nil
}

func (db *discordTestDB) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, nil
}

func (db *discordTestDB) Ping(ctx context.Context) error {
	return nil
}

func TestBeginDiscordEmailLink_EnqueuesVerificationWithoutMapping(t *testing.T) {
	db := &discordTestDB{
		queryRow: func(sql string, args ...any) pgx.Row {
			return discordTestRow{scan: func(dest ...any) error {
				*(dest[0].(*string)) = "challenge-1"
				return nil
			}}
		},
	}
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	err := beginDiscordEmailLink(context.Background(), "discord123", "John@Example.com", db, rdb)
	if err != nil {
		t.Fatalf("beginDiscordEmailLink failed: %v", err)
	}
	if strings.Contains(db.lastSQL, "discord_user_mappings") {
		t.Fatal("begin flow must not create a Discord user mapping")
	}
	if got := db.lastArgs[1]; got != "john@example.com" {
		t.Fatalf("normalized email = %v, want john@example.com", got)
	}

	items, err := rdb.LRange(context.Background(), "jobs", 0, -1).Result()
	if err != nil {
		t.Fatalf("read queued jobs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("queued jobs = %d, want 1", len(items))
	}
	var job Job
	if err := json.Unmarshal([]byte(items[0]), &job); err != nil {
		t.Fatalf("unmarshal job: %v", err)
	}
	if job.Type != "send_email" {
		t.Fatalf("job type = %q, want send_email", job.Type)
	}
	var emailJob EmailJob
	if err := json.Unmarshal(job.Data, &emailJob); err != nil {
		t.Fatalf("unmarshal email job: %v", err)
	}
	if emailJob.To != "john@example.com" || emailJob.Template != "discord_link_verification" {
		t.Fatalf("unexpected email job: %+v", emailJob)
	}
	data, ok := emailJob.Data.(map[string]any)
	if !ok {
		t.Fatalf("email job data type = %T", emailJob.Data)
	}
	token, _ := data["Token"].(string)
	if token == "" {
		t.Fatal("verification token missing from email")
	}
	storedHash, ok := db.lastArgs[2].([]byte)
	if !ok {
		t.Fatalf("stored token hash type = %T", db.lastArgs[2])
	}
	wantHash := sha256.Sum256([]byte(token))
	if !bytes.Equal(storedHash, wantHash[:]) {
		t.Fatal("stored token hash does not match emailed token")
	}
	if bytes.Equal(storedHash, []byte(token)) {
		t.Fatal("plaintext token was stored")
	}
}

func TestBeginDiscordEmailLink_RateLimited(t *testing.T) {
	db := &discordTestDB{}
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	err := beginDiscordEmailLink(context.Background(), "discord123", "john@example.com", db, rdb)
	if !errors.Is(err, errDiscordLinkRateLimited) {
		t.Fatalf("error = %v, want errDiscordLinkRateLimited", err)
	}
	if got := rdb.LLen(context.Background(), "jobs").Val(); got != 0 {
		t.Fatal("rate-limited request queued an email")
	}
}

func TestBeginDiscordEmailLink_InvalidEmail(t *testing.T) {
	db := &discordTestDB{}
	err := beginDiscordEmailLink(context.Background(), "discord123", "", db, nil)
	if err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestCompleteDiscordEmailLink_ConsumesChallengeAndMapsRequester(t *testing.T) {
	db := &discordTestDB{
		queryRow: func(sql string, args ...any) pgx.Row {
			return discordTestRow{scan: func(dest ...any) error {
				*(dest[0].(*string)) = "requester-1"
				return nil
			}}
		},
	}

	err := completeDiscordEmailLink(context.Background(), "discord123", "johndoe", "secret-token", db)
	if err != nil {
		t.Fatalf("completeDiscordEmailLink failed: %v", err)
	}
	if !strings.Contains(db.lastSQL, "set consumed_at = now()") {
		t.Fatal("verification query does not consume the challenge")
	}
	if !strings.Contains(db.lastSQL, "insert into discord_user_mappings") {
		t.Fatal("verification query does not create the mapping")
	}
	storedHash := db.lastArgs[1].([]byte)
	wantHash := sha256.Sum256([]byte("secret-token"))
	if !bytes.Equal(storedHash, wantHash[:]) {
		t.Fatal("verification query did not use the token hash")
	}
}

func TestCompleteDiscordEmailLink_RejectsInvalidOrReusedToken(t *testing.T) {
	db := &discordTestDB{}
	err := completeDiscordEmailLink(context.Background(), "discord123", "johndoe", "bad-token", db)
	if !errors.Is(err, errDiscordLinkInvalid) {
		t.Fatalf("error = %v, want errDiscordLinkInvalid", err)
	}
}

func TestDiscordEmailLinkEnabled(t *testing.T) {
	if discordEmailLinkEnabled(Config{SMTPHost: "smtp.example.com"}) {
		t.Fatal("email linking enabled without SMTP_FROM")
	}
	if !discordEmailLinkEnabled(Config{SMTPHost: "smtp.example.com", SMTPFrom: "helpdesk@example.com"}) {
		t.Fatal("email linking disabled with complete SMTP configuration")
	}
}

func TestDiscordModalTextInputValues_ExtractsDiscordGoPointerComponents(t *testing.T) {
	var data discordgo.ModalSubmitInteractionData
	err := json.Unmarshal([]byte(`{
		"custom_id": "create_ticket_modal",
		"components": [
			{"type": 1, "components": [{"type": 4, "custom_id": "ticket_title", "value": "Printer is jammed"}]},
			{"type": 1, "components": [{"type": 4, "custom_id": "ticket_desc", "value": "Third floor printer"}]},
			{"type": 1, "components": [{"type": 4, "custom_id": "ticket_priority", "value": "3"}]}
		]
	}`), &data)
	if err != nil {
		t.Fatalf("unmarshal Discord modal: %v", err)
	}

	values := discordModalTextInputValues(data)
	if got := values[discordTicketTitleInputID]; got != "Printer is jammed" {
		t.Fatalf("title = %q, want Printer is jammed", got)
	}
	if got := values[discordTicketDescInputID]; got != "Third floor printer" {
		t.Fatalf("description = %q, want Third floor printer", got)
	}
	if got := values[discordTicketPriorityInputID]; got != "3" {
		t.Fatalf("priority = %q, want 3", got)
	}
}

func TestSendCommentToDiscord_UnmappedTicketIgnoresUnavailableSession(t *testing.T) {
	dgSession.Store(nil)

	err := sendCommentToDiscord(context.Background(), &discordTestDB{}, "ticket-1", "hello")
	if err != nil {
		t.Fatalf("unmapped ticket returned error with unavailable Discord session: %v", err)
	}
}

func TestSendCommentToDiscord_MappedTicketRequiresSession(t *testing.T) {
	dgSession.Store(nil)
	db := &discordTestDB{
		queryRow: func(sql string, args ...any) pgx.Row {
			return discordTestRow{scan: func(dest ...any) error {
				*(dest[0].(*string)) = "thread-1"
				return nil
			}}
		},
	}

	err := sendCommentToDiscord(context.Background(), db, "ticket-1", "hello")
	if err == nil || !strings.Contains(err.Error(), "session is not available") {
		t.Fatalf("mapped ticket error = %v, want unavailable session error", err)
	}
}

func TestEffectiveMailConfigDatabaseOverridesEnvironment(t *testing.T) {
	db := &discordTestDB{
		queryRow: func(sql string, args ...any) pgx.Row {
			return discordTestRow{scan: func(dest ...any) error {
				*(dest[0].(*[]byte)) = []byte(`{
					"smtp_host":"db-smtp.example.com",
					"smtp_pass":"db-secret",
					"imap_host":"db-imap.example.com",
					"imap_port":"1993"
				}`)
				return nil
			}}
		},
	}
	base := Config{
		SMTPHost: "env-smtp.example.com",
		SMTPPort: "25",
		SMTPPass: "env-secret",
		IMAPPort: "993",
	}

	got := effectiveMailConfig(context.Background(), db, base)
	if got.SMTPHost != "db-smtp.example.com" || got.SMTPPass != "db-secret" {
		t.Fatalf("SMTP database override not applied: %+v", got)
	}
	if got.SMTPPort != "25" {
		t.Fatalf("empty database SMTP port should preserve environment fallback, got %q", got.SMTPPort)
	}
	if got.IMAPHost != "db-imap.example.com" || got.IMAPPort != "1993" {
		t.Fatalf("IMAP database override not applied: %+v", got)
	}
}
