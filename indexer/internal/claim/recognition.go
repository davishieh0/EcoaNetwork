package claim


import (
	"context"
	"encoding/json"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/claim/sqlc"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/entity"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/projection"
)


// --- CONSTANTS ---

// freshFor is how old a passing lookup may be and still count: the recheck
// interval and some slack, so a checker that stopped does not keep a claim
// verified forever on an old answer.
const freshFor = Recheck + 5*24*time.Hour


// --- CODE ---

// entityRepresentative is the shape the entities route answers.
type entityRepresentative = entity.Representative


// Recognized is a claim that counts here, as the API shows it.
type Recognized struct {
	EventID   string     `json:"eventId"`
	ClaimType string     `json:"claimType"`
	Entity    *string    `json:"entity,omitempty"`
	ExpiresAt time.Time  `json:"expiresAt"`

	// Since is when a domain claim began to verify without a gap.
	Since *time.Time `json:"since,omitempty"`

	// DNSSEC reports a domain claim verified by a DNSSEC-validated answer.
	DNSSEC bool `json:"dnssec,omitempty"`
}


// Representative is the one identity with authority over an entity: the
// sole holder of a domain claim on it that counts.
type Representative struct {
	Identity string     `json:"identity"`
	EventID  string     `json:"eventId"`
	Since    *time.Time `json:"since,omitempty"`
}


// Recognition is what the API needs to act on claims: whose word counts,
// and the claims and lookups stored.
type Recognition struct {
	policy  Policy
	queries *sqlc.Queries
	now     func() time.Time
}


// NewRecognition returns the recognition of policy, reading from db.
func NewRecognition(policy Policy, db sqlc.DBTX) *Recognition {
	return &Recognition{policy: policy, queries: sqlc.New(db), now: time.Now}
}


// Of returns the claims that count for each of subjects.
func (r *Recognition) Of(ctx context.Context,
                         subjects []string) (map[string][]Recognized, error) {
	now := r.now()

	rows, err := r.queries.RecognizedOf(ctx, sqlc.RecognizedOfParams{
		Subjects:   subjects,
		Now:        now,
		FreshAfter: now.Add(-freshFor),
		Verifiers:  r.policy.Human(),
	})
	if err != nil {
		return nil, err
	}

	held := map[string][]Recognized{}

	for _, row := range rows {
		held[row.Subject] = append(held[row.Subject], Recognized{
			EventID:   row.EventID,
			ClaimType: row.ClaimType,
			Entity:    row.Entity,
			ExpiresAt: row.ExpiresAt.UTC(),
			Since:     utc(row.VerifiedSince),
			DNSSEC:    row.Dnssec,
		})
	}

	return held, nil
}


// Representatives returns the representative of each of entities that has
// one. Two identities verified for one entity is a tie nobody wins: the
// entity then has none, as projection.ResolveFields decides.
func (r *Recognition) Representatives(
	ctx context.Context,
	entities []string,
) (map[string]Representative, error) {
	now := r.now()

	rows, err := r.queries.RepresentativesOf(ctx, sqlc.RepresentativesOfParams{
		Entities:   entities,
		Now:        now,
		FreshAfter: now.Add(-freshFor),
	})
	if err != nil {
		return nil, err
	}

	byEntity := map[string][]sqlc.RepresentativesOfRow{}
	for _, row := range rows {
		byEntity[row.Entity] = append(byEntity[row.Entity], row)
	}

	reps := map[string]Representative{}

	for entity, holding := range byEntity {
		first := holding[0]
		sole := true

		for _, other := range holding[1:] {
			if other.Subject != first.Subject {
				sole = false
			}
		}

		if sole {
			reps[entity] = Representative{
				Identity: first.Subject,
				EventID:  first.EventID,
				Since:    utc(first.VerifiedSince),
			}
		}
	}

	return reps, nil
}


// RepresentativeOf returns entity's representative, nil when it has none.
func (r *Recognition) RepresentativeOf(
	ctx context.Context,
	entity string,
) (*entityRepresentative, error) {
	reps, err := r.Representatives(ctx, []string{entity})
	if err != nil {
		return nil, err
	}

	rep, ok := reps[entity]
	if !ok {
		return nil, nil
	}

	return &entityRepresentative{
		Identity: rep.Identity,
		EventID:  rep.EventID,
		Since:    rep.Since,
	}, nil
}


// Identities is what projection.ResolveFields takes about each of authors:
// the claims that count for them. Weights wait on reputation.
func (r *Recognition) Identities(
	ctx context.Context,
	authors []string,
) (map[string]projection.Identity, error) {
	held, err := r.Of(ctx, authors)
	if err != nil {
		return nil, err
	}

	identities := map[string]projection.Identity{}

	for author, claims := range held {
		var judged []projection.Claim

		for _, c := range claims {
			if c.Entity == nil {
				continue
			}

			judged = append(judged, projection.Claim{
				Entity:         *c.Entity,
				Recognized:     true,
				SelfVerifiable: c.ClaimType == "domain",
			})
		}

		identities[author] = projection.Identity{Claims: judged}
	}

	return identities, nil
}


// EntityOfChain returns the entity a review chain is about.
func (r *Recognition) EntityOfChain(ctx context.Context,
                                    root string) (string, error) {
	return r.queries.EntityOfChain(ctx, root)
}


// AuthorOf reads the author of a signed envelope, "" when it has none.
func AuthorOf(envelope []byte) string {
	var head struct {
		Author string `json:"author"`
	}

	_ = json.Unmarshal(envelope, &head)

	return head.Author
}


// Official reports whether one of claims makes its holder the voice of
// entity: a domain claim on that very entity.
func Official(claims []Recognized, entity string) bool {
	for _, c := range claims {
		if c.ClaimType == "domain" && c.Entity != nil && *c.Entity == entity {
			return true
		}
	}

	return false
}


// utc returns t in UTC, or nil.
func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}

	u := t.UTC()

	return &u
}
