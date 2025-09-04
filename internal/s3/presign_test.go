package s3

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func newClient(t *testing.T) *minio.Client {
	t.Helper()
	mc, err := minio.New("localhost:9000", &minio.Options{Creds: credentials.NewStaticV4("k", "s", ""), Secure: false, Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	return mc
}

func TestPresignPutTTL(t *testing.T) {
	svc := Service{Client: newClient(t), Bucket: "bucket", MaxTTL: time.Minute}
	if _, err := svc.PresignPut(context.Background(), "k", "text/plain", 0); err == nil {
		t.Fatal("expected error for ttl <=0")
	}
	if _, err := svc.PresignPut(context.Background(), "k", "text/plain", time.Minute*2); err == nil {
		t.Fatal("expected error for ttl > MaxTTL")
	}
	u, err := svc.PresignPut(context.Background(), "k", "text/plain", 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	uu, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	if exp := uu.Query().Get("X-Amz-Expires"); exp != "30" {
		t.Fatalf("expected expires=30, got %s", exp)
	}
}

func TestPresignGetDisposition(t *testing.T) {
	svc := Service{Client: newClient(t), Bucket: "bucket", MaxTTL: time.Minute}
	u, err := svc.PresignGet(context.Background(), "k", "file.txt", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	uu, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	if cd := uu.Query().Get("response-content-disposition"); cd != "attachment; filename=\"file.txt\"" {
		t.Fatalf("unexpected content-disposition %s", cd)
	}
}
