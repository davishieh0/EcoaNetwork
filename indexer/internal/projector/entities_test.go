package projector


import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/claim"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/entity"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/fixturetest"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/projection"
	"github.com/jackc/pgx/v5/pgxpool"
)


// --- CODE ---

// entitiesAPI answers path through the entity routes, following policy.
func entitiesAPI(t *testing.T,
                 pool *pgxpool.Pool,
                 policy label.Policy,
                 path string) *httptest.ResponseRecorder {
	t.Helper()

	h := entity.NewHandler(entity.NewRepository(pool),
	                       label.NewModeration(policy, pool),
	                       claim.NewRecognition(claim.Policy{}, pool))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /entities", h.Search)
	mux.HandleFunc("GET /entities/{entity}", h.Get)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

	return rec
}


// entityOf decodes one entity answer.
func entityOf(t *testing.T, rec *httptest.ResponseRecorder) entity.Entity {
	t.Helper()

	var e entity.Entity
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("decode %d %q: %v", rec.Code, rec.Body.String(), err)
	}

	return e
}


// valueOf is a field's value, "" when no proposal sets it.
func valueOf(f *entity.Field) string {
	if f == nil {
		return ""
	}

	return f.Value
}


// propose is an entity.updated by the key of seed, written on day.
func propose(t *testing.T, seed byte, day, payload string) ([]byte, string) {
	t.Helper()

	return signAs(t, seed, "entity.updated", "2026-11-"+day+"T10:00:00Z", payload)
}


// padEntity is a fixture's entity name as fixturetest writes it in payloads.
func padEntity(name string) string {
	digits, ok := strings.CutPrefix(name, "cnpj:")
	if !ok {
		return name
	}

	return "cnpj:" + strings.Repeat("0", 14-len(digits)) + digits
}


func TestTheEntityMetadataFixturesThroughTheProjector(t *testing.T) {
	file := fixturetest.LoadFrom(t, "../../../protocol/fixtures",
	                             "entity-metadata")

	fixturetest.Run(t, file, func(t *testing.T, c fixturetest.Case) {
		// Arrange
		pool := migrated(t)
		built := c.Events(t)

		var raws [][]byte
		var subject string

		for _, b := range targetsFirst(built) {
			raws = append(raws, b.Raw)

			if b.Env.EventType == "entity.updated" && subject == "" {
				var p struct{ Entity string }
				json.Unmarshal(b.Env.Payload, &p)
				subject = p.Entity
			}
		}

		// the weights and claims the fixture gives as input, by real author
		identities := map[string]projection.Identity{}

		for name, raw := range c.Identities {
			var id projection.Identity
			if err := json.Unmarshal(raw, &id); err != nil {
				t.Fatalf("identity %s: %v", name, err)
			}

			for i := range id.Claims {
				id.Claims[i].Entity = padEntity(id.Claims[i].Entity)
			}

			identities[fixturetest.Author(name)] = id
		}

		// Act
		feed(t, pool, "node-a", raws...)

		// Assert: the proposals stored, minus the labeled ones, resolve under
		// the fixture's weights to the fixture's values
		rows, err := pool.Query(context.Background(),
			`SELECT e.envelope FROM entity_proposals p
			 JOIN events e ON e.event_id = p.event_id
			 WHERE p.entity = $1 AND NOT EXISTS (
			     SELECT 1 FROM labels l WHERE l.proposal = p.event_id
			     AND NOT l.revoked)`, subject)
		if err != nil {
			t.Fatalf("read proposals: %v", err)
		}

		var proposals []projection.Event

		for rows.Next() {
			var raw []byte
			rows.Scan(&raw)

			env, err := envelope.Decode(raw)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}

			proposals = append(proposals, projection.Event{
				ID: env.EventID, Author: env.Author, Type: env.EventType,
				CreatedAt: env.CreatedAt, Payload: env.Payload,
			})
		}

		fields := projection.ResolveFields(proposals, identities)
		canonical, _ := c.Expect["canonical"].(map[string]any)

		if len(fields) != len(canonical) {
			t.Errorf("fields = %+v, want %v", fields, canonical)
		}

		for name, want := range canonical {
			if fields[name].Value != want {
				t.Errorf("%s = %q, want %q", name, fields[name].Value, want)
			}
		}

		if tier, ok := c.Expect["tier"].(float64); ok {
			for name := range canonical {
				if float64(fields[name].Tier) != tier {
					t.Errorf("%s tier = %d, want %v", name, fields[name].Tier,
					         tier)
				}
			}
		}
	})
}


