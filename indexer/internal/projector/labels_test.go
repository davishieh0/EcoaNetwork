package projector


import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/event"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/response"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/review"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/fixturetest"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/projection"
	"github.com/jackc/pgx/v5/pgxpool"
)


// --- CODE ---

// didOf returns the did:key of the key of seed.
func didOf(seed byte) string {
	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, 32))

	return identity.FromKey(key.Public().(ed25519.PublicKey)).String()
}


// api answers path through every read route, following policy.
func api(t *testing.T,
         pool *pgxpool.Pool,
         policy label.Policy,
         path string) *httptest.ResponseRecorder {
	t.Helper()

	moderation := label.NewModeration(policy, pool)
	reviews := review.NewHandler(review.NewRepository(pool), moderation,
	                             review.Newest)
	responses := response.NewHandler(response.NewRepository(pool),
	                                 review.NewRepository(pool), moderation)
	events := event.NewHandler(event.NewRepository(pool), moderation)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /entities/{entity}/reviews", reviews.ListByEntity)
	mux.HandleFunc("GET /reviews/{eventId}", reviews.Get)
	mux.HandleFunc("GET /reviews/{eventId}/responses", responses.ListByReview)
	mux.HandleFunc("GET /events/{eventId}", events.Get)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

	return rec
}


