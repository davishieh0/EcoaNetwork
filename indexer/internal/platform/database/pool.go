package database


import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)


// --- CODE ---

// Open returns a pool of connections to the database at url.
//
// It does not connect yet: a database that is down does not stop the API
// from starting, and /health says so instead.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		// the error may quote the URL, password included, so it is not wrapped
		return nil, fmt.Errorf("open database: invalid DATABASE_URL")
	}

	return pool, nil
}
