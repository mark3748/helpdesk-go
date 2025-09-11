package main

import (
	"context"
	"encoding/json"
	"net/smtp"
	"strings"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"

	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
)

func TestSendEmail(t *testing.T) {
	var captured struct {
		addr string
		from string
		to   []string
		msg  string
	}
	smtpSendMail = func(addr string, _ smtp.Auth, from string, to []string, msg []byte) error {
		captured = struct {
			addr string
			from string
			to   []string
			msg  string
		}{addr, from, to, string(msg)}
		return nil
	}
	defer func() { smtpSendMail = smtp.SendMail }()

	c := Config{SMTPHost: "smtp", SMTPPort: "25", SMTPFrom: "from@example.com"}
	j := EmailJob{To: "to@example.com", Template: "ticket_created", Data: struct{ Number int }{1}}
	db := &execDB{}
	if err := sendEmail(context.Background(), db, c, j); err != nil {
		t.Fatalf("sendEmail: %v", err)
	}
	if captured.addr != "smtp:25" || captured.from != "from@example.com" || captured.to[0] != "to@example.com" {
		t.Fatalf("unexpected send params: %+v", captured)
	}
	if !strings.Contains(captured.msg, "Ticket created") {
		t.Fatalf("unexpected message: %s", captured.msg)
	}
	if db.lastSQL == "" || !strings.Contains(strings.ToLower(db.lastSQL), "email_outbound") {
		t.Fatalf("expected insert into email_outbound, got %q", db.lastSQL)
	}
	if len(db.lastArgs) != 6 || db.lastArgs[4].(int) != 0 {
		t.Fatalf("expected retries recorded, got %v", db.lastArgs)
	}
}

func TestProcessQueueJob(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	c := Config{SMTPFrom: "from@example.com"}
	job := Job{Type: "send_email", Data: json.RawMessage(`{"to":"t@example.com","template":"ticket_created","data":{"Number":1}}`)}
	payload, _ := json.Marshal(job)
	if err := rdb.LPush(context.Background(), "jobs", payload).Err(); err != nil {
		t.Fatalf("lpush: %v", err)
	}
	called := false
	send := func(ctx context.Context, db apppkg.DB, c Config, j EmailJob) error {
		called = true
		return nil
	}
	if err := processQueueJob(context.Background(), &execDB{}, c, rdb, send); err != nil {
		t.Fatalf("processQueueJob: %v", err)
	}
	if !called {
		t.Fatalf("sendEmail not called")
	}
}

type execDB struct {
	lastSQL  string
	lastArgs []any
}

func (f *execDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (f *execDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row { return execRow{} }
func (f *execDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	f.lastSQL = sql
	f.lastArgs = args
	return pgconn.CommandTag{}, nil
}
func (f *execDB) Begin(ctx context.Context) (pgx.Tx, error) { return nil, nil }

type execRow struct{}

func (execRow) Scan(dest ...any) error { return pgx.ErrNoRows }

// ==== SLA clock tests ====

type slaRow struct {
	ticketID   string
	calID      string
	respMS     int64
	resMS      int64
	lastStart  time.Time
	paused     bool
	respTarget int
	resTarget  int
}

type slaRows struct {
	data []slaRow
	i    int
}

func (r *slaRows) Close()                                       {}
func (r *slaRows) Err() error                                   { return nil }
func (r *slaRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *slaRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *slaRows) Next() bool                                   { return r.i < len(r.data) }
func (r *slaRows) Values() ([]any, error)                       { return nil, nil }
func (r *slaRows) RawValues() [][]byte                          { return nil }
func (r *slaRows) Conn() *pgx.Conn                              { return nil }
func (r *slaRows) Scan(dest ...any) error {
	row := r.data[r.i]
	r.i++
	*(dest[0].(*string)) = row.ticketID
	*(dest[1].(*string)) = row.calID
	*(dest[2].(*int64)) = row.respMS
	*(dest[3].(*int64)) = row.resMS
	*(dest[4].(*time.Time)) = row.lastStart
	*(dest[5].(*bool)) = row.paused
	*(dest[6].(*int)) = row.respTarget
	*(dest[7].(*int)) = row.resTarget
	return nil
}

type rowFunc func(dest ...any) error

type slaFakeRow struct{ f rowFunc }

func (r slaFakeRow) Scan(dest ...any) error { return r.f(dest...) }

type slaFakeRows struct {
	data [][]any
	i    int
}

func (r *slaFakeRows) Close()                                       {}
func (r *slaFakeRows) Err() error                                   { return nil }
func (r *slaFakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *slaFakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *slaFakeRows) Next() bool                                   { return r.i < len(r.data) }
func (r *slaFakeRows) Values() ([]any, error)                       { return nil, nil }
func (r *slaFakeRows) RawValues() [][]byte                          { return nil }
func (r *slaFakeRows) Conn() *pgx.Conn                              { return nil }
func (r *slaFakeRows) Scan(dest ...any) error {
	row := r.data[r.i]
	r.i++
	for i := range dest {
		switch d := dest[i].(type) {
		case *int:
			*d = row[i].(int)
		case *time.Time:
			*d = row[i].(time.Time)
		}
	}
	return nil
}

type slaDB struct {
	rows      []slaRow
	execCount int
}

func (db *slaDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	switch {
	case strings.Contains(sql, "ticket_sla_clocks"):
		return &slaRows{data: db.rows}, nil
	case strings.Contains(sql, "business_hours"):
		bh := [][]any{}
		for d := 0; d < 7; d++ {
			bh = append(bh, []any{d, 0, 86400})
		}
		return &slaFakeRows{data: bh}, nil
	case strings.Contains(sql, "holidays"):
		return &slaFakeRows{}, nil
	default:
		return &slaFakeRows{}, nil
	}
}

func (db *slaDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if strings.Contains(sql, "select tz from calendars") {
		return slaFakeRow{f: func(dest ...any) error {
			*(dest[0].(*string)) = "UTC"
			return nil
		}}
	}
	return slaFakeRow{f: func(dest ...any) error { return nil }}
}

func (db *slaDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	db.execCount++
	return pgconn.CommandTag{}, nil
}

func (db *slaDB) Begin(ctx context.Context) (pgx.Tx, error) { return nil, nil }

func TestUpdateSLAClocksPaused(t *testing.T) {
	now := time.Now().Add(-time.Minute)
	db := &slaDB{rows: []slaRow{
		{ticketID: "t1", calID: "cal1", lastStart: now, paused: true},
		{ticketID: "t2", calID: "cal1", lastStart: now, paused: false},
	}}
	if err := updateSLAClocks(context.Background(), db); err != nil {
		t.Fatalf("updateSLAClocks: %v", err)
	}
	if db.execCount != 1 {
		t.Fatalf("expected 1 exec, got %d", db.execCount)
	}
}
