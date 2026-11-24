package claim


import (
	"context"
	"encoding/json"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/claim/sqlc"
)


// --- CONSTANTS ---

// freshFor is how old a passing lookup may be and still count: the recheck
// interval and some slack, so a checker that stopped does not keep a claim
// verified forever on an old answer.
const freshFor = Recheck + 5*24*time.Hour


// --- CODE ---

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


// AuthorOf reads the author of a signed envelope, "" when it has none.
func AuthorOf(envelope []byte) string {
	var head struct {
		Author string `json:"author"`
	}

	_ = json.Unmarshal(envelope, &head)

	return head.Author
}


// utc returns t in UTC, or nil.
func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}

	u := t.UTC()

	return &u
}
