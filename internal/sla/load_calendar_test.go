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

func (r *fakeRows) Close()                           {}
func (r *fakeRows) Err() error                       { return nil }
func (r *fakeRows) CommandTag() pgconn.CommandTag    { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Next() bool                       { return r.i < len(r.data) }
func (r *fakeRows) Values() ([]any, error)           { return nil, nil }
func (r *fakeRows) RawValues() [][]byte              { return nil }
func (r *fakeRows) Conn() *pgx.Conn                  { return nil }
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

type fakeDB struct{
    tz string
    hours [][]any
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

func TestLoadCalendar_LoadsHoursAndHolidays(t *testing.T) {
    loc := "America/New_York"
    // hours: Mon 9-17
    hours := [][]any{{int(time.Monday), 9*3600, 17*3600}}
    // holiday: 2024-07-04
    hday := time.Date(2024, 7, 4, 0, 0, 0, 0, time.UTC)
    holidays := [][]any{{hday}}
    db := fakeDB{tz: loc, hours: hours, holidays: holidays}

    cal, err := LoadCalendar(context.Background(), db, "cal-1")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if cal.Location == nil || cal.Location.String() == "" {
        t.Fatalf("expected location to be set")
    }
    if _, ok := cal.Hours[time.Monday]; !ok {
        t.Fatalf("expected Monday hours to be set")
    }
    // verify holiday normalized to midnight in tz
    day := time.Date(2024, 7, 4, 0, 0, 0, 0, cal.Location)
    if _, ok := cal.Holidays[day]; !ok {
        t.Fatalf("expected holiday to be loaded")
    }
}

