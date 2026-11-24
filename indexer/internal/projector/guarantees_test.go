package projector


import (
	"context"
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	pb "github.com/eduardokairalla1/EcoaNetwork/indexer/internal/protobuf"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/seed"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/seed/sample"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/stream"
	"github.com/jackc/pgx/v5/pgxpool"
)


// --- GLOBALS ---

// deterministic are the tables every honest indexer must agree on, fed the
// same events (docs/architecture.md): never the observed ones, such as
// event_observations, event_holders or claim_checks, nor the cursors.
var deterministic = []string{
	"events", "reviews", "review_chains", "review_withdrawals", "votes",
	"version_votes", "responses", "labels", "entities", "entity_proposals",
	"profiles", "key_rotations", "key_revocations", "identities", "claims",
}


// --- CODE ---

// generated is a few hundred signed events with edits, votes, responses and
// withdrawals, from the seed generator.
func generated(t *testing.T) [][]byte {
	t.Helper()

	companies, err := seed.ReadReceita(sample.Estabelecimentos(),
	                                   sample.Empresas(), 12)
	if err != nil {
		t.Fatalf("ReadReceita: %v", err)
	}

	out, err := seed.Generate(seed.Plan{
		Companies: companies,
		Reviewers: 12,
		Reviews:   50,
		Until:     time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		Days:      20,
		Seed:      11,
		Operator:  seed.KeyOf(11, "operator", 0),
		Network:   "ecoa-dev",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	return out.Envelopes
}


// shuffled is raws in another order a node could hold them in: random, but
// never an event before the one it references, as a node stores nothing
// before its target.
func shuffled(t *testing.T, raws [][]byte, seedValue int64) [][]byte {
	t.Helper()

	random := rand.New(rand.NewSource(seedValue))
	ids := make([]string, len(raws))
	needs := make([][]string, len(raws))

	for i, raw := range raws {
		var e struct {
			EventID string `json:"eventId"`
			Payload struct {
				Target   string `json:"target"`
				Replaces string `json:"replaces"`
			} `json:"payload"`
		}

		if err := json.Unmarshal(raw, &e); err != nil {
			t.Fatalf("read event %d: %v", i, err)
		}

		ids[i] = e.EventID
		for _, ref := range []string{e.Payload.Target, e.Payload.Replaces} {
			if ref != "" {
				needs[i] = append(needs[i], ref)
			}
		}
	}

	placed := map[string]bool{}
	left := random.Perm(len(raws))
	var out [][]byte

	for len(left) > 0 {
		for k, i := range left {
			ready := true
			for _, ref := range needs[i] {
				if !placed[ref] {
					ready = false
					break
				}
			}

			if ready {
				out = append(out, raws[i])
				placed[ids[i]] = true
				left = append(left[:k], left[k+1:]...)
				break
			}
		}
	}

	return out
}


// nodeOf is a fake node holding raws in its log.
func nodeOf(logID string, raws [][]byte) *fakeNode {
	events := make([]*pb.StoredEvent, len(raws))

	for i, raw := range raws {
		events[i] = &pb.StoredEvent{
			Envelope:   raw,
			Sequence:   uint64(i + 1),
			ReceivedAt: time.Date(2026, 11, 2, 0, 0, i, 0, time.UTC).UnixNano(),
			LogId:      logID,
			Head:       uint64(len(raws)),
		}
	}

	return &fakeNode{logID: logID, events: events}
}


// projectAll runs the projector over nodes until it has caught up with all
// of them, as indexer rebuild does.
func projectAll(t *testing.T, pool *pgxpool.Pool, nodes map[string]*fakeNode) {
	t.Helper()

	before := settle
	settle = 300 * time.Millisecond
	t.Cleanup(func() { settle = before })

	var followers []stream.Follower
	for name, node := range nodes {
		followers = append(followers, stream.Follower{
			Node:       name,
			Client:     serveFake(t, node),
			PositionOf: positionsFrom(pool),
			FirstWait:  10 * time.Millisecond,
			MaxWait:    50 * time.Millisecond,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := feedAll(ctx, pool, followers, true); err != nil {
		t.Fatalf("feedAll: %v", err)
	}

	if ctx.Err() != nil {
		t.Fatalf("did not catch up in time")
	}
}


// snapshot is a digest of each deterministic table's rows, in a fixed order.
func snapshot(t *testing.T, pool *pgxpool.Pool) map[string]string {
	t.Helper()

	digests := map[string]string{}

	for _, table := range deterministic {
		var digest string
		var count int

		err := pool.QueryRow(context.Background(),
			`SELECT COALESCE(md5(string_agg(t::text, '|' ORDER BY t::text)), ''),
			        count(*)
			 FROM `+table+` t`).Scan(&digest, &count)
		if err != nil {
			t.Fatalf("snapshot %s: %v", table, err)
		}

		digests[table] = digest
	}

	return digests
}


// sameTables fails the test where two snapshots differ.
func sameTables(t *testing.T, what string, got, want map[string]string) {
	t.Helper()

	for _, table := range deterministic {
		if got[table] != want[table] {
			t.Errorf("%s: table %s differs", what, table)
		}
	}
}


// countOf is how many rows table holds.
func countOf(t *testing.T, pool *pgxpool.Pool, table string) int {
	t.Helper()

	var n int
	err := pool.QueryRow(context.Background(),
	                     "SELECT count(*) FROM "+table).Scan(&n)
	if err != nil {
		t.Fatalf("count %s: %v", table, err)
	}

	return n
}


func TestARebuildReproducesTheDeterministicTables(t *testing.T) {
	// Arrange: a projection of the log
	pool := migrated(t)
	log := generated(t)
	nodes := map[string]*fakeNode{"n1": nodeOf("log-1", log)}

	projectAll(t, pool, nodes)
	before := snapshot(t, pool)

	if countOf(t, pool, "events") != len(log) ||
		countOf(t, pool, "votes") == 0 ||
		countOf(t, pool, "review_withdrawals") == 0 {
		t.Fatalf("the log did not project as expected: %d events of %d",
		         countOf(t, pool, "events"), len(log))
	}

	// Act: drop it all and read the log again, as indexer rebuild does
	if err := Empty(context.Background(), pool); err != nil {
		t.Fatalf("Empty: %v", err)
	}

	if countOf(t, pool, "events") != 0 ||
		countOf(t, pool, "projector_cursors") != 0 {
		t.Fatalf("Empty left rows behind")
	}

	projectAll(t, pool, nodes)

	// Assert
	sameTables(t, "after the rebuild", snapshot(t, pool), before)
}


func TestTwoIndexersFedTheSameEventsInOtherOrdersAgree(t *testing.T) {
	// Arrange: one indexer reads a node; the other, two nodes holding the
	// same events in other orders, read at once
	log := generated(t)

	one := migrated(t)
	two := migrated(t)

	// Act
	projectAll(t, one, map[string]*fakeNode{
		"n1": nodeOf("log-1", log),
	})
	projectAll(t, two, map[string]*fakeNode{
		"n2": nodeOf("log-2", shuffled(t, log, 1)),
		"n3": nodeOf("log-3", shuffled(t, log, 2)),
	})

	// Assert
	sameTables(t, "two indexers", snapshot(t, two), snapshot(t, one))
}


func TestApplyingTheLogTwiceChangesNothing(t *testing.T) {
	// Arrange
	pool := migrated(t)
	log := generated(t)
	projectAll(t, pool, map[string]*fakeNode{"n1": nodeOf("log-1", log)})
	once := snapshot(t, pool)

	// Act: the same events again, from a node whose log started over, and
	// straight through the applier as another node would send them
	projectAll(t, pool, map[string]*fakeNode{"n1": nodeOf("log-1b", log)})
	feed(t, pool, "again", log...)

	// Assert
	sameTables(t, "applied again", snapshot(t, pool), once)
}


func TestACrashMidwayLosesAndDoublesNothing(t *testing.T) {
	// Arrange: what one uninterrupted run gives
	log := generated(t)
	node := nodeOf("log-1", log)

	clean := migrated(t)
	projectAll(t, clean, map[string]*fakeNode{"n1": node})
	want := snapshot(t, clean)

	// Act: a run stopped halfway, as a crash stops it, then started again
	pool := migrated(t)

	followers := []stream.Follower{{
		Node:       "n1",
		Client:     serveFake(t, nodeOf("log-1", log)),
		PositionOf: positionsFrom(pool),
		FirstWait:  10 * time.Millisecond,
		MaxWait:    50 * time.Millisecond,
	}}

	ctx, crash := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		run(ctx, pool, followers)
		close(done)
	}()

	eventually(t, "half the log applied", func() bool {
		return countOf(t, pool, "events") >= len(log)/2
	})
	crash()
	<-done

	stopped := countOf(t, pool, "events")
	projectAll(t, pool, map[string]*fakeNode{"n1": nodeOf("log-1", log)})

	// Assert: it resumed from its cursor, and ended where the clean run did
	if stopped >= len(log) {
		t.Fatalf("the crash came too late: %d of %d applied", stopped,
		         len(log))
	}

	sameTables(t, "after a crash", snapshot(t, pool), want)
}
