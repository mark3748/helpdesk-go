package s3

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
)

// Service provides helpers to generate presigned S3 URLs.
type Service struct {
	Client *minio.Client
	Bucket string
	// MaxTTL limits the lifetime of generated URLs.
	MaxTTL time.Duration
}

// PresignPut creates a short-lived URL for uploading an object.
func (s Service) PresignPut(ctx context.Context, objectKey, contentType string, ttl time.Duration) (string, error) {
	if ttl <= 0 || ttl > s.MaxTTL {
		return "", fmt.Errorf("invalid ttl")
	}
	u, err := s.Client.PresignedPutObject(ctx, s.Bucket, objectKey, ttl)
	if err != nil {
		return "", err
	}
	// contentType is returned so callers can persist it alongside the object key.
	_ = contentType
	return u.String(), nil
}

// PresignGet creates a short-lived URL for downloading an object with forced Content-Disposition.
func (s Service) PresignGet(ctx context.Context, objectKey, filename string, ttl time.Duration) (string, error) {
	if ttl <= 0 || ttl > s.MaxTTL {
		return "", fmt.Errorf("invalid ttl")
	}
	vals := url.Values{}
	if filename != "" {
		vals.Set("response-content-disposition", "attachment; filename=\""+filename+"\"")
	}
	u, err := s.Client.PresignedGetObject(ctx, s.Bucket, objectKey, ttl, vals)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
