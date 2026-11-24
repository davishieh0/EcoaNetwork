package projector


import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/claim"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/review"
	"github.com/jackc/pgx/v5/pgxpool"
)


// --- CODE ---

// listedBy reads an entity's reviews from an API ranking by ranking, every
// page of `limit`, and returns each item's eventId, in order, and the ranking
// the answer named.
func listedBy(t *testing.T,
              pool *pgxpool.Pool,
              ranking review.Ranking,
              entity string,
              limit string) ([]string, string) {
	t.Helper()

	h := review.NewHandler(review.NewRepository(pool),
	                       label.NewModeration(label.Policy{}, pool),
	                       claim.NewRecognition(claim.Policy{}, pool),
	                       ranking)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /entities/{entity}/reviews", h.ListByEntity)

	var ids []string
	var named string
	after := ""

	for {
		path := "/entities/" + entity + "/reviews?limit=" + limit
		if after != "" {
			path += "&after=" + url.QueryEscape(after)
		}

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

		var body struct {
			Ranking string `json:"ranking"`
			Items   []struct {
				Envelope string `json:"envelope"`
			} `json:"items"`
			Next *string `json:"next"`
		}

		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode %d %q: %v", rec.Code, rec.Body.String(), err)
		}

		named = body.Ranking
		for _, item := range body.Items {
			var e struct {
				EventID string `json:"eventId"`
			}
			_ = json.Unmarshal([]byte(item.Envelope), &e)
			ids = append(ids, e.EventID)
		}

		if body.Next == nil {
			return ids, named
		}

		after = *body.Next
	}
}


func TestTwoRankingsOrderTheSameReviewsDifferently(t *testing.T) {
	// Arrange: three reviews, the oldest the most endorsed
	pool := migrated(t)
	const entity = "cnpj:12345678000199"

	written := func(seed byte, at string) ([]byte, string) {
		return signAs(t, seed, "review.created", at,
		              `{"entity":"`+entity+`","rating":4,"body":"ok"}`)
	}
	vote := func(seed byte, kind, target string) []byte {
		raw, _ := signAs(t, seed, "review.endorsed", "2026-11-12T10:00:00Z",
		                 `{"target":"`+target+`"}`)
		if kind == "disputed" {
			raw, _ = signAs(t, seed, "review.disputed", "2026-11-12T10:00:00Z",
			                `{"target":"`+target+`"}`)
		}
		return raw
	}

	old, oldID := written('a', "2026-11-01T10:00:00Z")
	mid, midID := written('b', "2026-11-05T10:00:00Z")
	recent, recentID := written('c', "2026-11-10T10:00:00Z")

	feed(t, pool, "n1", old, mid, recent,
	     vote('v', "endorsed", oldID), vote('w', "endorsed", oldID),
	     vote('x', "endorsed", midID),
	     vote('y', "disputed", recentID))

	// Act: one page at a time, so the cursor carries the score
	newest, namedNewest := listedBy(t, pool, review.Newest, entity, "1")
	endorsed, namedEndorsed := listedBy(t, pool, review.Endorsed, entity, "1")

	// Assert
	if namedNewest != "newest" || namedEndorsed != "endorsed" {
		t.Fatalf("rankings named %q and %q", namedNewest, namedEndorsed)
	}

	wantNewest := []string{recentID, midID, oldID}
	wantEndorsed := []string{oldID, midID, recentID}

	for i := range 3 {
		if len(newest) != 3 || newest[i] != wantNewest[i] {
			t.Fatalf("newest = %v, want %v", newest, wantNewest)
		}
		if len(endorsed) != 3 || endorsed[i] != wantEndorsed[i] {
			t.Fatalf("endorsed = %v, want %v", endorsed, wantEndorsed)
		}
	}
}
