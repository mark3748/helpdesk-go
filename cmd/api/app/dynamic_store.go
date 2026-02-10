package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// DynamicObjectStore wraps another store (fallback) and checks the DB for
// runtime configuration. If DB config exists, it uses a cached MinIO client.
type DynamicObjectStore struct {
	DB       DB
	Fallback ObjectStore

	mu sync.RWMutex
	// ...
	cached       *minio.Client
	cachedBucket string
	cachedCfg    string // simple hash/string of config to detect changes
	lastChecked  time.Time
}

// Ensure interface compliance
var _ ObjectStore = (*DynamicObjectStore)(nil)

func (d *DynamicObjectStore) getClient(ctx context.Context) (ObjectStore, string, error) {
	// Fast path: capture current state under read lock
	d.mu.RLock()
	client := d.cached
	bucket := d.cachedBucket
	last := d.lastChecked
	d.mu.RUnlock()

	// Respect TTL for both positive (client exists) and negative (no config) caching
	// Note: using captured values is safe; state may have changed but this is just an optimization
	if time.Since(last) < 5*time.Second {
		if client != nil {
			return client, bucket, nil
		}
		// Return fallback during negative cache window
		return d.Fallback, "", nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Double check with write lock (re-read actual values to avoid race)
	if time.Since(d.lastChecked) < 5*time.Second {
		if d.cached != nil {
			return d.cached, d.cachedBucket, nil
		}
		return d.Fallback, "", nil
	}

	// Load settings
	if d.DB == nil {
		return d.Fallback, "", nil
	}

	var storageRaw []byte
	err := d.DB.QueryRow(ctx, "select storage from settings where id=1").Scan(&storageRaw)
	if err != nil {
		// Cache the "no config" state to avoid repeated DB queries
		// Clear any stale cached client to prevent serving old credentials
		d.cached = nil
		d.cachedBucket = ""
		d.lastChecked = time.Now()
		if err == pgx.ErrNoRows {
			return d.Fallback, "", nil
		}
		// On error, fallback (log it?)
		return d.Fallback, "", nil
	}

	if len(storageRaw) == 0 {
		// Cache the "empty config" state
		// Clear any stale cached client to prevent serving old credentials
		d.cached = nil
		d.cachedBucket = ""
		d.lastChecked = time.Now()
		return d.Fallback, "", nil
	}

	var cfg map[string]string
	if err := json.Unmarshal(storageRaw, &cfg); err != nil {
		// Cache the "invalid config" state
		// Clear any stale cached client to prevent serving old credentials
		d.cached = nil
		d.cachedBucket = ""
		d.lastChecked = time.Now()
		return d.Fallback, "", nil
	}

	endpoint := cfg["endpoint"]
	access := cfg["access_key_id"]
	secret := cfg["secret_access_key"]
	bucket = cfg["bucket"]
	useSSL := cfg["use_ssl"] == "true"

	// Auto-detect SSL scheme
	if strings.HasPrefix(endpoint, "https://") {
		endpoint = strings.TrimPrefix(endpoint, "https://")
		useSSL = true
	} else if strings.HasPrefix(endpoint, "http://") {
		endpoint = strings.TrimPrefix(endpoint, "http://")
		useSSL = false
	}

	if endpoint == "" || bucket == "" {
		// Cache the "incomplete config" state
		// Clear any stale cached client to prevent serving old credentials
		d.cached = nil
		d.cachedBucket = ""
		d.lastChecked = time.Now()
		return d.Fallback, "", nil
	}

	// Re-create client if config changed
	// Construct a config hash or string to compare
	cfgStr := fmt.Sprintf("%s|%s|%s|%v", endpoint, access, secret, useSSL)
	if d.cached != nil && d.cachedCfg == cfgStr {
		d.lastChecked = time.Now()
		// Update bucket in case it changed but other config didn't?
		// Actually bucket is NOT part of cfgStr above, so if ONLY bucket changes, we might miss it.
		// Let's include bucket in cfgStr to be safe.
		// But wait, if I include bucket in cfgStr, then cfgStr changes, so we re-create client.
		// Re-creating client just for bucket change is wasteful but safe.
		// Better: update cachedBucket if cfgStr matches but bucket differs?
		// No, let's just include bucket in cfgStr.
		// cfgStr := fmt.Sprintf("%s|%s|%s|%v|%s", endpoint, access, secret, useSSL, bucket)
		// But I need to change cfgStr line below.
		d.cachedBucket = bucket // Ensure it's updated if we return early?
		return d.cached, bucket, nil
	}

	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(access, secret, ""),
		Secure: useSSL,
	})
	if err != nil {
		// Cache the "client creation failure" state
		// Clear any stale cached client to prevent serving old credentials
		d.cached = nil
		d.cachedBucket = ""
		d.lastChecked = time.Now()
		return d.Fallback, "", nil
	}

	d.cached = mc
	d.cachedBucket = bucket
	d.cachedCfg = cfgStr
	d.lastChecked = time.Now()

	return mc, bucket, nil
}

