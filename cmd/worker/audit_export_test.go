package main

import (
    "context"
    "encoding/json"
    "strings"
    "testing"
    "time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
)

type auditEvent struct {
	ID, ActorType, ActorID, EntityType, EntityID, Action string
	At                                                   time.Time
}

type auditRows struct {
	data []auditEvent
	idx  int
}

func (r *auditRows) Close()                                       {}
func (r *auditRows) Err() error                                   { return nil }
func (r *auditRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *auditRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *auditRows) Next() bool                                   { return r.idx < len(r.data) }
func (r *auditRows) Scan(dest ...any) error {
	e := r.data[r.idx]
	r.idx++
	*(dest[0].(*string)) = e.ID
	*(dest[1].(*string)) = e.ActorType
	*(dest[2].(*string)) = e.ActorID
	*(dest[3].(*string)) = e.EntityType
	*(dest[4].(*string)) = e.EntityID
	*(dest[5].(*string)) = e.Action
	*(dest[6].(*time.Time)) = e.At
	return nil
}
func (r *auditRows) Values() ([]any, error) { return nil, nil }
func (r *auditRows) RawValues() [][]byte    { return nil }
func (r *auditRows) Conn() *pgx.Conn        { return nil }

type auditDB struct {
	events []auditEvent
}

func (db *auditDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return &auditRows{data: db.events}, nil
}

func (db *auditDB) Ping(ctx context.Context) error { return nil }

func TestHandleAuditExportJob(t *testing.T) {
	store := newFakeObjectStore()
	defer store.Close()
	mr := miniredis.RunT(t)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	ev := auditEvent{ID: "1", ActorType: "user", ActorID: "u1", EntityType: "ticket", EntityID: "t1", Action: "create", At: time.Unix(0, 0)}
	db := &auditDB{events: []auditEvent{ev}}
	cfg := Config{AuditExportBucket: "bucket"}

	handleAuditExportJob(context.Background(), cfg, db, store, rdb, "job1")

	val, err := rdb.Get(context.Background(), "audit_export:job1").Result()
	if err != nil {
		t.Fatalf("redis get: %v", err)
	}
	var st struct {
		ExportStatus
		JSONKey string `json:"json_key"`
	}
	if err := json.Unmarshal([]byte(val), &st); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if st.Status != "done" || st.ObjectKey == "" || st.JSONKey == "" {
		t.Fatalf("unexpected status: %+v", st)
	}
    // Read the CSV directly from the fake store
    got := strings.TrimSpace(string(store.objects[st.ObjectKey]))
    want := "id,actor_type,actor_id,entity_type,entity_id,action,at\n1,user,u1,ticket,t1,create,1970-01-01T00:00:00Z"
    if got != want {
        t.Fatalf("csv mismatch: got %q want %q", got, want)
    }
    jb := store.objects[st.JSONKey]
    var arr []map[string]string
    if err := json.Unmarshal(jb, &arr); err != nil {
        t.Fatalf("json decode: %v", err)
    }
	if len(arr) != 1 || arr[0]["id"] != "1" || arr[0]["action"] != "create" {
		t.Fatalf("json mismatch: %+v", arr)
	}
	lastID, _ := rdb.Get(context.Background(), "audit_export:last_id").Result()
	if lastID != "1" {
		t.Fatalf("last id mismatch: %s", lastID)
	}
}
