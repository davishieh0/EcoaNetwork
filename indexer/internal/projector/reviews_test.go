package projector


import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/eduardokairalla1/EcoaNetwork/indexer/internal/protobuf"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/page"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/review"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/stream"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/fixturetest"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/projection"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/testevent"
	"github.com/jackc/pgx/v5/pgxpool"
)


// --- CONSTANTS ---

// bakery is the entity the tests review.
const bakery = "cnpj:12345678000199"


// --- CODE ---

// feed applies raws in order, as one node's log, or stops the test.
func feed(t *testing.T, pool *pgxpool.Pool, node string, raws ...[]byte) {
	t.Helper()

	a := applier{pool: pool}

	for i, raw := range raws {
		r := stream.Received{Node: node, Event: &pb.StoredEvent{
			Envelope:   raw,
			Sequence:   uint64(i + 1),
			ReceivedAt: time.Now().UnixNano(),
			LogId:      "log-" + node,
		}}

		if err := a.apply(r); err != nil {
			t.Fatalf("apply %d: %v", i+1, err)
		}
	}
}


// sign returns an event of eventType by alice, written at, and its eventId.
func sign(t *testing.T, eventType, at, payload string) ([]byte, string) {
	t.Helper()

	return signAs(t, 'a', eventType, at, payload)
}


// signedBy returns an event of eventType by the key of seed.
func signedBy(t *testing.T, seed byte, eventType, at, payload string) []byte {
	t.Helper()

	raw, _ := signAs(t, seed, eventType, at, payload)

	return raw
}


