package projector


import (
	"context"
	"strings"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/page"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/review"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/fixturetest"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/projection"
	"github.com/jackc/pgx/v5/pgxpool"
)


// --- CODE ---

// tallyOf returns the endorsements and disputes stored for version.
func tallyOf(t *testing.T, pool *pgxpool.Pool, version string) (int, int) {
	t.Helper()

	var endorsements, disputes int

	err := pool.QueryRow(context.Background(),
		`SELECT COALESCE((SELECT endorsements FROM version_votes WHERE version = $1), 0),
		        COALESCE((SELECT disputes FROM version_votes WHERE version = $1), 0)`,
		version).Scan(&endorsements, &disputes)
	if err != nil {
		t.Fatalf("read tally: %v", err)
	}

	return endorsements, disputes
}


// ruleTally is what the protocol's rule counts on version from events.
func ruleTally(version string, events []projection.Event) (int, int) {
	var endorsements, disputes int

	voters := map[string]bool{}

	for _, e := range events {
		if e.Type == "review.endorsed" || e.Type == "review.disputed" {
			voters[e.Author] = true
		}
	}

	for voter := range voters {
		vote, found := projection.CurrentVote(voter, version, events)

		switch {
		case !found:
		case vote.Type == "review.endorsed":
			endorsements++
		default:
			disputes++
		}
	}

	return endorsements, disputes
}


func TestTheVoteFixturesThroughTheProjector(t *testing.T) {
	file := fixturetest.LoadFrom(t, "../../../protocol/fixtures",
	                             "vote-and-label-binding")

	fixturetest.Run(t, file, func(t *testing.T, c fixturetest.Case) {
		// Arrange
		pool := migrated(t)
		built := c.Events(t)

		ids := map[string]string{}
		var events []projection.Event
		var raws [][]byte

		for _, b := range built {
			ids[b.ID] = b.Env.EventID
			events = append(events, projection.Event{
				ID:        b.Env.EventID,
				Author:    b.Env.Author,
				Type:      b.Env.EventType,
				CreatedAt: b.Env.CreatedAt,
				Payload:   b.Env.Payload,
			})
		}

		for _, b := range targetsFirst(built) {
			raws = append(raws, b.Raw)
		}

		// Act
		feed(t, pool, "node-a", raws...)

		// Assert: each version's tally is what the rule counts
		versions := []string{}
		if head, ok := c.Expect["head"].(string); ok {
			versions = append(versions, head)
		}

		historical, _ := c.Expect["historical"].(map[string]any)
		for name := range historical {
			versions = append(versions, name)
		}

		for _, name := range versions {
			gotE, gotD := tallyOf(t, pool, ids[name])
			wantE, wantD := ruleTally(ids[name], events)

			if gotE != wantE || gotD != wantD {
				t.Errorf("%s: tally %d/%d, the rule says %d/%d",
				         name, gotE, gotD, wantE, wantD)
			}
		}

		// and the fixture's own counts, but where a tie on createdAt makes
		// the winner depend on the real eventIds, not on the fixture's names
		if strings.Contains(c.Name, "tied") {
			return
		}

		if score, ok := c.Expect["headScore"].(map[string]any); ok {
			gotE, gotD := tallyOf(t, pool, ids[c.Expect["head"].(string)])

			if gotE != len(score["endorsed"].([]any)) ||
			   gotD != len(score["disputed"].([]any)) {
				t.Errorf("head tally %d/%d, the fixture expects %v",
				         gotE, gotD, score)
			}
		}

		for name, want := range historical {
			endorsed := want.(map[string]any)["endorsed"].([]any)

			if gotE, _ := tallyOf(t, pool, ids[name]); gotE != len(endorsed) {
				t.Errorf("%s endorsements = %d, the fixture expects %d",
				         name, gotE, len(endorsed))
			}
		}

		// a vote on something with no display surface is kept and ignored
		if stored, ok := c.Expect["stored"].([]any); ok {
			if count(t, pool, "events") != len(stored) ||
			   count(t, pool, "votes") != 0 {
				t.Errorf("events = %d, votes = %d, want %d and 0",
				         count(t, pool, "events"), count(t, pool, "votes"),
				         len(stored))
			}
		}
	})
}


func TestTheListingShowsTheVotesOnTheHeadAndAnEditStartsOver(t *testing.T) {
	// Arrange: two endorsements and a dispute on the first version
	pool := migrated(t)
	ctx := context.Background()
	page := page.Page{Limit: 20}

	created, root := sign(t, "review.created", "2026-11-01T10:00:00Z",
		`{"entity":"`+bakery+`","rating":3,"body":"bom"}`)

	raws := [][]byte{created}

	for i, kind := range []string{"endorsed", "endorsed", "disputed"} {
		raw := signedBy(t, byte('b'+i), "review."+kind, "2026-11-01T11:00:00Z",
			`{"target":"`+root+`"}`)
		raws = append(raws, raw)
	}

	// Act
	feed(t, pool, "node-a", raws...)

	// Assert
	listed, _, err := review.NewRepository(pool).HeadsByEntity(ctx, bakery,
	                                                          page, review.Hide{})
	if err != nil || len(listed) != 1 ||
	   listed[0].Endorsements != 2 || listed[0].Disputes != 1 {
		t.Fatalf("listed = %+v, %v, want 2 endorsements and 1 dispute",
		         listed, err)
	}

	// Act: an edit, which no vote targeted
	edited, _ := sign(t, "review.replaced", "2026-11-02T10:00:00Z",
		`{"replaces":"`+root+`","rating":5,"body":"melhorou"}`)
	feed(t, pool, "node-b", edited)

	// Assert: the head starts at zero; the old version keeps its votes
	listed, _, err = review.NewRepository(pool).HeadsByEntity(ctx, bakery,
	                                                         page, review.Hide{})
	if err != nil || listed[0].Endorsements != 0 || listed[0].Disputes != 0 {
		t.Fatalf("after the edit: %+v, %v, want no votes on the head",
		         listed, err)
	}

	if e, d := tallyOf(t, pool, root); e != 2 || d != 1 {
		t.Fatalf("old version = %d/%d, want 2/1", e, d)
	}
}
