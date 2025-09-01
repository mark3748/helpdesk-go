package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// fakeRow implements pgx.Row
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

// fakeRows implements pgx.Rows with minimal behavior used in code
type fakeRows struct{}

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return nil }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Next() bool                                   { return false }
func (r *fakeRows) Scan(dest ...any) error                       { return nil }
func (r *fakeRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeRows) RawValues() [][]byte                          { return nil }
func (r *fakeRows) Conn() *pgx.Conn                              { return nil }

// fakeDB satisfies the DB interface and simulates user lookup/insert and empty roles
type fakeDB struct{}

func (db *fakeDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	// roles query returns empty set
	return &fakeRows{}, nil
}
func (db *fakeDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	s := strings.ToLower(sql)
	if strings.Contains(s, "from users where external_id") {
		// simulate no existing user
		return &fakeRow{err: pgx.ErrNoRows}
	}
	if strings.HasPrefix(strings.TrimSpace(s), "insert into users") {
		// return a generated id
		return &fakeRow{scan: func(dest ...any) error {
			if len(dest) > 0 {
				if p, ok := dest[0].(*string); ok {
					*p = "00000000-0000-0000-0000-000000000001"
				}
			}
			return nil
		}}
	}
	return &fakeRow{}
}
func (db *fakeDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func TestAuth_WithJWKS_ValidToken(t *testing.T) {
	// generate RSA keypair
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa generate: %v", err)
	}

	// build JWK Set with public key
	pubJWK, err := jwk.FromRaw(&priv.PublicKey)
	if err != nil {
		t.Fatalf("jwk from raw: %v", err)
	}
	_ = pubJWK.Set("kid", "test-key")
	_ = pubJWK.Set("alg", "RS256")
	set := jwk.NewSet()
	set.AddKey(pubJWK)

	// JWKS server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(set)
	}))
	defer ts.Close()

	// keyfunc using the JWKS URL (single fetch)
	keyset, err := jwk.Fetch(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("fetch jwks: %v", err)
	}
	keyf := func(tk *jwt.Token) (any, error) {
		kid, _ := tk.Header["kid"].(string)
		if kid != "" {
			if key, ok := keyset.LookupKeyID(kid); ok {
				var pub any
				if err := key.Raw(&pub); err != nil {
					return nil, err
				}
				return pub, nil
			}
		}
		it := keyset.Iterate(context.Background())
		if it.Next(context.Background()) {
			pair := it.Pair()
			if key, ok := pair.Value.(jwk.Key); ok {
				var pub any
				if err := key.Raw(&pub); err != nil {
					return nil, err
				}
				return pub, nil
			}
		}
		return nil, jwt.ErrTokenSignatureInvalid
	}

	// sign a JWT
	claims := jwt.MapClaims{
		"iss":   "https://issuer.example",
		"sub":   "user-123",
		"email": "user@example.com",
		"name":  "User Name",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key"
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	cfg := Config{Env: "test", OIDCIssuer: "https://issuer.example"}
	app := NewApp(cfg, &fakeDB{}, keyf, nil, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	app.r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. body=%s", rr.Code, rr.Body.String())
	}
}