// signAs returns an event of eventType by the key of seed, and its eventId.
func signAs(t *testing.T,
            seed byte,
            eventType, at, payload string) ([]byte, string) {
	t.Helper()

	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, 32))

	raw, err := testevent.Sign(key, testevent.Fields{
		Protocol:  "ecoa-dev",
		Version:   1,
		EventType: eventType,
		CreatedAt: at,
		Payload:   payload,
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	env, err := envelope.Decode(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	return raw, env.EventID
}


// chainOf returns the state of the chain rooted at root.
func chainOf(t *testing.T, pool *pgxpool.Pool, root string) (head *string,
                                                              withdrawn,
                                                              truncated bool) {
	t.Helper()

	err := pool.QueryRow(context.Background(),
		`SELECT head, withdrawn, truncated FROM review_chains
		 WHERE chain_root = $1`, root).Scan(&head, &withdrawn, &truncated)
	if err != nil {
		t.Fatalf("read chain: %v", err)
	}

	return head, withdrawn, truncated
}


// targetsFirst orders built so every event comes after the one it names, as
// a node's log does; otherwise in arrival order.
func targetsFirst(built []fixturetest.Built) []fixturetest.Built {
	var ordered []fixturetest.Built

	placed := map[string]bool{}
	pending := append([]fixturetest.Built(nil), built...)

	for len(pending) > 0 {
		var later []fixturetest.Built

		for _, b := range pending {
			if target := targetOf(b.Env); target == "" || placed[target] {
				ordered = append(ordered, b)
				placed[b.Env.EventID] = true
			} else {
				later = append(later, b)
			}
		}

		pending = later
	}

	return ordered
}


// targetOf is the event env edits or withdraws, "" for a new review.
func targetOf(env *envelope.Envelope) string {
	var p struct {
		Replaces string `json:"replaces"`
		Target   string `json:"target"`
	}

	json.Unmarshal(env.Payload, &p)

	if p.Replaces != "" {
		return p.Replaces
	}

	return p.Target
}


func TestTheChainHeadFixturesThroughTheProjector(t *testing.T) {
	file := fixturetest.LoadFrom(t, "../../../protocol/fixtures", "chain-head")

	fixturetest.Run(t, file, func(t *testing.T, c fixturetest.Case) {
		// its times run past minute 59 (10:66:00Z), so its events cannot be
		// signed; TestAChainBeyondSixtyFourLinksIsTruncated covers the rule
		if c.Name == "chain beyond the 64-link cap" {
			t.Skip("the fixture's times are not valid instants")
		}

		// Arrange
		pool := migrated(t)
		built := c.Events(t)

		ids := map[string]string{}
		var root string
		var events []projection.Event

		for _, b := range built {
			ids[b.ID] = b.Env.EventID
			events = append(events, projection.Event{
				ID:        b.Env.EventID,
				Author:    b.Env.Author,
				Type:      b.Env.EventType,
				CreatedAt: b.Env.CreatedAt,
				Payload:   b.Env.Payload,
			})

			if b.Env.EventType == "review.created" {
				root = b.Env.EventID
			}
		}

		var raws [][]byte
		for _, b := range targetsFirst(built) {
			raws = append(raws, b.Raw)
		}

		// Act
		feed(t, pool, "node-a", raws...)

		// Assert: what was stored is what the rule says of these events
		head, withdrawn, truncated := chainOf(t, pool, root)
		want := projection.ResolveHead(root, events)

		if (head == nil) != (want.ID == "") || (head != nil && *head != want.ID) {
			t.Errorf("head = %v, the rule says %q", head, want.ID)
		}

		if withdrawn != want.Withdrawn || truncated != want.Truncated {
			t.Errorf("withdrawn, truncated = %v, %v; the rule says %v, %v",
			         withdrawn, truncated, want.Withdrawn, want.Truncated)
		}

		// and what the fixture expects, but for a tie on createdAt: it is
		// broken by the real eventIds, not by the order of the fixture's names
		if _, tie := c.Expect["tiebreak"]; tie {
			return
		}

		if name, ok := c.Expect["head"].(string); ok {
			if head == nil || *head != ids[name] {
				t.Errorf("head = %v, the fixture expects %s", head, name)
			}
		}

		if w, ok := c.Expect["withdrawn"].(bool); ok && withdrawn != w {
			t.Errorf("withdrawn = %v, the fixture expects %v", withdrawn, w)
		}
	})
}


func TestAChainBeyondSixtyFourLinksIsTruncated(t *testing.T) {
	// Arrange: an original and 66 edits, each of the one before, a minute apart
	pool := migrated(t)
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	created, root := sign(t, "review.created", start.Format(time.RFC3339),
		`{"entity":"`+bakery+`","rating":3,"body":"v0"}`)

	raws := [][]byte{created}
	previous := root

	for i := 1; i <= 66; i++ {
		at := start.Add(time.Duration(i) * time.Minute).Format(time.RFC3339)

		raw, id := sign(t, "review.replaced", at,
			`{"replaces":"`+previous+`","rating":3,"body":"v"}`)

		raws = append(raws, raw)
		previous = id
	}

	// Act
	feed(t, pool, "node-a", raws...)

	// Assert: no head, and nothing listed
	head, _, truncated := chainOf(t, pool, root)

	listed, _, err := review.NewRepository(pool).HeadsByEntity(
		context.Background(), bakery, page.Page{Limit: 20}, review.Hide{})

	if head != nil || !truncated || err != nil || len(listed) != 0 {
		t.Fatalf("head = %v, truncated = %v, listed = %d, %v",
		         head, truncated, len(listed), err)
	}
}


func TestAPublishedReviewIsListedEditedAndWithdrawn(t *testing.T) {
	// Arrange
	pool := migrated(t)
	reviews := review.NewRepository(pool)
	page := page.Page{Limit: 20}
	ctx := context.Background()

	created, root := sign(t, "review.created", "2026-11-01T10:00:00Z",
		`{"entity":"`+bakery+`","rating":3,"body":"bom"}`)
	edited, edit := sign(t, "review.replaced", "2026-11-02T10:00:00Z",
		`{"replaces":"`+root+`","rating":5,"body":"melhorou"}`)
	withdrawal, _ := sign(t, "review.withdrawn", "2026-11-03T10:00:00Z",
		`{"target":"`+edit+`","reason":"mistake"}`)

	// Act and Assert: created
	feed(t, pool, "node-a", created)

	listed, _, err := reviews.HeadsByEntity(ctx, bakery, page, review.Hide{})
	if err != nil || len(listed) != 1 || listed[0].EventID != root ||
	   listed[0].Edited {
		t.Fatalf("after created: %+v, %v", listed, err)
	}

	// Act and Assert: edited
	feed(t, pool, "node-b", created, edited)

	listed, _, err = reviews.HeadsByEntity(ctx, bakery, page, review.Hide{})
	if err != nil || len(listed) != 1 || listed[0].EventID != edit ||
	   !listed[0].Edited || listed[0].ChainRoot != root {
		t.Fatalf("after replaced: %+v, %v", listed, err)
	}

	// Act and Assert: withdrawn
	feed(t, pool, "node-c", created, edited, withdrawal)

	listed, _, err = reviews.HeadsByEntity(ctx, bakery, page, review.Hide{})
	if err != nil || len(listed) != 0 {
		t.Fatalf("after withdrawn: %+v, %v, want nothing listed", listed, err)
	}

	got, err := reviews.ByID(ctx, root)
	if err != nil || !got.Withdrawn {
		t.Fatalf("ByID = %+v, %v, want withdrawn", got, err)
	}
}


func TestAReviewFromTwoNodesIsProjectedOnce(t *testing.T) {
	// Arrange
	pool := migrated(t)
	created, _ := sign(t, "review.created", "2026-11-01T10:00:00Z",
		`{"entity":"`+bakery+`","rating":3,"body":"bom"}`)

	// Act
	feed(t, pool, "node-a", created)
	feed(t, pool, "node-b", created)

	// Assert
	if count(t, pool, "reviews") != 1 || count(t, pool, "review_chains") != 1 {
		t.Fatalf("reviews = %d, chains = %d, want 1 and 1",
		         count(t, pool, "reviews"), count(t, pool, "review_chains"))
	}
}


func TestAnEditOfNoKnownReviewIsSkippedAndTheCursorMoves(t *testing.T) {
	// Arrange: an edit of a version this indexer never saw
	pool := migrated(t)
	orphan, _ := sign(t, "review.replaced", "2026-11-02T10:00:00Z",
		`{"replaces":"evt_`+string(bytes.Repeat([]byte{'q'}, 52))+
		`","rating":5,"body":"x"}`)

	// Act
	feed(t, pool, "node-a", orphan)

	// Assert: stored as an event, projected nowhere, the log moves on
	if count(t, pool, "events") != 1 || count(t, pool, "reviews") != 0 ||
	   cursorOf(t, pool, "node-a").After != 1 {
		t.Fatalf("events = %d, reviews = %d, cursor = %+v",
		         count(t, pool, "events"), count(t, pool, "reviews"),
		         cursorOf(t, pool, "node-a"))
	}
}
