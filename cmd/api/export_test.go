package main

import (
    "context"
    "encoding/json"
    "errors"
    "io"
    "net/http"
    "net/http/httptest"
    "net/url"
    "strings"
    "testing"
    "time"

    miniredis "github.com/alicebob/miniredis/v2"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
    "github.com/minio/minio-go/v7"
    "github.com/redis/go-redis/v9"
)

// fakeObjectStore stores objects in memory and serves them over HTTP.
type fakeObjectStore struct { objects map[string][]byte }

func newFakeObjectStore() *fakeObjectStore {
    return &fakeObjectStore{objects: map[string][]byte{}}
}

func (f *fakeObjectStore) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
    b, err := io.ReadAll(reader)
    if err != nil {
        return minio.UploadInfo{}, err
    }
    f.objects[objectName] = b
    return minio.UploadInfo{Size: int64(len(b))}, nil
}

func (f *fakeObjectStore) RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error {
    delete(f.objects, objectName)
    return nil
}

func (f *fakeObjectStore) PresignedPutObject(ctx context.Context, bucketName, objectName string, expiry time.Duration) (*url.URL, error) {
    _ = ctx
    _ = expiry
    u, _ := url.Parse("http://fake/" + bucketName + "/" + objectName)
    return u, nil
}

func (f *fakeObjectStore) StatObject(ctx context.Context, bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error) {
    _ = ctx
    _ = opts
    b, ok := f.objects[objectName]
    if !ok {
        return minio.ObjectInfo{}, errors.New("not found")
    }
    return minio.ObjectInfo{Key: objectName, Size: int64(len(b))}, nil
}

func (f *fakeObjectStore) URL() string { return "http://fake" }
func (f *fakeObjectStore) Close()      {}

// ticket represents a row returned by the fake DB.
type ticket struct {
    ID, Number, Title, Status string
    Priority                  int16
}

type ticketRows struct {
    data []ticket
    idx  int
}

func (r *ticketRows) Close()                                       {}
func (r *ticketRows) Err() error                                   { return nil }
func (r *ticketRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *ticketRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *ticketRows) Next() bool                                   { return r.idx < len(r.data) }
func (r *ticketRows) Scan(dest ...any) error {
    t := r.data[r.idx]
    r.idx++
    *(dest[0].(*string)) = t.ID
    *(dest[1].(*string)) = t.Number
    *(dest[2].(*string)) = t.Title
    *(dest[3].(*string)) = t.Status
    *(dest[4].(*int16)) = t.Priority
    return nil
}
func (r *ticketRows) Values() ([]any, error) { return nil, nil }
func (r *ticketRows) RawValues() [][]byte    { return nil }
func (r *ticketRows) Conn() *pgx.Conn        { return nil }

// exportDB returns predefined ticket rows.
type exportDB struct {
    tickets []ticket
    count   int
}

func (db *exportDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
    return &ticketRows{data: db.tickets}, nil
}

type countRow struct{ n int }

func (r countRow) Scan(dest ...any) error {
    *(dest[0].(*int)) = r.n
    return nil
}

func (db *exportDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
    return countRow{n: db.count}
}

func (db *exportDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
    return pgconn.CommandTag{}, nil
}

func TestExportTickets(t *testing.T) {
    store := newFakeObjectStore()

    db := &exportDB{tickets: []ticket{{ID: "1", Number: "TKT-1", Title: "First", Status: "Open", Priority: 1}}, count: 1}
    cfg := Config{Env: "test", TestBypassAuth: true, MinIOEndpoint: strings.TrimPrefix(store.URL(), "http://"), MinIOBucket: "bucket"}
    app := NewApp(cfg, db, nil, store, nil)

    body := strings.NewReader(`{"ids":["1"]}`)
    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodPost, "/exports/tickets", body)
    req.Header.Set("Content-Type", "application/json")
    app.r.ServeHTTP(rr, req)

    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rr.Code)
    }
    var resp map[string]string
    if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
        t.Fatalf("invalid json: %v", err)
    }
    url := resp["url"]
    if url == "" {
        t.Fatalf("missing url in response")
    }
    // Extract objectKey and read from fake store directly (no network in CI)
    parts := strings.Split(url, "/")
    objectKey := parts[len(parts)-1]
    b := store.objects[objectKey]
    got := strings.TrimSpace(string(b))
    want := "id,number,title,status,priority\n1,TKT-1,First,Open,1"
    if got != want {
        t.Fatalf("csv mismatch: got %q want %q", got, want)
    }
}

func TestExportTicketsAsync(t *testing.T) {
    mr := miniredis.RunT(t)
    defer mr.Close()
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

    store := newFakeObjectStore()
    defer store.Close()

    db := &exportDB{count: exportSyncLimit + 1}
    cfg := Config{Env: "test", TestBypassAuth: true, MinIOEndpoint: strings.TrimPrefix(store.URL(), "http://"), MinIOBucket: "bucket"}
    app := NewApp(cfg, db, nil, store, rdb)

    body := strings.NewReader(`{"ids":["1","2"]}`)
    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodPost, "/exports/tickets", body)
    req.Header.Set("Content-Type", "application/json")
    app.r.ServeHTTP(rr, req)

    if rr.Code != http.StatusAccepted {
        t.Fatalf("expected 202, got %d", rr.Code)
    }
    var resp map[string]string
    if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
        t.Fatalf("invalid json: %v", err)
    }
    jobID := resp["job_id"]
    if jobID == "" {
        t.Fatalf("missing job_id")
    }

    // simulate worker completion
    st := struct {
        Requester string `json:"requester"`
        Status    string `json:"status"`
        URL       string `json:"url"`
    }{"test-user", "done", "http://example.com/export.csv"}
    b, _ := json.Marshal(st)
    if err := rdb.Set(context.Background(), "export_tickets:"+jobID, b, 0).Err(); err != nil {
        t.Fatalf("redis set: %v", err)
    }

    rr = httptest.NewRecorder()
    req = httptest.NewRequest(http.MethodGet, "/exports/tickets/"+jobID, nil)
    app.r.ServeHTTP(rr, req)

    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rr.Code)
    }
    resp = map[string]string{}
    if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
        t.Fatalf("invalid json: %v", err)
    }
    if resp["url"] != "http://example.com/export.csv" {
        t.Fatalf("unexpected url %q", resp["url"])
    }
}
