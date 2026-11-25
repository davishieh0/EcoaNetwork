package projector


import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/claim"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/entity"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/identities"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label"
	"github.com/jackc/pgx/v5/pgxpool"
)


// --- CODE ---

// records answers each name with the records set for it.
type records map[string][]string


func (r records) TXT(_ context.Context, name string) (claim.Answer, error) {
	return claim.Answer{Records: r[name]}, nil
}


// claimBy is a verification.claim_created signed by the key of seed.
func claimBy(t *testing.T, seed byte, payload string) ([]byte, string) {
	t.Helper()

	return signAs(t, seed, "verification.claim_created",
	              "2026-11-20T10:00:00Z", payload)
}


// domainClaim is the self-signed domain claim of seed on domain.
func domainClaim(t *testing.T, seed byte, domain string) ([]byte, string) {
	t.Helper()

	return claimBy(t, seed, `{"claimType":"domain","subject":"`+didOf(seed)+
	               `","entity":"domain:`+domain+`","evidenceRef":`+
	               `"dns-txt:_ecoa-verify.`+domain+`",`+
	               `"expiresAt":"2027-11-01T00:00:00Z"}`)
}


// check runs one round of lookups through recs.
func check(t *testing.T, pool *pgxpool.Pool, recs records) {
	t.Helper()

	if err := claim.NewChecker(pool, recs).Round(context.Background());
		err != nil {
		t.Fatalf("Round: %v", err)
	}
}


// entityAt answers GET /entities/{entity}, which must exist.
func entityAt(t *testing.T, pool *pgxpool.Pool, id string) entity.Entity {
	t.Helper()

	rec := entitiesAPI(t, pool, label.Policy{}, "/entities/"+id)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /entities/%s = %d %q", id, rec.Code, rec.Body.String())
	}

	return entityOf(t, rec)
}


// named is an entity.updated by someone else naming domain's entity, so it
// exists.
func named(t *testing.T, domain string) []byte {
	t.Helper()

	raw, _ := signAs(t, 'z', "entity.updated", "2026-11-01T10:00:00Z",
	                 `{"entity":"domain:`+domain+`","name":"`+domain+`"}`)

	return raw
}


// officialOf answers the responses of review and reports, for each, whether
// it is official.
func officialOf(t *testing.T, pool *pgxpool.Pool, review string) []bool {
	t.Helper()

	rec := responsesPath(t, pool, "/reviews/"+review+"/responses")

	var body struct {
		Items []struct {
			Observed struct {
				Official bool `json:"official"`
			} `json:"observed"`
		} `json:"items"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %d %q: %v", rec.Code, rec.Body.String(), err)
	}

	var official []bool
	for _, item := range body.Items {
		official = append(official, item.Observed.Official)
	}

	return official
}


// claimsOf answers GET /identities/{did} recognizing policy's verifiers,
// and returns the claim types that count.
func claimsOf(t *testing.T,
              pool *pgxpool.Pool,
              policy claim.Policy,
              did string) []string {
	t.Helper()

	h := identities.NewHandler(identities.NewRepository(pool),
	                           claim.NewRecognition(policy, pool))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /identities/{did}", h.Get)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
	                                       "/identities/"+did, nil))

	var body struct {
		Claims []claim.Recognized `json:"claims"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %d %q: %v", rec.Code, rec.Body.String(), err)
	}

	var types []string
	for _, c := range body.Claims {
		types = append(types, c.ClaimType)
	}

	return types
}


func TestAVerifiedDomainClaimMakesItsHolderTheRepresentative(t *testing.T) {
	// Arrange: the first proposal names it, then the holder proposes another
	pool := migrated(t)
	const domain = "padaria.test"
	entityID := "domain:" + domain

	first, _ := signAs(t, 'e', "entity.updated", "2026-11-10T10:00:00Z",
	                   `{"entity":"`+entityID+`","name":"Primeiro nome"}`)
	own, _ := signAs(t, 'd', "entity.updated", "2026-11-12T10:00:00Z",
	                 `{"entity":"`+entityID+`","name":"Nome oficial"}`)
	claimed, _ := domainClaim(t, 'd', domain)

	feed(t, pool, "n1", first, own, claimed)

	// Assert: stored, not yet looked up, so nothing counts
	before := entityAt(t, pool, entityID)
	if before.Observed.Representative != nil ||
		valueOf(before.Observed.Name) != "Primeiro nome" {
		t.Fatalf("before the lookup = %+v", before.Observed)
	}

	// Act: the record is there
	check(t, pool, records{"_ecoa-verify." + domain: {
		"ecoa-identity=" + didOf('d')}})

	// Assert: the representative, and tier 1 over the first proposal
	after := entityAt(t, pool, entityID)
	if after.Observed.Representative == nil ||
		after.Observed.Representative.Identity != didOf('d') ||
		after.Observed.Representative.Since == nil {
		t.Fatalf("representative = %+v", after.Observed.Representative)
	}

	if valueOf(after.Observed.Name) != "Nome oficial" ||
		after.Observed.Name.Tier != 1 {
		t.Fatalf("name = %+v, want the holder's at tier 1",
		         after.Observed.Name)
	}
}


