package main

import (
	"context"
	"io"
	"strings"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/emersion/go-imap"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"

	app "github.com/mark3748/helpdesk-go/cmd/api/app"
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

func (f *fakeDB) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, nil
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

func TestProcessIMAPMessage_EnqueueAck(t *testing.T) {
	db := newFakeDB()
	store := newFakeStore()
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	c := Config{MinIOBucket: "bkt"}
	if err := processIMAPMessage(context.Background(), c, db, store, rdb, []byte(sampleEmail)); err != nil {
		t.Fatalf("processIMAPMessage: %v", err)
	}
	res, err := rdb.LLen(context.Background(), "jobs").Result()
	if err != nil || res != 1 {
		t.Fatalf("expected 1 job enqueued, got %d err %v", res, err)
	}
}

type bytesLiteral struct {
	data []byte
	idx  int
}

func (l *bytesLiteral) Read(p []byte) (int, error) {
	if l.idx >= len(l.data) {
		return 0, io.EOF
	}
	n := copy(p, l.data[l.idx:])
	l.idx += n
	return n, nil
}

func (l *bytesLiteral) Len() int { return len(l.data) - l.idx }

type fakeIMAPClient struct {
	raw    []byte
	stored bool
}

func (f *fakeIMAPClient) Login(username, password string) error { return nil }
func (f *fakeIMAPClient) Select(mailbox string, readOnly bool) (*imap.MailboxStatus, error) {
	return &imap.MailboxStatus{Messages: 1}, nil
}
func (f *fakeIMAPClient) Search(criteria *imap.SearchCriteria) ([]uint32, error) {
	return []uint32{1}, nil
}
func (f *fakeIMAPClient) Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	// Use the same body section requested by pollIMAP so msg.GetBody can find it.
	var section *imap.BodySectionName
	for _, it := range items {
		if s, err := imap.ParseBodySectionName(it); err == nil {
			section = s
			break
		}
	}
	if section == nil {
		section = &imap.BodySectionName{}
	}
	msg := &imap.Message{SeqNum: 1, Body: map[*imap.BodySectionName]imap.Literal{section: &bytesLiteral{data: f.raw}}}
	ch <- msg
	close(ch)
	return nil
}
func (f *fakeIMAPClient) Store(seqset *imap.SeqSet, item imap.StoreItem, value interface{}, ch chan *imap.Message) error {
	f.stored = true
	return nil
}
func (f *fakeIMAPClient) Logout() error { return nil }

func TestPollIMAP(t *testing.T) {
	raw := []byte(sampleEmail)
	cli := &fakeIMAPClient{raw: raw}
	dialOrig := dialIMAP
	dialIMAP = func(addr string) (imapClient, error) { return cli, nil }
	defer func() { dialIMAP = dialOrig }()

	var processed []byte
	procOrig := processIMAP
	processIMAP = func(ctx context.Context, c Config, db app.DB, store app.ObjectStore, rdb *redis.Client, r []byte) error {
		processed = r
		return nil
	}
	defer func() { processIMAP = procOrig }()

	if err := pollIMAP(context.Background(), Config{IMAPHost: "host", IMAPUser: "u", IMAPPass: "p", IMAPFolder: "INBOX"}, nil, nil, nil); err != nil {
		t.Fatalf("pollIMAP: %v", err)
	}
	if string(processed) != string(raw) {
		t.Fatalf("processIMAP called with wrong data")
	}
	if !cli.stored {
		t.Fatalf("expected store called")
	}
}
