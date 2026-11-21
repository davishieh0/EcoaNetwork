package event


import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)


// --- CODE ---

// get calls GET /events/{eventId} through a router, as the API mounts it.
func get(t *testing.T, pool *pgxpool.Pool, eventID string) *httptest.ResponseRecorder {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /events/{eventId}", NewHandler(NewRepository(pool)).Get)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
	                                       "/events/"+eventID,
	                                       nil))

	return rec
}


func TestGetReturnsTheEnvelopeByteForByte(t *testing.T) {
	// Arrange
	pool := migrated(t)
	insert(t, pool, id("b"), tricky, true)

	// Act
	rec := get(t, pool, id("b"))

	// Assert
	var body struct {
		Envelope string         `json:"envelope"`
		Derived  map[string]any `json:"derived"`
		Observed struct {
			ReceivedAt string `json:"receivedAt"`
		} `json:"observed"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}

	if rec.Code != http.StatusOK || !bytes.Equal([]byte(body.Envelope), tricky) {
		t.Fatalf("GET = %d, envelope %q, want %q",
		         rec.Code, body.Envelope, tricky)
	}

	if body.Derived == nil || body.Observed.ReceivedAt != "2026-11-21T14:02:11Z" {
		t.Fatalf("derived = %v, observed = %+v", body.Derived, body.Observed)
	}
}


func TestGetOmitsReceivedAtWithoutAnObservation(t *testing.T) {
	// Arrange
	pool := migrated(t)
	insert(t, pool, id("b"), tricky, false)

	// Act
	rec := get(t, pool, id("b"))

	// Assert
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"observed":{}`)) {
		t.Fatalf("GET = %q, want an empty observed", rec.Body.String())
	}
}


func TestGetAnswers404ForAnUnknownEvent(t *testing.T) {
	// Arrange
	pool := migrated(t)

	// Act
	rec := get(t, pool, id("b"))

	// Assert
	if rec.Code != http.StatusNotFound ||
	   !bytes.Contains(rec.Body.Bytes(), []byte(`"error":"event_not_found"`)) {
		t.Fatalf("GET = %d %q", rec.Code, rec.Body.String())
	}
}


func TestGetAnswers400ForAMalformedID(t *testing.T) {
	for _, bad := range []string{"evt_short", "abc", id("B"), id("1")} {
		t.Run(bad, func(t *testing.T) {
			// Act: no database is needed to refuse it
			rec := get(t, nil, bad)

			// Assert
			if rec.Code != http.StatusBadRequest ||
			   !bytes.Contains(rec.Body.Bytes(),
			                   []byte(`"error":"invalid_event_id"`)) {
				t.Fatalf("GET %s = %d %q", bad, rec.Code, rec.Body.String())
			}
		})
	}
}


func TestGetAnswers503WhenTheDatabaseIsUnreachable(t *testing.T) {
	// Arrange: a pool over a port nothing listens on
	pool, err := pgxpool.New(t.Context(),
		"postgres://indexer:x@127.0.0.1:1/ecoa_indexer?connect_timeout=1")
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	// Act
	rec := get(t, pool, id("b"))

	// Assert
	if rec.Code != http.StatusServiceUnavailable ||
	   !bytes.Contains(rec.Body.Bytes(), []byte(`"error":"database_unavailable"`)) {
		t.Fatalf("GET = %d %q", rec.Code, rec.Body.String())
	}
}
