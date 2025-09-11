package kb

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type Article struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	BodyMD string `json:"body_md"`
}

func Search(ctx context.Context, db DB, q string) ([]Article, error) {
	q = strings.TrimSpace(q)
	rows, err := db.Query(ctx, `select id::text, slug, title, body_md from kb_articles where title ilike '%'||$1||'%' or body_md ilike '%'||$1||'%' order by title limit 20`, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Article{}
	for rows.Next() {
		var a Article
		if err := rows.Scan(&a.ID, &a.Slug, &a.Title, &a.BodyMD); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func Get(ctx context.Context, db DB, slug string) (Article, error) {
	var a Article
	err := db.QueryRow(ctx, `select id::text, slug, title, body_md from kb_articles where slug=$1`, slug).Scan(&a.ID, &a.Slug, &a.Title, &a.BodyMD)
	return a, err
}

func Create(ctx context.Context, db DB, a Article) (Article, error) {
	var out Article
	err := db.QueryRow(ctx, `insert into kb_articles (slug, title, body_md) values ($1,$2,$3) returning id::text, slug, title, body_md`, a.Slug, a.Title, a.BodyMD).Scan(&out.ID, &out.Slug, &out.Title, &out.BodyMD)
	return out, err
}

func Update(ctx context.Context, db DB, slug string, a Article) (Article, error) {
	var out Article
	err := db.QueryRow(ctx, `update kb_articles set slug=$1, title=$2, body_md=$3, updated_at=now() where slug=$4 returning id::text, slug, title, body_md`, a.Slug, a.Title, a.BodyMD, slug).Scan(&out.ID, &out.Slug, &out.Title, &out.BodyMD)
	return out, err
}

func Delete(ctx context.Context, db DB, slug string) error {
	_, err := db.Exec(ctx, `delete from kb_articles where slug=$1`, slug)
	return err
}