// DynamicObjectStore may override the caller-provided bucket name with a bucket
// that is resolved from runtime configuration (for example, loaded from the DB).
// The resolve method always returns the effective bucket name to use so that
// wrapper methods can delegate to the underlying store correctly.
// PutObject delegates
func (d *DynamicObjectStore) PutObject(ctx context.Context, bucketName string, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	store, realBucket, err := d.resolve(ctx, bucketName)
	if err != nil {
		return minio.UploadInfo{}, err
	}
	// override bucket if dynamic
	return store.PutObject(ctx, realBucket, objectName, reader, objectSize, opts)
}

func (d *DynamicObjectStore) RemoveObject(ctx context.Context, bucketName string, objectName string, opts minio.RemoveObjectOptions) error {
	store, realBucket, err := d.resolve(ctx, bucketName)
	if err != nil {
		return err
	}
	return store.RemoveObject(ctx, realBucket, objectName, opts)
}

func (d *DynamicObjectStore) resolve(ctx context.Context, defaultBucket string) (ObjectStore, string, error) {
	store, dynBucket, err := d.getClient(ctx)
	if err != nil {
		return d.Fallback, defaultBucket, err
	}
	if dynBucket != "" {
		return store, dynBucket, nil
	}
	return d.Fallback, defaultBucket, nil
}

// StatObject delegates to the active store
func (d *DynamicObjectStore) StatObject(ctx context.Context, bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error) {
	store, realBucket, err := d.resolve(ctx, bucketName)
	if err != nil {
		return minio.ObjectInfo{}, err
	}
	return store.StatObject(ctx, realBucket, objectName, opts)
}

// PresignedPutObject delegates to the active store
func (d *DynamicObjectStore) PresignedPutObject(ctx context.Context, bucketName, objectName string, expiry time.Duration) (*url.URL, error) {
	store, realBucket, err := d.resolve(ctx, bucketName)
	if err != nil {
		return nil, err
	}
	return store.PresignedPutObject(ctx, realBucket, objectName, expiry)
}