func TestAnEntityShowsTheFirstProposalOfEachField(t *testing.T) {
	// Arrange: a review names the entity; then a name and a category, a
	// later rival name, and a later category fix that leaves the name alone
	pool := migrated(t)

	created, _ := reviewOf(t, 'b', "01", "ok")
	first, firstID := propose(t, 'c', "02",
		`{"entity":"`+bakery+`","name":"Padaria do João","category":"retail"}`)
	rival, _ := propose(t, 'd', "03", `{"entity":"`+bakery+`","name":"Outro Nome"}`)
	fix, _ := propose(t, 'e', "04",
		`{"entity":"`+bakery+`","category":"food-and-drink"}`)

	feed(t, pool, "node-a", created, first, rival, fix)

	// Act
	rec := entitiesAPI(t, pool, label.Policy{}, "/entities/"+bakery)

	// Assert: v1 has no weights, so the first value of each field holds
	e := entityOf(t, rec)

	if rec.Code != http.StatusOK || valueOf(e.Observed.Name) != "Padaria do João" ||
	   e.Observed.Name.Proposal != firstID ||
	   valueOf(e.Observed.Category) != "retail" ||
	   e.Observed.Description != nil || e.Derived.Proposals != 3 {
		t.Fatalf("GET = %d %q", rec.Code, rec.Body.String())
	}
}


func TestALabeledProposalLeavesTheContest(t *testing.T) {
	// Arrange: a vandal's first name, labeled spam, and a later good one
	pool := migrated(t)

	vandal, vandalID := propose(t, 'v', "01",
		`{"entity":"`+bakery+`","name":"COMPRE AQUI"}`)
	good, _ := propose(t, 'c', "02", `{"entity":"`+bakery+`","name":"Padaria"}`)
	spam, _ := labelOf(t, 'm', vandalID, "spam")

	feed(t, pool, "node-a", vandal, good, spam)

	// Act
	following := entityOf(t, entitiesAPI(t, pool, label.Follow(didOf('m')),
		"/entities/"+bakery))
	notFollowing := entityOf(t, entitiesAPI(t, pool, label.Policy{},
		"/entities/"+bakery))

	// Assert
	if valueOf(following.Observed.Name) != "Padaria" ||
	   valueOf(notFollowing.Observed.Name) != "COMPRE AQUI" {
		t.Fatalf("following = %q, not following = %q",
		         valueOf(following.Observed.Name),
		         valueOf(notFollowing.Observed.Name))
	}
}


func TestAnEntityIsUnknownOnlyWhenNothingNamedIt(t *testing.T) {
	// Arrange: one entity named by a review alone
	pool := migrated(t)
	created, _ := reviewOf(t, 'b', "01", "ok")
	feed(t, pool, "node-a", created)

	// Act
	reviewed := entitiesAPI(t, pool, label.Policy{}, "/entities/"+bakery)
	unknown := entitiesAPI(t, pool, label.Policy{}, "/entities/gtin:07891000100103")
	malformed := entitiesAPI(t, pool, label.Policy{}, "/entities/twitter:x")

	// Assert
	e := entityOf(t, reviewed)

	if reviewed.Code != http.StatusOK || e.Observed.Name != nil ||
	   e.Derived.Proposals != 0 {
		t.Fatalf("reviewed = %d %q", reviewed.Code, reviewed.Body.String())
	}

	if unknown.Code != http.StatusNotFound || malformed.Code != http.StatusBadRequest {
		t.Fatalf("unknown = %d, malformed = %d", unknown.Code, malformed.Code)
	}
}


