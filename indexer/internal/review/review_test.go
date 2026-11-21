package review


import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/database"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/database/dbtest"
	"github.com/jackc/pgx/v5/pgxpool"
)


// --- CONSTANTS ---

// bakery and market are the entities the tests review.
const (
	bakery = "cnpj:12345678000199"
	market = "cnpj:98765432000188"
)


// --- GLOBALS ---

// day0 is when the oldest test review was written.
var day0 = time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)


// --- CODE ---

// evt returns a well-formed eventId named after name.
func evt(name string) string {
	digest := sha256.Sum256([]byte(name))
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).
		EncodeToString(digest[:])

	return "evt_" + strings.ToLower(encoded)
}


// envelope is the stored bytes of the version named name, with what a JSON
// parser on the way would change: spacing and an escape.
func envelope(name string) []byte {
	return []byte(`{"name":"` + name + `",  "body":"café"}`)
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


// version stores the version name of the chain root, editing replaces when
// it is not "", written on day0 plus day days.
func version(t *testing.T,
             pool *pgxpool.Pool,
             name, root, replaces, entity string,
             day int) {
	t.Helper()

	ctx := context.Background()
	at := day0.AddDate(0, 0, day)

	_, err := pool.Exec(ctx,
		`INSERT INTO events (event_id, event_type, author, created_at, envelope)
		 VALUES ($1, 'review.created', 'did:key:z6Mk', $2, $3)`,
		evt(name), at, envelope(name))
	if err != nil {
		t.Fatalf("insert event %s: %v", name, err)
	}

	var replaced *string
	if replaces != "" {
		id := evt(replaces)
		replaced = &id
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO reviews (event_id, chain_root, replaces, author, entity,
		                      rating, body, created_at)
		 VALUES ($1, $2, $3, 'did:key:z6Mk', $4, 4, 'body', $5)`,
		evt(name), evt(root), replaced, entity, at)
	if err != nil {
		t.Fatalf("insert review %s: %v", name, err)
	}
}


// chain stores the state of the chain root, with head as its current version.
func chain(t *testing.T,
           pool *pgxpool.Pool,
           root, entity, head string,
           day int,
           withdrawn, truncated bool) {
	t.Helper()

	var current *string
	if head != "" {
		id := evt(head)
		current = &id
	}

	_, err := pool.Exec(context.Background(),
		`INSERT INTO review_chains (chain_root, entity, head, withdrawn,
		                            truncated, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		evt(root), entity, current, withdrawn, truncated,
		day0.AddDate(0, 0, day))
	if err != nil {
		t.Fatalf("insert chain %s: %v", root, err)
	}
}


// review stores a chain of one version, written on day0 plus day days.
func review(t *testing.T, pool *pgxpool.Pool, name, entity string, day int) {
	t.Helper()

	version(t, pool, name, name, "", entity, day)
	chain(t, pool, name, entity, name, day, false, false)
}


// call answers path through the routes of the reviews, as the API mounts them.
func call(t *testing.T, pool *pgxpool.Pool, path string) *httptest.ResponseRecorder {
	t.Helper()

	h := NewHandler(NewRepository(pool))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /entities/{entity}/reviews", h.ListByEntity)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

	return rec
}


// answer is any answer of the review routes, decoded.
type answer struct {
	Items    []gotItem `json:"items"`
	Next     *string   `json:"next"`
	Review   gotItem   `json:"review"`
	Versions []gotItem `json:"versions"`
	Error    string    `json:"error"`
}


// gotItem is one item of an answer, decoded.
type gotItem struct {
	Envelope string `json:"envelope"`
	Derived  struct {
		ChainRoot string `json:"chainRoot"`
		Head      bool   `json:"head"`
		Withdrawn bool   `json:"withdrawn"`
		Edited    *bool  `json:"edited"`
	} `json:"derived"`
	Observed map[string]any `json:"observed"`
}


// decode reads rec's body, or stops the test.
func decode(t *testing.T, rec *httptest.ResponseRecorder) answer {
	t.Helper()

	var a answer
	if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}

	return a
}


// names returns the version named in each item's envelope, in order.
func names(items []gotItem) []string {
	var got []string

	for _, it := range items {
		var e struct{ Name string }
		json.Unmarshal([]byte(it.Envelope), &e)
		got = append(got, e.Name)
	}

	return got
}


// list returns the path of the bakery's listing with query.
func list(query string) string {
	return "/entities/" + bakery + "/reviews" + query
}


// same reports whether got and want hold the same names in the same order.
func same(got, want []string) bool {
	return fmt.Sprint(got) == fmt.Sprint(want)
}
