package kb

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
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
