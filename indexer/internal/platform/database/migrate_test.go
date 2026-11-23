package database


import (
	"bytes"
	"context"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/database/dbtest"
	"github.com/jackc/pgx/v5"
)


// --- CODE ---

// connect opens a connection to url, closed with the test.
func connect(t *testing.T, url string) *pgx.Conn {
	t.Helper()

	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	t.Cleanup(func() { conn.Close(context.Background()) })

	return conn
}


func TestMigrateCreatesTheBaseTables(t *testing.T) {
	// Arrange
	url := dbtest.URL(t)

	// Act
	err := Migrate(context.Background(), url)

	// Assert
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	conn := connect(t, url)

	for _, table := range []string{
		"events",
		"event_observations",
		"projector_cursors",
	} {
		var found bool

		err := conn.QueryRow(context.Background(),
			"SELECT to_regclass($1) IS NOT NULL", table).Scan(&found)
		if err != nil || !found {
			t.Errorf("table %s: found = %v, %v", table, found, err)
		}
	}
}


func TestMigrateTwiceChangesNothing(t *testing.T) {
	// Arrange
	url := dbtest.URL(t)

	if err := Migrate(context.Background(), url); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}

	// Act
	err := Migrate(context.Background(), url)

	// Assert
	if err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}


func TestAnEnvelopeComesBackByteForByte(t *testing.T) {
	// Arrange
	url := dbtest.URL(t)

	if err := Migrate(context.Background(), url); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	conn := connect(t, url)

	// what jsonb would change: key order, spacing, escapes, a big number
	envelope := []byte(`{"z":1,  "a":"café", "n":12345678901234567890}`)

	_, err := conn.Exec(context.Background(),
		`INSERT INTO events (event_id, event_type, author, created_at, envelope)
		 VALUES ('evt_a', 'review.created', 'did:key:z6Mk',
		         '2026-11-18T12:00:00Z', $1)`,
		envelope)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Act
	var got []byte

	err = conn.QueryRow(context.Background(),
		"SELECT envelope FROM events WHERE event_id = 'evt_a'").Scan(&got)

	// Assert
	if err != nil || !bytes.Equal(got, envelope) {
		t.Fatalf("envelope = %q, %v, want %q", got, err, envelope)
	}
}