func TestARecordForAnotherIdentityVerifiesNothing(t *testing.T) {
	// Arrange
	pool := migrated(t)
	claimed, _ := domainClaim(t, 'd', "loja.test")
	feed(t, pool, "n1", named(t, "loja.test"), claimed)

	// Act
	check(t, pool, records{"_ecoa-verify.loja.test": {
		"ecoa-identity=" + didOf('x')}})

	// Assert
	if rep := entityAt(t, pool, "domain:loja.test").Observed.Representative;
		rep != nil {
		t.Fatalf("representative = %+v, want none", rep)
	}
}


func TestASelfSignedClaimAboutSomeoneElseIsNeverLookedUp(t *testing.T) {
	// Arrange: d signs that x holds the domain
	pool := migrated(t)
	claimed, _ := claimBy(t, 'd', `{"claimType":"domain","subject":"`+
	                      didOf('x')+`","entity":"domain:outra.test",`+
	                      `"evidenceRef":"dns-txt:_ecoa-verify.outra.test",`+
	                      `"expiresAt":"2027-11-01T00:00:00Z"}`)
	feed(t, pool, "n1", named(t, "outra.test"), claimed)

	// Act: even with x's record in place
	check(t, pool, records{"_ecoa-verify.outra.test": {
		"ecoa-identity=" + didOf('x')}})

	// Assert
	if rep := entityAt(t, pool, "domain:outra.test").Observed.Representative;
		rep != nil {
		t.Fatalf("representative = %+v, want none", rep)
	}
}


func TestARevokedClaimStopsCounting(t *testing.T) {
	// Arrange
	pool := migrated(t)
	claimed, claimID := domainClaim(t, 'd', "fim.test")
	feed(t, pool, "n1", named(t, "fim.test"), claimed)
	check(t, pool, records{"_ecoa-verify.fim.test": {
		"ecoa-identity=" + didOf('d')}})

	// another key cannot revoke it
	foreign, _ := signAs(t, 'x', "verification.claim_revoked",
	                     "2026-11-21T10:00:00Z", `{"target":"`+claimID+`"}`)
	feed(t, pool, "n1", foreign)

	if entityAt(t, pool, "domain:fim.test").Observed.Representative == nil {
		t.Fatalf("revoked by another key, want it to still count")
	}

	// Act: its issuer revokes it
	revoked, _ := signAs(t, 'd', "verification.claim_revoked",
	                     "2026-11-22T10:00:00Z", `{"target":"`+claimID+`"}`)
	feed(t, pool, "n1", revoked)

	// Assert
	if rep := entityAt(t, pool, "domain:fim.test").Observed.Representative;
		rep != nil {
		t.Fatalf("representative = %+v after the revocation", rep)
	}
}


func TestTwoVerifiedHoldersLeaveTheEntityWithoutRepresentative(t *testing.T) {
	// Arrange: the domain vouches for both
	pool := migrated(t)
	one, _ := domainClaim(t, 'd', "dupla.test")
	two, _ := domainClaim(t, 'f', "dupla.test")
	feed(t, pool, "n1", named(t, "dupla.test"), one, two)

	// Act
	check(t, pool, records{"_ecoa-verify.dupla.test": {
		"ecoa-identity=" + didOf('d'), "ecoa-identity=" + didOf('f')}})

	// Assert: a tie nobody wins
	if rep := entityAt(t, pool, "domain:dupla.test").Observed.Representative;
		rep != nil {
		t.Fatalf("representative = %+v, want none on a tie", rep)
	}
}


func TestTheRepresentativesResponseIsOfficial(t *testing.T) {
	// Arrange: a review of the domain, answered by its holder and a customer
	pool := migrated(t)
	reviewed, reviewID := signAs(t, 'a', "review.created",
	                             "2026-11-15T10:00:00Z",
	                             `{"entity":"domain:padaria.test","rating":2,`+
	                             `"body":"Demorou."}`)
	claimed, _ := domainClaim(t, 'd', "padaria.test")
	official, _ := signAs(t, 'd', "review.response_created",
	                      "2026-11-16T10:00:00Z",
	                      `{"target":"`+reviewID+`","body":"Pedimos desculpas."}`)
	customer, _ := signAs(t, 'c', "review.response_created",
	                      "2026-11-16T11:00:00Z",
	                      `{"target":"`+reviewID+`","body":"Comigo também."}`)

	feed(t, pool, "n1", reviewed, claimed, official, customer)
	check(t, pool, records{"_ecoa-verify.padaria.test": {
		"ecoa-identity=" + didOf('d')}})

	// Act
	got := officialOf(t, pool, reviewID)

	// Assert
	if len(got) != 2 || !got[0] || got[1] {
		t.Fatalf("official = %v, want the holder's only", got)
	}
}


func TestAHumanClaimCountsOnlyFromARecognizedVerifier(t *testing.T) {
	// Arrange: v issues a human claim about c
	pool := migrated(t)
	claimed, _ := claimBy(t, 'v', `{"claimType":"human","subject":"`+
	                      didOf('c')+`","expiresAt":"2028-11-01T00:00:00Z"}`)
	feed(t, pool, "n1", claimed)

	// Act
	unknown := claimsOf(t, pool, claim.Policy{}, didOf('c'))
	followed := claimsOf(t, pool, claim.Recognize(didOf('v')), didOf('c'))

	// Assert
	if len(unknown) != 0 {
		t.Fatalf("claims without the verifier = %v, want none", unknown)
	}

	if len(followed) != 1 || followed[0] != "human" {
		t.Fatalf("claims with the verifier = %v, want human", followed)
	}
}

