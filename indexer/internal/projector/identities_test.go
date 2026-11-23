package projector


import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/identities"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/fixturetest"
	"github.com/jackc/pgx/v5/pgxpool"
)


// --- CODE ---

// identityOf answers GET /identities/{did}.
func identityOf(t *testing.T, pool *pgxpool.Pool, did string) identities.Identity {
	t.Helper()

	h := identities.NewHandler(identities.NewRepository(pool))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /identities/{did}", h.Get)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/identities/"+did, nil))

	var id identities.Identity
	if err := json.Unmarshal(rec.Body.Bytes(), &id); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("GET = %d %q", rec.Code, rec.Body.String())
	}

	return id
}


// rotate is the key of seed naming the key of next as its successor.
func rotate(t *testing.T, seed, next byte, at string) []byte {
	t.Helper()

	raw, _ := signAs(t, seed, "identity.key_rotated", at,
		`{"successor":"`+didOf(next)+`"}`)

	return raw
}


func TestTheSuccessionFixturesThroughTheProjector(t *testing.T) {
	file := fixturetest.LoadFrom(t, "../../../protocol/fixtures",
	                             "identity-succession")

	fixturetest.Run(t, file, func(t *testing.T, c fixturetest.Case) {
		switch c.Name {
		case "transitive rotation chain",
		     "both rotations arrive in the same back-fill",
		     "a rotation forming a cycle is rejected":
		default:
			// what one node accepts or refuses, by what it observed first: an
			// indexer reading that node never receives what it refused
			t.Skip("a node's own observation")
		}

		// Arrange
		pool := migrated(t)

		var raws [][]byte
		for _, b := range c.Events(t) {
			raws = append(raws, b.Raw)
		}

		first := fixturetest.Author("alice_k1")

		// Act
		feed(t, pool, "node-a", raws...)

		// Assert
		id := identityOf(t, pool, first)

		if want, ok := c.Expect["contested"].(bool); ok && id.Contested != want {
			t.Errorf("contested = %v, want %v", id.Contested, want)
		}

		if want, ok := c.Expect["successor"].(string); ok {
			if id.Current != fixturetest.Author(want) {
				t.Errorf("current = %s, want %s", id.Current, want)
			}
		}

		if chain, ok := c.Expect["chain"].([]any); ok {
			var want []string
			for _, name := range chain {
				want = append(want, fixturetest.Author(name.(string)))
			}

			if !slices.Equal(id.Keys, want) {
				t.Errorf("keys = %v, want %v", id.Keys, want)
			}
		}
	})
}


func TestTwoNodesWithDifferentSuccessorsContestTheIdentity(t *testing.T) {
	// Arrange: each node kept a different first rotation of k1
	pool := migrated(t)

	// Act
	feed(t, pool, "node-a", rotate(t, '1', '2', "2026-11-01T10:00:00Z"))
	feed(t, pool, "node-b", rotate(t, '1', 'x', "2026-11-01T10:05:00Z"))

	// Assert: neither successor is honored
	id := identityOf(t, pool, didOf('1'))
	second := identityOf(t, pool, didOf('2'))

	if !id.Contested || !slices.Equal(id.Keys, []string{didOf('1')}) ||
	   second.Identity != didOf('2') {
		t.Fatalf("k1 = %+v, k2 = %+v", id, second)
	}
}


func TestVotesBeforeAndAfterARotationAreOneVoter(t *testing.T) {
	// Arrange: k1 endorses, k2 disputes later, and the rotation linking them
	// arrives only after both votes
	pool := migrated(t)

	created, root := reviewOf(t, 'b', "01", "ok")
	endorse := signedBy(t, '1', "review.endorsed", "2026-11-02T10:00:00Z",
		`{"target":"`+root+`"}`)
	dispute := signedBy(t, '2', "review.disputed", "2026-11-04T10:00:00Z",
		`{"target":"`+root+`"}`)

	feed(t, pool, "node-a", created, endorse, dispute)

	if e, d := tallyOf(t, pool, root); e != 1 || d != 1 {
		t.Fatalf("before the rotation = %d/%d, want 1/1: two keys", e, d)
	}

	// Act
	feed(t, pool, "node-b", rotate(t, '1', '2', "2026-11-03T10:00:00Z"))

	// Assert: one voter, the latest vote
	if e, d := tallyOf(t, pool, root); e != 0 || d != 1 {
		t.Fatalf("after the rotation = %d/%d, want 0/1: one voter", e, d)
	}
}


