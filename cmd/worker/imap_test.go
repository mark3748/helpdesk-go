package main

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/minio/minio-go/v7"
)

// fakeStore implements app.ObjectStore for tests.
type fakeStore struct{ objects map[string][]byte }

func newFakeStore() *fakeStore { return &fakeStore{objects: make(map[string][]byte)} }

func (f *fakeStore) PutObject(ctx context.Context, bucket, object string, r io.Reader, size int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	b, _ := io.ReadAll(r)
	f.objects[bucket+"/"+object] = b
	return minio.UploadInfo{Bucket: bucket, Key: object, Size: int64(len(b))}, nil
}

func (f *fakeStore) RemoveObject(ctx context.Context, bucket, object string, opts minio.RemoveObjectOptions) error {
	delete(f.objects, bucket+"/"+object)
	return nil
}

// fakeDB implements app.DB for tests.
type fakeDB struct {
	tickets     int64
	attachments int
	inbound     map[string]int64
}

func newFakeDB() *fakeDB { return &fakeDB{inbound: make(map[string]int64)} }

func (f *fakeDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

type fakeRow struct {
	val int64
	err error
}

func (r fakeRow) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) > 0 {
		if p, ok := dest[0].(*int64); ok {
			*p = r.val
		}
	}
	return nil
}

func (f *fakeDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if strings.HasPrefix(sql, "select ticket_id from email_inbound") {
		id, ok := f.inbound[args[0].(string)]
		if ok {
			return fakeRow{val: id}
		}
		return fakeRow{err: pgx.ErrNoRows}
	}
	if strings.HasPrefix(sql, "insert into tickets") {
		f.tickets++
		return fakeRow{val: f.tickets}
	}
	return fakeRow{err: pgx.ErrNoRows}
}

func (f *fakeDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	if strings.HasPrefix(sql, "insert into attachments") {
		f.attachments++
	}
	if strings.HasPrefix(sql, "insert into email_inbound") {
		msgID := args[2].(string)
		ticketID := args[3].(int64)
		f.inbound[msgID] = ticketID
	}
	return pgconn.CommandTag{}, nil
}

const sampleEmail = "Subject: Test\r\nFrom: sender@example.com\r\nMessage-Id: <msg1@example.com>\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=BOUND\r\n\r\n--BOUND\r\nContent-Type: text/plain\r\n\r\nhello body\r\n--BOUND\r\nContent-Type: text/plain; name=\"note.txt\"\r\nContent-Disposition: attachment; filename=\"note.txt\"\r\n\r\nattachment data\r\n--BOUND--\r\n"

func TestProcessIMAPMessage_Attachment(t *testing.T) {
	db := newFakeDB()
	store := newFakeStore()
	c := Config{MinIOBucket: "bkt"}
	if err := processIMAPMessage(context.Background(), c, db, store, nil, []byte(sampleEmail)); err != nil {
		t.Fatalf("processIMAPMessage: %v", err)
	}
	if db.tickets != 1 {
		t.Fatalf("expected ticket created, got %d", db.tickets)
	}
	if db.attachments != 1 {
		t.Fatalf("expected 1 attachment, got %d", db.attachments)
	}
	if len(store.objects) != 2 { // raw email + attachment
		t.Fatalf("expected 2 stored objects, got %d", len(store.objects))
	}
}

func TestProcessIMAPMessage_Duplicate(t *testing.T) {
	db := newFakeDB()
	db.inbound["<msg1@example.com>"] = 1
	store := newFakeStore()
	c := Config{MinIOBucket: "bkt"}
	if err := processIMAPMessage(context.Background(), c, db, store, nil, []byte(sampleEmail)); err != nil {
		t.Fatalf("processIMAPMessage: %v", err)
	}
	if db.tickets != 0 {
		t.Fatalf("expected no ticket, got %d", db.tickets)
	}
	if db.attachments != 0 {
		t.Fatalf("expected no attachments, got %d", db.attachments)
	}
	if len(store.objects) != 0 {
		t.Fatalf("expected no stored objects, got %d", len(store.objects))
	}
}
