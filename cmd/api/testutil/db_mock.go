package testutil

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// MockDB implements app.DB for testing.
type MockDB struct {
	QueryFunc    func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRowFunc func(ctx context.Context, sql string, args ...interface{}) pgx.Row
	ExecFunc     func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	BeginFunc    func(ctx context.Context) (pgx.Tx, error)
}

func (m *MockDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, sql, args...)
	}
	return &MockRows{}, nil
}

func (m *MockDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(ctx, sql, args...)
	}
	return &MockRow{}
}

func (m *MockDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, sql, args...)
	}
	return pgconn.CommandTag{}, nil
}

func (m *MockDB) Begin(ctx context.Context) (pgx.Tx, error) {
	if m.BeginFunc != nil {
		return m.BeginFunc(ctx)
	}
	return nil, nil
}

// MockRow implements pgx.Row.
type MockRow struct {
	ScanFunc func(dest ...interface{}) error
}

func (r *MockRow) Scan(dest ...interface{}) error {
	if r.ScanFunc != nil {
		return r.ScanFunc(dest...)
	}
	return nil
}

// MockRows implements pgx.Rows.
type MockRows struct {
	NextFunc  func() bool
	ScanFunc  func(dest ...interface{}) error
	CloseFunc func()
	ErrFunc   func() error
}

func (r *MockRows) Close() {
	if r.CloseFunc != nil {
		r.CloseFunc()
	}
}

func (r *MockRows) Err() error {
	if r.ErrFunc != nil {
		return r.ErrFunc()
	}
	return nil
}

func (r *MockRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (r *MockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *MockRows) Next() bool {
	if r.NextFunc != nil {
		return r.NextFunc()
	}
	return false
}

func (r *MockRows) Scan(dest ...interface{}) error {
	if r.ScanFunc != nil {
		return r.ScanFunc(dest...)
	}
	return nil
}

func (r *MockRows) Values() ([]interface{}, error) { return nil, nil }

func (r *MockRows) RawValues() [][]byte { return nil }

func (r *MockRows) Conn() *pgx.Conn { return nil }