func TestAProfileFollowsItsIdentityAcrossKeys(t *testing.T) {
	// Arrange: k2's profile arrives before the rotation that makes k2 k1's
	pool := migrated(t)

	old, _ := profileOf(t, '1', "2026-11-01T10:00:00Z", `{"displayName":"old"}`)
	newer, _ := profileOf(t, '2', "2026-11-03T10:00:00Z", `{"displayName":"new"}`)

	feed(t, pool, "node-a", old, newer)

	// Act
	feed(t, pool, "node-b", rotate(t, '1', '2', "2026-11-02T10:00:00Z"))

	// Assert: both keys answer the identity's latest profile, and one row
	for _, key := range []byte{'1', '2'} {
		dids, names := displayNames(t, profilesAPI(t, pool, label.Policy{},
			"/profiles/"+didOf(key)))

		if !slices.Equal(names, []string{"new"}) || dids[0] != didOf('1') {
			t.Fatalf("/profiles/k%c = %v %v, want new, of k1", key, dids, names)
		}
	}

	if count(t, pool, "profiles") != 1 {
		t.Fatalf("profiles = %d, want one for the identity", count(t, pool, "profiles"))
	}
}


func TestAModeratorWhoRotatedIsStillFollowedAndRevokesTheirOwn(t *testing.T) {
	// Arrange: the moderator labels with k1, rotates, labels again with k2
	pool := migrated(t)

	a, aID := reviewOf(t, 'b', "01", "a")
	b, bID := reviewOf(t, 'c', "02", "b")
	first, firstID := labelOf(t, '1', aID, "spam")
	moved := rotate(t, '1', '2', "2026-11-20T11:00:00Z")
	second, _ := labelOf(t, '2', bID, "spam")

	feed(t, pool, "node-a", a, b, first, moved, second)

	following := label.Follow(didOf('1'))
	path := "/entities/" + bakery + "/reviews"

	// Assert: following k1 follows the label signed by k2 too
	if bodies, _ := listed(t, api(t, pool, following, path)); len(bodies) != 0 {
		t.Fatalf("listing = %v, want both hidden", bodies)
	}

	// Act: k2 takes back the label k1 published
	revoke, _ := signAs(t, '2', "moderation.label_revoked",
		"2026-11-21T10:00:00Z", `{"target":"`+firstID+`"}`)
	feed(t, pool, "node-b", revoke)

	// Assert
	if bodies, _ := listed(t, api(t, pool, following, path)); !slices.Equal(bodies, []string{"a"}) {
		t.Fatalf("listing = %v, want a back", bodies)
	}
}


func TestAContestedIdentityKeepsItsKeysApart(t *testing.T) {
	// Arrange: k1 rotated to two keys; k1 and k2 both vote and both profile
	pool := migrated(t)

	created, root := reviewOf(t, 'b', "01", "ok")
	v1 := signedBy(t, '1', "review.endorsed", "2026-11-02T10:00:00Z",
		`{"target":"`+root+`"}`)
	v2 := signedBy(t, '2', "review.endorsed", "2026-11-04T10:00:00Z",
		`{"target":"`+root+`"}`)
	p1, _ := profileOf(t, '1', "2026-11-01T10:00:00Z", `{"displayName":"alice"}`)
	p2, _ := profileOf(t, '2', "2026-11-05T10:00:00Z", `{"displayName":"thief"}`)

	feed(t, pool, "node-a", created, v1, v2, p1, p2,
		rotate(t, '1', '2', "2026-11-03T10:00:00Z"))
	feed(t, pool, "node-b", rotate(t, '1', 'x', "2026-11-03T10:01:00Z"))

	// Assert: two voters, and k2's profile is not k1's
	if e, _ := tallyOf(t, pool, root); e != 2 {
		t.Fatalf("endorsements = %d, want 2: k2 is not k1 any more", e)
	}

	_, names := displayNames(t, profilesAPI(t, pool, label.Policy{},
		"/profiles/"+didOf('1')))

	if !slices.Equal(names, []string{"alice"}) {
		t.Fatalf("/profiles/k1 = %v, want alice", names)
	}
}


func TestTheIdentityRouteShowsKeysAndRevocations(t *testing.T) {
	// Arrange
	pool := migrated(t)

	revoke, revokeID := signAs(t, '2', "identity.key_revoked",
		"2026-11-05T10:00:00Z", `{"reason":"compromised"}`)

	feed(t, pool, "node-a",
		rotate(t, '1', '2', "2026-11-01T10:00:00Z"),
		rotate(t, '2', '3', "2026-11-02T10:00:00Z"),
		revoke)

	// Act
	id := identityOf(t, pool, didOf('2'))
	alone := identityOf(t, pool, didOf('z'))

	// Assert
	if id.Identity != didOf('1') ||
	   !slices.Equal(id.Keys, []string{didOf('1'), didOf('2'), didOf('3')}) ||
	   id.Current != didOf('3') || id.Contested ||
	   len(id.Revoked) != 1 || id.Revoked[0].EventID != revokeID {
		t.Fatalf("identity = %+v", id)
	}

	if alone.Identity != didOf('z') || !slices.Equal(alone.Keys, []string{didOf('z')}) {
		t.Fatalf("a key that never rotated = %+v", alone)
	}

	if count(t, pool, "key_revocations") != 1 {
		t.Fatal("the revocation was not stored")
	}
}
