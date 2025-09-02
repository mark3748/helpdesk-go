package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/minio/minio-go/v7"
)

// fakeObjectStore stores objects in memory and serves them over HTTP.
type fakeObjectStore struct {
	objects map[string][]byte
	srv     *httptest.Server
}

func newFakeObjectStore() *fakeObjectStore {
	fos := &fakeObjectStore{objects: map[string][]byte{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
		if len(parts) != 2 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		key := parts[1]
		if b, ok := fos.objects[key]; ok {
			_, _ = w.Write(b)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	fos.srv = httptest.NewServer(mux)
	return fos
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

func (f *fakeObjectStore) URL() string { return f.srv.URL }
func (f *fakeObjectStore) Close()      { f.srv.Close() }

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
}

func (db *exportDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return &ticketRows{data: db.tickets}, nil
}
func (db *exportDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return nil
}
func (db *exportDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func TestExportTickets(t *testing.T) {
	store := newFakeObjectStore()
	defer store.Close()

	db := &exportDB{tickets: []ticket{{ID: "1", Number: "TKT-1", Title: "First", Status: "Open", Priority: 1}}}
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
	res, err := http.Get(url)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	got := strings.TrimSpace(string(b))
	want := "id,number,title,status,priority\n1,TKT-1,First,Open,1"
	if got != want {
		t.Fatalf("csv mismatch: got %q want %q", got, want)
	}
}
