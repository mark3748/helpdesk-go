package main

import (
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
)

// fakeRow implements pgx.Row (for tests in this package)
type fakeRow struct {
    err  error
    scan func(dest ...any) error
}

func (r *fakeRow) Scan(dest ...any) error {
    if r.err != nil {
        return r.err
    }
    if r.scan != nil {
        return r.scan(dest...)
    }
    return nil
}

// fakeRows implements pgx.Rows with minimal behavior used in tests
type fakeRows struct{}

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return nil }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Next() bool                                   { return false }
func (r *fakeRows) Scan(dest ...any) error                       { return nil }
func (r *fakeRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeRows) RawValues() [][]byte                          { return nil }
func (r *fakeRows) Conn() *pgx.Conn                              { return nil }
