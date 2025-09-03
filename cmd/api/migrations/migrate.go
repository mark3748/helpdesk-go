package migrations

import (
    "context"
    "embed"
    "fmt"
    "sort"
    "strings"

    "github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.sql
var fs embed.FS

// Apply runs embedded SQL migrations marked with `-- +goose Up` in filename order.
func Apply(ctx context.Context, db *pgxpool.Pool) error {
    entries, err := fs.ReadDir(".")
    if err != nil {
        return err
    }
    names := make([]string, 0, len(entries))
    for _, e := range entries {
        if e.IsDir() { continue }
        if strings.HasSuffix(e.Name(), ".sql") {
            names = append(names, e.Name())
        }
    }
    sort.Strings(names)
    for _, name := range names {
        b, err := fs.ReadFile(name)
        if err != nil {
            return fmt.Errorf("read %s: %w", name, err)
        }
        up := extractUp(string(b))
        if strings.TrimSpace(up) == "" {
            continue
        }
        if _, err := db.Exec(ctx, up); err != nil {
            return fmt.Errorf("apply %s: %w", name, err)
        }
    }
    return nil
}

func extractUp(s string) string {
    // Find the section after `-- +goose Up` and before next `-- +goose Down`
    i := strings.Index(s, "-- +goose Up")
    if i < 0 {
        return s
    }
    s = s[i+len("-- +goose Up"):]
    j := strings.Index(s, "-- +goose Down")
    if j >= 0 {
        s = s[:j]
    }
    return s
}

