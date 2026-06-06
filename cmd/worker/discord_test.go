package main

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func TestHandleLinkEmail_NewUser(t *testing.T) {
	db := &discordTestDB{
		queryRow: func(sql string, args ...any) pgx.Row {
			// Simulate no existing requester and no existing mapping
			return discordTestRow{scan: func(dest ...any) error {
				return pgx.ErrNoRows
			}}
		},
	}

	err := handleLinkEmail(context.Background(), "discord123", "johndoe", "john@example.com", db)
	if err != nil {
		t.Fatalf("handleLinkEmail failed: %v", err)
	}

	// Verify new requester was inserted
	foundRequesterInsert := false
	foundMappingInsert := false
	for _, sql := range db.execs {
		if strings.Contains(sql, "insert into requesters") {
			foundRequesterInsert = true
		}
		if strings.Contains(sql, "insert into discord_user_mappings") {
			foundMappingInsert = true
		}
	}

	if !foundRequesterInsert {
		t.Error("expected insert into requesters")
	}
	if !foundMappingInsert {
		t.Error("expected insert into discord_user_mappings")
	}
}

func TestHandleLinkEmail_ExistingRequester(t *testing.T) {
	db := &discordTestDB{
		queryRow: func(sql string, args ...any) pgx.Row {
			// Simulate existing requester with email
			if strings.Contains(sql, "requesters") {
				return discordTestRow{scan: func(dest ...any) error {
					*(dest[0].(*string)) = "req-uuid-123"
					return nil
				}}
			}
			return discordTestRow{scan: func(dest ...any) error {
				return pgx.ErrNoRows
			}}
		},
	}

	err := handleLinkEmail(context.Background(), "discord123", "johndoe", "john@example.com", db)
	if err != nil {
		t.Fatalf("handleLinkEmail failed: %v", err)
	}

	// Verify we only upsert mapping to the existing requester ID, no requester insert
	foundRequesterInsert := false
	foundMappingInsert := false
	for _, sql := range db.execs {
		if strings.Contains(sql, "insert into requesters") {
			foundRequesterInsert = true
		}
		if strings.Contains(sql, "insert into discord_user_mappings") {
			foundMappingInsert = true
		}
	}

	if foundRequesterInsert {
		t.Error("should not insert new requester if email already exists")
	}
	if !foundMappingInsert {
		t.Error("expected upsert into discord_user_mappings")
	}
}

func TestHandleLinkEmail_ExistingMappingUpdateEmail(t *testing.T) {
	db := &discordTestDB{
		queryRow: func(sql string, args ...any) pgx.Row {
			// Simulate no requester with email, but existing mapping
			if strings.Contains(sql, "select id::text") {
				return discordTestRow{scan: func(dest ...any) error {
					return pgx.ErrNoRows
				}}
			}
			if strings.Contains(sql, "discord_user_mappings") {
				return discordTestRow{scan: func(dest ...any) error {
					*(dest[0].(*string)) = "req-uuid-123"
					return nil
				}}
			}
			return discordTestRow{scan: func(dest ...any) error {
				return pgx.ErrNoRows
			}}
		},
	}

	err := handleLinkEmail(context.Background(), "discord123", "johndoe", "john@example.com", db)
	if err != nil {
		t.Fatalf("handleLinkEmail failed: %v", err)
	}

	// Verify we updated the requester's email directly
	foundUpdateEmail := false
	for _, sql := range db.execs {
		if strings.Contains(sql, "update requesters set email") {
			foundUpdateEmail = true
		}
	}

	if !foundUpdateEmail {
		t.Error("expected update requesters set email")
	}
}

func TestHandleLinkEmail_InvalidEmail(t *testing.T) {
	db := &discordTestDB{}
	err := handleLinkEmail(context.Background(), "discord123", "johndoe", "", db)
	if err == nil {
		t.Fatal("expected error for empty email")
	}
}