// listed decodes a list answer into each item's body and its labels.
func listed(t *testing.T, rec *httptest.ResponseRecorder) ([]string, [][]string) {
	t.Helper()

	var a struct {
		Items []struct {
			Envelope string `json:"envelope"`
			Observed struct {
				Labels []label.Applied `json:"labels"`
			} `json:"observed"`
		} `json:"items"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
		t.Fatalf("decode %d %q: %v", rec.Code, rec.Body.String(), err)
	}

	var bodies []string
	var labels [][]string

	for _, it := range a.Items {
		var e struct {
			Payload struct{ Body string } `json:"payload"`
		}

		json.Unmarshal([]byte(it.Envelope), &e)
		bodies = append(bodies, e.Payload.Body)

		var names []string
		for _, l := range it.Observed.Labels {
			names = append(names, l.Label)
		}

		labels = append(labels, names)
	}

	return bodies, labels
}


// reviewOf publishes a review by the key of seed, written on day, with body.
func reviewOf(t *testing.T, seed byte, day, body string) ([]byte, string) {
	t.Helper()

	return signAs(t, seed, "review.created", "2026-11-"+day+"T10:00:00Z",
		`{"entity":"`+bakery+`","rating":3,"body":"`+body+`"}`)
}


// labelOf is a label by the key of seed on target.
func labelOf(t *testing.T, seed byte, target, name string) ([]byte, string) {
	t.Helper()

	return signAs(t, seed, "moderation.label_created", "2026-11-20T10:00:00Z",
		`{"target":"`+target+`","label":"`+name+`"}`)
}


func TestTheLabelFixturesThroughTheProjector(t *testing.T) {
	file := fixturetest.LoadFrom(t, "../../../protocol/fixtures",
	                             "vote-and-label-binding")

	fixturetest.Run(t, file, func(t *testing.T, c fixturetest.Case) {
		if _, ok := c.Expect["labelsOnHead"]; !ok {
			t.Skip("not about labels")
		}

		// Arrange
		pool := migrated(t)
		built := c.Events(t)

		ids := map[string]string{}
		var events []projection.Event
		var raws [][]byte
		var root string

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

		for _, b := range targetsFirst(built) {
			raws = append(raws, b.Raw)
		}

		// Act
		feed(t, pool, "node-a", raws...)

		// Assert: the labels that hold on the chain are the rule's
		rows, err := pool.Query(context.Background(),
			`SELECT event_id FROM labels
			 WHERE chain_root = $1 AND NOT revoked ORDER BY event_id`, root)
		if err != nil {
			t.Fatalf("read labels: %v", err)
		}

		var got []string
		for rows.Next() {
			var id string
			rows.Scan(&id)
			got = append(got, id)
		}

		var rule []string
		chain := projection.Chain(root, events)
		for _, l := range projection.LabelsOnChain(chain, events) {
			rule = append(rule, l.ID)
		}

		var want []string
		for _, l := range c.Expect["labelsOnHead"].([]any) {
			want = append(want, ids[l.(map[string]any)["id"].(string)])
		}

		slices.Sort(rule)
		slices.Sort(want)

		if !slices.Equal(got, rule) || !slices.Equal(got, want) {
			t.Fatalf("labels = %v, the rule says %v, the fixture %v",
			         got, rule, want)
		}
	})
}


func TestHiddenLabelsByAFollowedModeratorLeaveTheListing(t *testing.T) {
	// Arrange: one review labeled spam, one off-topic, one clean
	pool := migrated(t)

	spam, spamID := reviewOf(t, 'b', "01", "spam")
	off, offID := reviewOf(t, 'c', "02", "off")
	clean, _ := reviewOf(t, 'd', "03", "clean")
	l1, _ := labelOf(t, 'm', spamID, "spam")
	l2, _ := labelOf(t, 'm', offID, "off-topic")

	feed(t, pool, "node-a", spam, off, clean, l1, l2)

	following := label.Follow(didOf('m'))
	path := "/entities/" + bakery + "/reviews"

	cases := []struct {
		name   string
		policy label.Policy
		path   string
		bodies []string
		labels [][]string
	}{
		{"following", following, path,
		 []string{"clean", "off"}, [][]string{nil, {"off-topic"}}},
		{"following, asked to show", following, path + "?labeled=show",
		 []string{"clean", "off", "spam"},
		 [][]string{nil, {"off-topic"}, {"spam"}}},
		{"following nobody", label.Policy{}, path,
		 []string{"clean", "off", "spam"}, [][]string{nil, nil, nil}},
		{"following someone else", label.Follow(didOf('z')), path,
		 []string{"clean", "off", "spam"}, [][]string{nil, nil, nil}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Act
			bodies, labels := listed(t, api(t, pool, c.policy, c.path))

			// Assert
			if !slices.Equal(bodies, c.bodies) ||
			   !slices.EqualFunc(labels, c.labels, slices.Equal[[]string]) {
				t.Fatalf("listing = %v %v, want %v %v",
				         bodies, labels, c.bodies, c.labels)
			}
		})
	}
}


func TestARemovedReviewLeavesEveryRoute(t *testing.T) {
	// Arrange: a review, its edit and an answer, then labeled illegal on
	// the original version
	pool := migrated(t)

	created, root := reviewOf(t, 'b', "01", "bad")
	edited, edit := signAs(t, 'b', "review.replaced", "2026-11-02T10:00:00Z",
		`{"replaces":"`+root+`","rating":3,"body":"still bad"}`)
	answer, _ := signAs(t, 'c', "review.response_created",
		"2026-11-03T10:00:00Z", `{"target":"`+root+`","body":"oi"}`)
	illegal, _ := labelOf(t, 'm', root, "illegal")

	feed(t, pool, "node-a", created, edited, answer, illegal)

	following := label.Follow(didOf('m'))

	// Act and Assert: out of the listing, even when asked to show
	bodies, _ := listed(t, api(t, pool, following,
		"/entities/"+bakery+"/reviews?labeled=show"))
	if len(bodies) != 0 {
		t.Fatalf("listing = %v, want nothing", bodies)
	}

	// and 451 with nothing of it, through any of its versions
	for _, path := range []string{
		"/reviews/" + root,
		"/reviews/" + edit,
		"/reviews/" + root + "/responses",
		"/events/" + root,
		"/events/" + edit,
	} {
		rec := api(t, pool, following, path)

		if rec.Code != http.StatusUnavailableForLegalReasons ||
		   strings.Contains(rec.Body.String(), "bad") {
			t.Errorf("GET %s = %d %q, want 451", path, rec.Code,
			         rec.Body.String())
		}
	}

	// a client of an indexer that does not follow the moderator still sees it
	if rec := api(t, pool, label.Policy{}, "/reviews/"+root); rec.Code != http.StatusOK {
		t.Fatalf("not following: GET = %d, want 200", rec.Code)
	}
}


func TestARevokedLabelStopsCounting(t *testing.T) {
	// Arrange
	pool := migrated(t)

	created, root := reviewOf(t, 'b', "01", "back")
	spam, spamID := labelOf(t, 'm', root, "spam")
	revoked, _ := signAs(t, 'm', "moderation.label_revoked",
		"2026-11-21T10:00:00Z", `{"target":"`+spamID+`"}`)
	byOther, _ := signAs(t, 'z', "moderation.label_revoked",
		"2026-11-21T10:00:00Z", `{"target":"`+spamID+`"}`)

	following := label.Follow(didOf('m'))
	path := "/entities/" + bakery + "/reviews"

	// Act: someone else tries to take the label back
	feed(t, pool, "node-a", created, spam, byOther)

	// Assert: it still counts
	if bodies, _ := listed(t, api(t, pool, following, path)); len(bodies) != 0 {
		t.Fatalf("after another's revoke: %v, want still hidden", bodies)
	}

	// Act: its moderator takes it back
	feed(t, pool, "node-b", created, spam, byOther, revoked)

	// Assert
	bodies, labels := listed(t, api(t, pool, following, path))
	if !slices.Equal(bodies, []string{"back"}) || labels[0] != nil {
		t.Fatalf("after the revoke: %v %v, want back, unlabeled",
		         bodies, labels)
	}
}


func TestALabelOnAResponseHidesOnlyThatResponse(t *testing.T) {
	// Arrange
	pool := migrated(t)

	created, root := reviewOf(t, 'b', "01", "ok")
	good, _ := signAs(t, 'c', "review.response_created",
		"2026-11-02T10:00:00Z", `{"target":"`+root+`","body":"good"}`)
	bad, badID := signAs(t, 'd', "review.response_created",
		"2026-11-03T10:00:00Z", `{"target":"`+root+`","body":"bad"}`)
	spam, _ := labelOf(t, 'm', badID, "spam")

	feed(t, pool, "node-a", created, good, bad, spam)

	following := label.Follow(didOf('m'))
	path := "/reviews/" + root + "/responses"

	// Act
	hidden, _ := listed(t, api(t, pool, following, path))
	shown, labels := listed(t, api(t, pool, following, path+"?labeled=show"))

	// Assert
	if !slices.Equal(hidden, []string{"good"}) {
		t.Fatalf("responses = %v, want only good", hidden)
	}

	if !slices.Equal(shown, []string{"good", "bad"}) ||
	   !slices.Equal(labels[1], []string{"spam"}) {
		t.Fatalf("shown = %v %v, want good, and bad marked spam", shown, labels)
	}

	// the review itself is not touched
	if rec := api(t, pool, following, "/reviews/"+root); rec.Code != http.StatusOK {
		t.Fatalf("GET review = %d, want 200", rec.Code)
	}
}


func TestTheListingPagesThroughWhatIsLeftWhenLabelsHideSome(t *testing.T) {
	// Arrange: five reviews, the second and the fourth labeled spam
	pool := migrated(t)

	var raws [][]byte

	for i, body := range []string{"r1", "r2", "r3", "r4", "r5"} {
		raw, id := reviewOf(t, byte('b'+i), "0"+string(rune('1'+i)), body)
		raws = append(raws, raw)

		if body == "r2" || body == "r4" {
			l, _ := labelOf(t, 'm', id, "spam")
			raws = append(raws, l)
		}
	}

	feed(t, pool, "node-a", raws...)

	following := label.Follow(didOf('m'))

	// Act: one per page
	var got []string
	path := "/entities/" + bakery + "/reviews?limit=1"

	for range 5 {
		rec := api(t, pool, following, path)
		bodies, _ := listed(t, rec)
		got = append(got, bodies...)

		var a struct{ Next *string }
		json.Unmarshal(rec.Body.Bytes(), &a)

		if a.Next == nil {
			break
		}

		path = "/entities/" + bakery + "/reviews?limit=1&after=" + *a.Next
	}

	// Assert
	if !slices.Equal(got, []string{"r5", "r3", "r1"}) {
		t.Fatalf("pages = %v, want r5, r3, r1", got)
	}
}


func TestLabeledMustBeShowOrAbsent(t *testing.T) {
	// Act
	rec := api(t, nil, label.Policy{},
		"/entities/"+bakery+"/reviews?labeled=all")

	// Assert
	if rec.Code != http.StatusBadRequest ||
	   !strings.Contains(rec.Body.String(), "invalid_labeled_error") {
		t.Fatalf("GET = %d %q", rec.Code, rec.Body.String())
	}
}


func TestTheClientChoosesTheModeratorsItFollows(t *testing.T) {
	// Arrange: m labels one review spam and removes another; the indexer's
	// own file follows m
	pool := migrated(t)

	spam, spamID := reviewOf(t, 'b', "01", "spam")
	gone, goneID := reviewOf(t, 'c', "02", "gone")
	clean, _ := reviewOf(t, 'd', "03", "clean")
	l1, _ := labelOf(t, 'm', spamID, "spam")
	l2, _ := labelOf(t, 'm', goneID, "illegal")

	feed(t, pool, "node-a", spam, gone, clean, l1, l2)

	file := label.Follow(didOf('m'))
	list := "/entities/" + bakery + "/reviews"

	cases := []struct {
		name   string
		query  string
		bodies []string
	}{
		{"the indexer's own list", "", []string{"clean"}},
		{"a client following m", "?moderators=" + didOf('m'),
		 []string{"clean"}},
		{"a client following nobody", "?moderators=",
		 []string{"clean", "gone", "spam"}},
		{"a client following someone else", "?moderators=" + didOf('z'),
		 []string{"clean", "gone", "spam"}},
		{"a client following m and someone else",
		 "?moderators=" + didOf('z') + "," + didOf('m'), []string{"clean"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Act
			bodies, _ := listed(t, api(t, pool, file, list+c.query))

			// Assert: the label is a projection, not a deletion
			if !slices.Equal(bodies, c.bodies) {
				t.Fatalf("listing = %v, want %v", bodies, c.bodies)
			}
		})
	}

	// the removed review answers by id only to a client not following m
	byID := "/reviews/" + goneID
	if rec := api(t, pool, file, byID+"?moderators="+didOf('m'));
		rec.Code != http.StatusUnavailableForLegalReasons {
		t.Fatalf("following m, GET %s = %d, want 451", byID, rec.Code)
	}

	if rec := api(t, pool, file, byID+"?moderators="); rec.Code != http.StatusOK {
		t.Fatalf("following nobody, GET %s = %d, want 200", byID, rec.Code)
	}

	// a list that is not did:key is refused
	if rec := api(t, pool, file, list+"?moderators=someone");
		rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed moderators = %d, want 400", rec.Code)
	}
}