func TestAnEntityIdentifierIsCheckedAgainstItsDescriptor(t *testing.T) {
	// Arrange
	pool := migrated(t)

	descriptor := projection.Descriptor{Name: "Feira da Praça", Kind: "place",
	                                    Country: "BR", Locality: "Recife"}

	honest, err := projection.DeriveEntityID(descriptor)
	if err != nil {
		t.Fatalf("DeriveEntityID: %v", err)
	}

	forged := "entity:" + strings.Repeat("a", 52)
	d, _ := json.Marshal(descriptor)

	a, _ := propose(t, 'c', "01",
		`{"entity":"`+honest+`","name":"Feira","descriptor":`+string(d)+`}`)
	b, _ := propose(t, 'c', "02",
		`{"entity":"`+forged+`","name":"Feira","descriptor":`+string(d)+`}`)

	feed(t, pool, "node-a", a, b)

	// Act
	gotHonest := entityOf(t, entitiesAPI(t, pool, label.Policy{}, "/entities/"+honest))
	gotForged := entityOf(t, entitiesAPI(t, pool, label.Policy{}, "/entities/"+forged))

	// Assert
	if gotHonest.Derived.DescriptorMatches == nil || !*gotHonest.Derived.DescriptorMatches ||
	   gotForged.Derived.DescriptorMatches == nil || *gotForged.Derived.DescriptorMatches {
		t.Fatalf("honest = %v, forged = %v", gotHonest.Derived.DescriptorMatches,
		         gotForged.Derived.DescriptorMatches)
	}
}


func TestSearchFindsAnEntityByAnyPartOfAnyOfItsNames(t *testing.T) {
	// Arrange: a bakery that changed its name, a market, and a vandal's name
	// on the market, labeled
	pool := migrated(t)

	market := "cnpj:98765432000188"

	old, _ := propose(t, 'c', "01", `{"entity":"`+bakery+`","name":"Pão Quente"}`)
	renamed, _ := propose(t, 'c', "02",
		`{"entity":"`+bakery+`","name":"Padaria do João"}`)
	shop, _ := propose(t, 'd', "03", `{"entity":"`+market+`","name":"Mercado Sol"}`)
	vandal, vandalID := propose(t, 'v', "04",
		`{"entity":"`+market+`","name":"Padaria Falsa"}`)
	spam, _ := labelOf(t, 'm', vandalID, "spam")

	feed(t, pool, "node-a", old, renamed, shop, vandal, spam)

	following := label.Follow(didOf('m'))

	cases := map[string][]string{
		"joao":   {bakery},
		"PADARIA": {bakery},
		"pão":    {bakery},
		"quente": {bakery},
		"sol":    {market},
		"zzz":    nil,
		"":       {market, bakery},
		"50%":    nil,
	}

	for q, want := range cases {
		t.Run(q, func(t *testing.T) {
			// Act
			rec := entitiesAPI(t, pool, following, "/entities?q="+urlEncode(q))

			// Assert
			var a struct{ Items []entity.Entity }
			json.Unmarshal(rec.Body.Bytes(), &a)

			var got []string
			for _, e := range a.Items {
				got = append(got, e.Entity)
			}

			if rec.Code != http.StatusOK || !slices.Equal(got, want) {
				t.Fatalf("q=%q: %d %v, want %v", q, rec.Code, got, want)
			}
		})
	}

	// a result shows the name that holds, not the one that matched
	var a struct{ Items []entity.Entity }
	json.Unmarshal(entitiesAPI(t, pool, following, "/entities?q=quente").Body.Bytes(), &a)

	if valueOf(a.Items[0].Observed.Name) != "Pão Quente" {
		t.Fatalf("name = %q, want the first proposal's", valueOf(a.Items[0].Observed.Name))
	}

	if rec := entitiesAPI(t, pool, following, "/entities?q="+strings.Repeat("x", 121)); rec.Code != http.StatusBadRequest {
		t.Fatalf("a long q = %d, want 400", rec.Code)
	}
}


// urlEncode escapes q for a query string.
func urlEncode(q string) string {
	return strings.NewReplacer("%", "%25", " ", "%20").Replace(q)
}