// NOTE: `FsObjectStore` does NOT support presigned URLs.
// The `Get` handler serves files directly for FS.
// Only `Presign` (upload) supports FS via a hack (returning internal URL).
//
// If we want to hide FS specifics, `PresignedGet` for FS could return an internal URL?
// `Get` handler redirects to `PresignedGet` result.
// If FS returns `nil` for `PresignedGet`, the handler needs to know to serve bytes.
//
// Maybe `Get` handler keeps "Serve bytes" logic but uses interface to "Get content"?
// But MinIO redirects.
//
// Let's stick to the interface approach.
// `PresignedGet` -> returns URL.
// For FS, it could return empty URL?
//
// Handlers `Get`:
// url, err := store.PresignedGet(...)
// if url != nil { redirect(url); return }
// // else fallback to serving content?
// reader, err := store.GetObject(...)
// copy(writer, reader)
//
// MinIO `GetObject` streams content.
// `FsObjectStore` `GetObject` opens file.
//
// This unifies everything!
//
// But I need to implement `GetObject` for MinIO too.
//
// For now, to minimize changes:
// `DynamicObjectStore` can have a method `Unwrap() any`?
// No, that defeats the purpose if underlying changes.
//
// Let's add `PresignedGet` and `PresignedPut` to interface.
// For `FsObjectStore`, `PresignedGet` returns error "not supported".
// `DynamicObjectStore` delegates.
//
// Handler `Get`:
// u, err := a.M.PresignedGet(...)
// if err == nil { redirect(u); return }
// // if error is "not supported" (FS), proceed to file serving.
// // How to serve file? `StatObject` + `io.Reader`?
// // `FsObjectStore` needs to expose `Reader`?
//
// This is getting complicated for a quick fix.
//
// ALTERNATIVE:
// `DynamicObjectStore` embeds `*minio.Client`? No, it switches.
//
// Let's look at `handlers/attachments.go` again.
// `Get` handler checks `if mc, ok := a.M.(*minio.Client)` then `if fs, ok := a.M.(*app.FsObjectStore)`.
//
// If I configure `DynamicObjectStore` to have `IsSelectable() bool`?
//
// What if `DynamicObjectStore` implements `ObjectStore` AND we update handlers to use:
// `store := a.M`
// `if dyn, ok := store.(*DynamicObjectStore); ok { store = dyn.Current(ctx) }`
// Then `store` is either `*minio.Client` or `*FsObjectStore`.
//
// This seems easiest!
// `DynamicObjectStore.Current(ctx)` returns the underlying `ObjectStore` (MinIO or FS).
// Then the existing type switches in handlers `store.(*minio.Client)` will work!
//
// So:
// 1. Create `DynamicObjectStore` struct.
// 2. Add `Current(ctx) ObjectStore` method.
// 3. In handlers, resolve:
//    `store := a.M`
//    `if d, ok := store.(*app.DynamicObjectStore); ok { store = d.Current(c.Request.Context()) }`
// 4. Then proceed with `if mc, ok := store.(*minio.Client) ...`
//
// This requires:
// - Making `DynamicObjectStore` public in `app` package.
// - Updating 4 handlers: `Get`, `Upload`, `Presign`, `Finalize`, `Delete`, `UploadObject` (maybe).
// - And `features.go`.
//
// This requires minimal interface changes (none!).
// I just need to make sure `DynamicObjectStore` implements the EXISTING `ObjectStore` interface (Put, Remove) so it can be assigned to `a.M`.
// And `Put/Remove` on `DynamicObjectStore` should delegate too (for cases where we just call method).
//
// Implementation Details for `DynamicObjectStore`:
// - `cachedBucket` string.
// - `Current(ctx)` returns `ObjectStore`.
// - `GetBucket(ctx)` returns the bucket name to use.
//
// But wait, the handlers access `a.Cfg.MinIOBucket`.
// If I use dynamic store, I need the dynamic bucket name.
//
// `DynamicObjectStore` should probably ALSO expose the bucket name.
// `GetBucketName(ctx, defaultName string) string`.
//
// So in handlers:
// `store, bucket := app.ResolveStore(ctx, a.M, a.Cfg.MinIOBucket)`
//
// I can add a helper `ResolveStore` in `app` package.
//
// `Modify app/app.go` to add `ResolveStore`.
// `Modify app/dynamic_store.go` to add struct.
// `Modify properties` of `DynamicObjectStore` to be thread-safe cache.
//
// The existing `ObjectStore` interface:
// 	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
// 	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
//
// `DynamicObjectStore` must implement these.
// And checking `IsConfigured`.
