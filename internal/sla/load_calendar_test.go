package sla

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type rowFunc func(dest ...any) error

type fakeRow struct{ f rowFunc }

func (r fakeRow) Scan(dest ...any) error { return r.f(dest...) }

type fakeRows struct {
	data [][]any
	i    int
}

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return nil }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Next() bool                                   { return r.i < len(r.data) }
func (r *fakeRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeRows) RawValues() [][]byte                          { return nil }
func (r *fakeRows) Conn() *pgx.Conn                              { return nil }
func (r *fakeRows) Scan(dest ...any) error {
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

type fakeDB struct {
	tz       string
	hours    [][]any
	holidays [][]any
}

func (db fakeDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if len(db.hours) > 0 && sql == "select dow, start_sec, end_sec from business_hours where calendar_id=$1" {
		return &fakeRows{data: db.hours}, nil
	}
	if len(db.holidays) > 0 && sql == "select date from holidays where calendar_id=$1" {
		return &fakeRows{data: db.holidays}, nil
	}
	return &fakeRows{}, nil
}
func (db fakeDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if sql == "select tz from calendars where id=$1" {
		return fakeRow{f: func(dest ...any) error {
			*(dest[0].(*string)) = db.tz
			return nil
		}}
	}
	return fakeRow{f: func(dest ...any) error { return nil }}
}

func TestLoadCalendar(t *testing.T) {
	loc := "America/New_York"
	cases := []struct {
		name     string
		db       fakeDB
		validate func(t *testing.T, cal *Calendar)
	}{
		{
			name: "normalizes holidays",
			db: fakeDB{
				tz:    loc,
				hours: [][]any{{int(time.Monday), 9 * 3600, 17 * 3600}},
				holidays: [][]any{{
					time.Date(2024, 7, 4, 15, 30, 0, 0, time.UTC), // not midnight
				}},
			},
			validate: func(t *testing.T, cal *Calendar) {
				day := time.Date(2024, 7, 4, 0, 0, 0, 0, cal.Location)
				if _, ok := cal.Holidays[day]; !ok {
					t.Fatalf("expected holiday to be normalized")
				}
			},
		},
		{
			name: "loads varying business hours",
			db: fakeDB{
				tz: loc,
				hours: [][]any{
					{int(time.Monday), 8 * 3600, 12 * 3600},
					{int(time.Tuesday), 10 * 3600, 15 * 3600},
				},
			},
			validate: func(t *testing.T, cal *Calendar) {
				m := cal.Hours[time.Monday]
				if m.StartSec != 8*3600 || m.EndSec != 12*3600 {
					t.Fatalf("unexpected Monday hours: %+v", m)
				}
				tu := cal.Hours[time.Tuesday]
				if tu.StartSec != 10*3600 || tu.EndSec != 15*3600 {
					t.Fatalf("unexpected Tuesday hours: %+v", tu)
				}
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cal, err := LoadCalendar(context.Background(), tt.db, "cal-1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cal.Location == nil || cal.Location.String() == "" {
				t.Fatalf("expected location to be set")
			}
			tt.validate(t, cal)
		})
	}
}
