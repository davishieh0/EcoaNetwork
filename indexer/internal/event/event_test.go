package event


import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/database"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/database/dbtest"
	"github.com/jackc/pgx/v5/pgxpool"
)


// --- GLOBALS ---

// tricky is an envelope with what a JSON parser on the way would change: key
// order, spacing, an escape, a number beyond float64.
var tricky = []byte(`{"z":1,  "a":"café", "n":12345678901234567890}`)


// receivedAt is when the test events were observed.
var receivedAt = time.Date(2026, 11, 21, 14, 2, 11, 0, time.UTC)


// --- CODE ---

// id returns a well-formed eventId ending in c.
func id(c string) string {
	return "evt_" + strings.Repeat("a", 51) + c
}


// migrated returns a pool over the emptied, migrated test database.
func migrated(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := dbtest.URL(t)

	if err := database.Migrate(context.Background(), url); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	return dbtest.Pool(t, url)
}


// insert stores an event with envelope, observed at receivedAt when observed.
func insert(t *testing.T,
            pool *pgxpool.Pool,
            eventID string,
            envelope []byte,
            observed bool) {
	t.Helper()

	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO events (event_id, event_type, author, created_at, envelope)
		 VALUES ($1, 'review.created', 'did:key:z6Mk', $2, $3)`,
		eventID, receivedAt, envelope)
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}

	if !observed {
		return
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO event_observations (event_id, received_at) VALUES ($1, $2)`,
		eventID, receivedAt)
	if err != nil {
		t.Fatalf("insert observation: %v", err)
	}
}
