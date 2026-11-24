package projector


import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label"
)


// --- CODE ---

func TestTheSummaryCountsTheHeadsThatCount(t *testing.T) {
	// Arrange: five reviews of one domain, and what happens to them
	pool := migrated(t)
	const domain = "resumo.test"
	entityID := "domain:" + domain

	review := func(seed byte, at string, rating string) ([]byte, string) {
		return signAs(t, seed, "review.created", at,
		              `{"entity":"`+entityID+`","rating":`+rating+
		              `,"body":"ok"}`)
	}

	five, _ := review('a', "2026-11-10T10:00:00Z", "5")
	edited, editedID := review('b', "2026-11-10T11:00:00Z", "1")
	two, _ := review('c', "2026-11-10T12:00:00Z", "2")
	gone, goneID := review('f', "2026-11-10T13:00:00Z", "5")
	spam, spamID := review('g', "2026-11-10T14:00:00Z", "1")
	own, _ := review('d', "2026-11-10T15:00:00Z", "5")

	// b edits 1 into 4; f withdraws; m (followed) labels g's spam; d holds
	// the domain
	edit, _ := signAs(t, 'b', "review.replaced", "2026-11-11T10:00:00Z",
	                  `{"replaces":"`+editedID+`","rating":4,"body":"melhorou"}`)
	withdraw, _ := signAs(t, 'f', "review.withdrawn", "2026-11-11T10:00:00Z",
	                      `{"target":"`+goneID+`"}`)
	labeled, _ := signAs(t, 'm', "moderation.label_created",
	                     "2026-11-11T10:00:00Z",
	                     `{"target":"`+spamID+`","label":"spam"}`)
	claimed, _ := domainClaim(t, 'd', domain)

	feed(t, pool, "n1", five, edited, two, gone, spam, own,
	     edit, withdraw, labeled, claimed)
	check(t, pool, records{"_ecoa-verify." + domain: {
		"ecoa-identity=" + didOf('d')}})

	// Act
	rec := entitiesAPI(t, pool, label.Follow(didOf('m')), "/entities/"+entityID)

	// Assert: a's 5, b's edit to 4, c's 2; f withdrew, g is hidden, d's is
	// a conflict of interest
	var body struct {
		Observed struct {
			Summary struct {
				Count        int     `json:"count"`
				Mean         float64 `json:"mean"`
				Distribution [5]int  `json:"distribution"`
				Conflicted   int     `json:"conflicted"`
			} `json:"summary"`
		} `json:"observed"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil ||
		rec.Code != http.StatusOK {
		t.Fatalf("GET = %d %q", rec.Code, rec.Body.String())
	}

	s := body.Observed.Summary
	if s.Count != 3 || s.Mean != 3.67 ||
		s.Distribution != [5]int{0, 1, 0, 1, 1} || s.Conflicted != 1 {
		t.Fatalf("summary = %+v, want 3 reviews, mean 3.67, [0 1 0 1 1], "+
		         "1 conflicted", s)
	}
}


func TestSearchFiltersByTheCategoryThatHolds(t *testing.T) {
	// Arrange: one bakery, one bank, and a shop whose first category holds
	// although a later proposal says food
	pool := migrated(t)
	bakery, _ := signAs(t, 'a', "entity.updated", "2026-11-10T10:00:00Z",
	                    `{"entity":"domain:padaria.test","name":"Padaria",`+
	                    `"category":"food-and-drink"}`)
	bank, _ := signAs(t, 'a', "entity.updated", "2026-11-10T11:00:00Z",
	                  `{"entity":"domain:banco.test","name":"Banco",`+
	                  `"category":"financial-services"}`)
	shop, _ := signAs(t, 'a', "entity.updated", "2026-11-10T12:00:00Z",
	                  `{"entity":"domain:loja.test","name":"Loja",`+
	                  `"category":"retail"}`)
	challenger, _ := signAs(t, 'b', "entity.updated", "2026-11-11T12:00:00Z",
	                        `{"entity":"domain:loja.test",`+
	                        `"category":"food-and-drink"}`)
	feed(t, pool, "n1", bakery, bank, shop, challenger)

	// Act
	food := entitiesAPI(t, pool, label.Policy{},
	                    "/entities?category=food-and-drink")
	bad := entitiesAPI(t, pool, label.Policy{}, "/entities?category=bakeries")

	// Assert
	var body struct {
		Items []struct {
			Entity string `json:"entity"`
		} `json:"items"`
	}

	if err := json.Unmarshal(food.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", food.Body.String(), err)
	}

	if len(body.Items) != 1 || body.Items[0].Entity != "domain:padaria.test" {
		t.Fatalf("food = %+v, want the bakery only", body.Items)
	}

	if bad.Code != http.StatusBadRequest {
		t.Fatalf("an unknown category = %d, want 400", bad.Code)
	}
}
