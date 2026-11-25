package projector


import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/claim"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/page"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/response"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/review"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/fixturetest"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/projection"
	"github.com/jackc/pgx/v5/pgxpool"
)


// --- CODE ---

// responsesPath answers path through the responses route, as the API mounts it.
func responsesPath(t *testing.T,
                   pool *pgxpool.Pool,
                   path string) *httptest.ResponseRecorder {
	t.Helper()

	h := response.NewHandler(response.NewRepository(pool),
	                         review.NewRepository(pool),
	                         label.NewModeration(label.Policy{}, pool),
	                         claim.NewRecognition(claim.Policy{}, pool))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /reviews/{eventId}/responses", h.ListByReview)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

	return rec
}


// responseItem is one item of the responses route, decoded.
type responseItem struct {
	Envelope string `json:"envelope"`
	Derived  struct {
		ReplyTo          string `json:"replyTo"`
		RespondedVersion string `json:"respondedVersion"`
		TextChanged      bool   `json:"textChanged"`
		Depth            int    `json:"depth"`
	} `json:"derived"`
}


// responsesOf decodes the answer of the responses route.
func responsesOf(t *testing.T, rec *httptest.ResponseRecorder) ([]responseItem,
                                                                 *string) {
	t.Helper()

	var a struct {
		Items []responseItem `json:"items"`
		Next  *string        `json:"next"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}

	return a.Items, a.Next
}


// bodies returns each response's body, in order.
func bodies(items []responseItem) []string {
	var got []string

	for _, it := range items {
		var e struct {
			Payload struct{ Body string } `json:"payload"`
		}

		json.Unmarshal([]byte(it.Envelope), &e)
		got = append(got, e.Payload.Body)
	}

	return got
}


func TestTheResponseFixtureThroughTheProjector(t *testing.T) {
	file := fixturetest.LoadFrom(t, "../../../protocol/fixtures",
	                             "vote-and-label-binding")

	fixturetest.Run(t, file, func(t *testing.T, c fixturetest.Case) {
		if _, ok := c.Expect["responsesOnHead"]; !ok {
			t.Skip("not about responses")
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

		// Assert: what was stored is what the rule attaches to the chain
		stored, _, err := response.NewRepository(pool).ByChain(
			context.Background(), root, page.Page{Limit: 100}, review.Hide{})
		if err != nil {
			t.Fatalf("ByChain: %v", err)
		}

		var got, rule []string

		for _, r := range stored {
			got = append(got, r.EventID)
		}

		chain := projection.Chain(root, events)
		for _, r := range projection.ResponsesOnChain(chain, events) {
			rule = append(rule, r.ID)
		}

		slices.Sort(got)
		slices.Sort(rule)

		if !slices.Equal(got, rule) {
			t.Fatalf("responses = %v, the rule says %v", got, rule)
		}

		// and what the fixture expects of it
		if name, ok := c.Expect["respondedTo"].(string); ok {
			if stored[0].RespondedVersion != ids[name] {
				t.Errorf("responded to %s, want %s", stored[0].RespondedVersion,
				         name)
			}
		}

		if changed, ok := c.Expect["textChangedSinceResponse"].(bool); ok {
			if (stored[0].RespondedVersion != stored[0].Head) != changed {
				t.Errorf("text changed = %v, want %v",
				         stored[0].RespondedVersion != stored[0].Head, changed)
			}
		}
	})
}


func TestAConversationKeepsItsPlaceAndIsShownAtMostTwoDeep(t *testing.T) {
	// Arrange: a review, the company's answer, the author's reply, and a
	// reply to that, then the review edited
	pool := migrated(t)

	created, root := sign(t, "review.created", "2026-11-01T10:00:00Z",
		`{"entity":"`+bakery+`","rating":2,"body":"cafe frio"}`)
	answer, answerID := signAs(t, 'c', "review.response_created",
		"2026-11-01T11:00:00Z", `{"target":"`+root+`","body":"vamos ver"}`)
	reply, replyID := sign(t, "review.response_created", "2026-11-01T12:00:00Z",
		`{"target":"`+answerID+`","body":"obrigado"}`)
	again, _ := signAs(t, 'c', "review.response_created",
		"2026-11-01T13:00:00Z", `{"target":"`+replyID+`","body":"de nada"}`)
	edited, edit := sign(t, "review.replaced", "2026-11-02T10:00:00Z",
		`{"replaces":"`+root+`","rating":4,"body":"resolvido"}`)

	feed(t, pool, "node-a", created, answer, reply, again, edited)

	// Act: asked through the edit, not the original
	rec := responsesPath(t, pool, "/reviews/"+edit+"/responses")

	// Assert
	items, next := responsesOf(t, rec)

	if rec.Code != http.StatusOK ||
	   !slices.Equal(bodies(items), []string{"vamos ver", "obrigado", "de nada"}) ||
	   next != nil {
		t.Fatalf("GET = %d, %v, next %v", rec.Code, bodies(items), next)
	}

	depths := []int{items[0].Derived.Depth, items[1].Derived.Depth,
	                items[2].Derived.Depth}

	if !slices.Equal(depths, []int{1, 2, 2}) {
		t.Fatalf("depths = %v, want 1, 2, 2 (the third is 3 deep, shown at 2)",
		         depths)
	}

	if items[0].Derived.ReplyTo != root || items[2].Derived.ReplyTo != replyID {
		t.Fatalf("replyTo = %s, %s", items[0].Derived.ReplyTo,
		         items[2].Derived.ReplyTo)
	}

	for i, it := range items {
		if it.Derived.RespondedVersion != root || !it.Derived.TextChanged {
			t.Fatalf("item %d = %+v, want responding to the original, "+
			         "since changed", i, it.Derived)
		}
	}

	var stored int

	pool.QueryRow(context.Background(),
		"SELECT depth FROM responses WHERE body = 'de nada'").Scan(&stored)

	if stored != 3 {
		t.Fatalf("stored depth = %d, want the real 3", stored)
	}
}


func TestResponsesPageThroughEveryOneInOrder(t *testing.T) {
	// Arrange: five responses
	pool := migrated(t)

	created, root := sign(t, "review.created", "2026-11-01T10:00:00Z",
		`{"entity":"`+bakery+`","rating":3,"body":"ok"}`)

	raws := [][]byte{created}

	for i := 1; i <= 5; i++ {
		raw := signedBy(t, byte('a'+i), "review.response_created",
			fmt.Sprintf("2026-11-01T1%d:00:00Z", i),
			fmt.Sprintf(`{"target":"%s","body":"r%d"}`, root, i))
		raws = append(raws, raw)
	}

	feed(t, pool, "node-a", raws...)

	// Act
	var got []string
	path := "/reviews/" + root + "/responses?limit=2"

	for range 3 {
		items, next := responsesOf(t, responsesPath(t, pool, path))
		got = append(got, bodies(items)...)

		if next == nil {
			break
		}

		path = "/reviews/" + root + "/responses?limit=2&after=" + *next
	}

	// Assert
	if !slices.Equal(got, []string{"r1", "r2", "r3", "r4", "r5"}) {
		t.Fatalf("pages = %v, want r1 to r5, each once", got)
	}
}


func TestResponsesAnswerGoneNotFoundAndBadRequest(t *testing.T) {
	// Arrange: a withdrawn review that had an answer
	pool := migrated(t)

	created, root := sign(t, "review.created", "2026-11-01T10:00:00Z",
		`{"entity":"`+bakery+`","rating":3,"body":"ok"}`)
	answer := signedBy(t, 'c', "review.response_created",
		"2026-11-01T11:00:00Z", `{"target":"`+root+`","body":"oi"}`)
	withdrawal, _ := sign(t, "review.withdrawn", "2026-11-01T12:00:00Z",
		`{"target":"`+root+`","reason":"privacy"}`)

	feed(t, pool, "node-a", created, answer, withdrawal)

	cases := map[string]int{
		"/reviews/" + root + "/responses": http.StatusGone,
		"/reviews/evt_" + strings.Repeat("q", 52) + "/responses": http.StatusNotFound,
		"/reviews/evt_short/responses":                            http.StatusBadRequest,
		"/reviews/" + root + "/responses?limit=0":                 http.StatusBadRequest,
	}

	for path, want := range cases {
		// Act
		rec := responsesPath(t, pool, path)

		// Assert
		if rec.Code != want || bytes.Contains(rec.Body.Bytes(), []byte(`"oi"`)) {
			t.Errorf("GET %s = %d %q, want %d", path, rec.Code,
			         rec.Body.String(), want)
		}
	}
}


func TestAResponseToSomethingThatIsNoReviewIsKeptAndIgnored(t *testing.T) {
	// Arrange: an answer to a profile
	pool := migrated(t)

	profile, profileID := sign(t, "profile.updated", "2026-11-01T10:00:00Z",
		`{"displayName":"alice"}`)
	answer := signedBy(t, 'c', "review.response_created",
		"2026-11-01T11:00:00Z", `{"target":"`+profileID+`","body":"oi"}`)

	// Act
	feed(t, pool, "node-a", profile, answer)

	// Assert
	if count(t, pool, "events") != 2 || count(t, pool, "responses") != 0 {
		t.Fatalf("events = %d, responses = %d, want 2 and 0",
		         count(t, pool, "events"), count(t, pool, "responses"))
	}
}


func TestAReplyNeverComesBeforeWhatItAnswers(t *testing.T) {
	// Arrange: an answer, a reply in the same second, and a reply whose
	// author's clock put it before the answer
	pool := migrated(t)

	created, root := sign(t, "review.created", "2026-11-01T10:00:00Z",
		`{"entity":"`+bakery+`","rating":3,"body":"ok"}`)
	answer, answerID := signAs(t, 'c', "review.response_created",
		"2026-11-01T11:00:00Z", `{"target":"`+root+`","body":"answer"}`)
	same, _ := sign(t, "review.response_created", "2026-11-01T11:00:00Z",
		`{"target":"`+answerID+`","body":"same second"}`)
	early, _ := signAs(t, 'd', "review.response_created",
		"2026-11-01T10:30:00Z", `{"target":"`+answerID+`","body":"clock behind"}`)

	feed(t, pool, "node-a", created, answer, same, early)

	// Act
	items, _ := responsesOf(t, responsesPath(t, pool, "/reviews/"+root+"/responses"))

	// Assert: the answer first, whatever the replies' ids and clocks say
	got := bodies(items)

	if len(got) != 3 || got[0] != "answer" {
		t.Fatalf("order = %v, want the answer first", got)
	}
}
