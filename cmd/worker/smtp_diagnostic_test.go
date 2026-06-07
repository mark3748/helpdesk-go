package main

import (
	"crypto/tls"
	"net"
	"net/smtp"
	"os"
	"strings"
	"testing"
	"time"
)

// TestSMTPDiagnosticHandshake bypasses Redis and the worker dispatch loop to
// expose SMTP connection, STARTTLS, and authentication errors directly.
//
// Run with:
// SMTP_DIAGNOSTIC=1 go test ./cmd/worker -run TestSMTPDiagnosticHandshake -v
func TestSMTPDiagnosticHandshake(t *testing.T) {
	if os.Getenv("SMTP_DIAGNOSTIC") != "1" {
		t.Skip("set SMTP_DIAGNOSTIC=1 to run the direct SMTP handshake")
	}

	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	user := strings.TrimSpace(os.Getenv("SMTP_USER"))
	pass := os.Getenv("SMTP_PASS")
	if host == "" {
		t.Fatal("SMTP_HOST is required")
	}
	if port == "" {
		port = "587"
	}

	addr := net.JoinHostPort(host, port)
	t.Logf("connecting to SMTP relay %s", addr)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		t.Fatalf("SMTP TCP connect failed: %v", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		t.Fatalf("SMTP greeting failed: %v", err)
	}
	defer client.Close()

	if err := client.Hello("localhost"); err != nil {
		t.Fatalf("SMTP EHLO/HELO failed: %v", err)
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		t.Log("SMTP server advertised STARTTLS; negotiating TLS")
		if err := client.StartTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: host,
		}); err != nil {
			t.Fatalf("SMTP STARTTLS failed: %v", err)
		}
	} else {
		t.Log("SMTP server did not advertise STARTTLS")
	}

	if user != "" {
		if ok, mechanisms := client.Extension("AUTH"); ok {
			t.Logf("SMTP server advertised AUTH mechanisms: %s", mechanisms)
		} else {
			t.Fatal("SMTP credentials are configured, but the server did not advertise AUTH")
		}
		if err := client.Auth(smtp.PlainAuth("", user, pass, host)); err != nil {
			t.Fatalf("SMTP AUTH failed: %v", err)
		}
		t.Log("SMTP authentication succeeded")
	} else {
		t.Log("SMTP_USER is empty; skipping authentication")
	}

	if err := client.Quit(); err != nil {
		t.Fatalf("SMTP QUIT failed: %v", err)
	}
	t.Log("SMTP handshake completed successfully; no message was sent")
}
