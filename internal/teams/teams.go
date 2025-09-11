package teams

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func List(ctx context.Context, db DB) ([]Team, error) {
	rows, err := db.Query(ctx, `select id::text, name from teams order by name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Team
	for rows.Next() {
		var t Team
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
