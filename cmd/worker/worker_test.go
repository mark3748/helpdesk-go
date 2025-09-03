package main

import (
	"context"
	"encoding/json"
	"net/smtp"
	"strings"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
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
	if err := sendEmail(c, j); err != nil {
		t.Fatalf("sendEmail: %v", err)
	}
	if captured.addr != "smtp:25" || captured.from != "from@example.com" || captured.to[0] != "to@example.com" {
		t.Fatalf("unexpected send params: %+v", captured)
	}
	if !strings.Contains(captured.msg, "Ticket created") {
		t.Fatalf("unexpected message: %s", captured.msg)
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
	send := func(c Config, j EmailJob) error {
		called = true
		return nil
	}
	if err := processQueueJob(context.Background(), c, rdb, send); err != nil {
		t.Fatalf("processQueueJob: %v", err)
	}
	if !called {
		t.Fatalf("sendEmail not called")
	}
}
