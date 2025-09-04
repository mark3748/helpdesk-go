package events

import (
    "context"
    "net/http"
    "net/http/httptest"
    "sort"
    "strings"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"

    apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
    authpkg "github.com/mark3748/helpdesk-go/cmd/api/auth"
)

// fakeRow and fakeRows provide minimal pgx interfaces for the event store.
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

type event struct {
	id        string
	typ       string
	payload   []byte
	createdAt time.Time
}

type eventRows struct {
	idx int
	evs []event
}

func (r *eventRows) Close()                                       {}
func (r *eventRows) Err() error                                   { return nil }
func (r *eventRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *eventRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *eventRows) Next() bool                                   { return r.idx < len(r.evs) }
func (r *eventRows) Scan(dest ...any) error {
	ev := r.evs[r.idx]
	r.idx++
	if len(dest) >= 4 {
		if s, ok := dest[0].(*string); ok {
			*s = ev.id
		}
		if s, ok := dest[1].(*string); ok {
			*s = ev.typ
		}
		if b, ok := dest[2].(*[]byte); ok {
			*b = ev.payload
		}
		if t, ok := dest[3].(*time.Time); ok {
			*t = ev.createdAt
		}
	}
	return nil
}
func (r *eventRows) Values() ([]any, error) { return nil, nil }
func (r *eventRows) RawValues() [][]byte    { return nil }
func (r *eventRows) Conn() *pgx.Conn        { return nil }

type fakeEventDB struct {
	events []event
}

func (db *fakeEventDB) add(typ, payload string) string {
	id := uuid.New().String()
	db.events = append(db.events, event{id: id, typ: typ, payload: []byte(payload), createdAt: time.Now()})
	return id
}

func (db *fakeEventDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
    since, _ := args[0].(time.Time)
    sinceID, _ := args[1].(string)
    out := []event{}
    for _, e := range db.events {
        if e.createdAt.After(since) || (e.createdAt.Equal(since) && e.id != sinceID) {
            out = append(out, e)
        }
    }
    // Ensure stable ordering to match ORDER BY created_at ASC, id ASC
    sort.Slice(out, func(i, j int) bool {
        if out[i].createdAt.Equal(out[j].createdAt) {
            return out[i].id < out[j].id
        }
        return out[i].createdAt.Before(out[j].createdAt)
    })
    return &eventRows{evs: out}, nil
}

func (db *fakeEventDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	id, _ := args[0].(string)
	for _, e := range db.events {
		if e.id == id {
			return &fakeRow{scan: func(dest ...any) error {
				if len(dest) > 0 {
					if t, ok := dest[0].(*time.Time); ok {
						*t = e.createdAt
					}
				}
				return nil
			}}
		}
	}
	return &fakeRow{err: pgx.ErrNoRows}
}

func (db *fakeEventDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func TestStreamResume(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &fakeEventDB{}
	first := db.add("ticket_created", `{"id":"1"}`)
	time.Sleep(time.Millisecond)
	second := db.add("ticket_updated", `{"id":"1"}`)

	a := apppkg.NewApp(apppkg.Config{Env: "test", TestBypassAuth: true}, db, nil, nil, nil)
	a.R.GET("/events", authpkg.Middleware(a), Stream(a))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	req.Header.Set("Last-Event-ID", first)
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)

	done := make(chan struct{})
	go func() {
		a.R.ServeHTTP(rr, req)
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done

	body := rr.Body.String()
	if strings.Contains(body, first) {
		t.Fatalf("stream included old event: %s", body)
	}
	if !strings.Contains(body, second) {
		t.Fatalf("stream missing new event: %s", body)
	}
}

func TestStreamResume_SameTimestamp(t *testing.T) {
    gin.SetMode(gin.TestMode)
    db := &fakeEventDB{}
    ts := time.Now()
    // Manually craft two events with identical timestamps
    firstID := uuid.New().String()
    secondID := uuid.New().String()
    db.events = append(db.events,
        event{id: firstID, typ: "ticket_created", payload: []byte(`{"id":"1"}`), createdAt: ts},
        event{id: secondID, typ: "ticket_updated", payload: []byte(`{"id":"1"}`), createdAt: ts},
    )

    a := apppkg.NewApp(apppkg.Config{Env: "test", TestBypassAuth: true}, db, nil, nil, nil)
    a.R.GET("/events", authpkg.Middleware(a), Stream(a))

    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/events", nil)
    req.Header.Set("Last-Event-ID", firstID)
    ctx, cancel := context.WithCancel(context.Background())
    req = req.WithContext(ctx)

    done := make(chan struct{})
    go func() {
        a.R.ServeHTTP(rr, req)
        close(done)
    }()
    time.Sleep(20 * time.Millisecond)
    cancel()
    <-done

    body := rr.Body.String()
    if strings.Contains(body, firstID) {
        t.Fatalf("stream included old event: %s", body)
    }
    if !strings.Contains(body, secondID) {
        t.Fatalf("stream missing new equal-timestamp event: %s", body)
    }
}
