// Package dbtest gives tests the test database, emptied and migrated.
package dbtest


import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)


// --- CONSTANTS ---

// defaultURL is TEST_DATABASE_URL when unset: the empty database infra/local
// creates for tests, never the one the service uses.
const defaultURL = "postgres://indexer:goodpassword@127.0.0.1:5432/" +
                   "ecoa_indexer_test?sslmode=disable"


// --- CODE ---

// URL returns the test database seen through a schema of the test's own,
// dropped when the test ends, or skips the test without a test database.
//
// Each test has its own schema so that tests in different packages, which go
// test runs at the same time, never see or erase each other's tables.
func URL(t *testing.T) string {
	t.Helper()

	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		base = defaultURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Skipf("no test database (start infra/local): %v", err)
	}
	defer conn.Close(context.Background())

	schema := "test_" + hex.EncodeToString(randomBytes(8))

	if _, err := conn.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	t.Cleanup(func() { drop(base, schema) })

	return withSearchPath(t, base, schema)
}


// withSearchPath returns base with every connection's search_path on schema.
func withSearchPath(t *testing.T, base, schema string) string {
	t.Helper()

	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("TEST_DATABASE_URL: %v", err)
	}

	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()

	return u.String()
}


// drop removes schema and everything in it.
func drop(base, schema string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, base)
	if err != nil {
		return
	}
	defer conn.Close(context.Background())

	conn.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
}


// randomBytes returns n bytes from the system's secure random source.
func randomBytes(n int) []byte {
	b := make([]byte, n)

	// never fails: crypto/rand panics rather than return short
	rand.Read(b)

	return b
}


// Pool returns a pool over url, closed with the test.
func Pool(t *testing.T, url string) *pgxpool.Pool {
	t.Helper()

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}
