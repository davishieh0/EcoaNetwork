package identities


import (
	"context"
	"errors"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/identities/sqlc"
	"github.com/jackc/pgx/v5"
)


// --- CODE ---

// Revocation is one key's identity.key_revoked.
type Revocation struct {
	Key     string  `json:"key"`
	EventID string  `json:"eventId"`
	Reason  *string `json:"reason,omitempty"`
}


// Identity is one identity as the API returns it.
type Identity struct {
	// Identity is the first key of its chain, which names it.
	Identity string `json:"identity"`

	// Keys is every key of it, in the order of its chain.
	Keys []string `json:"keys"`

	// Current is its last key: the one it signs with now.
	Current string `json:"current"`

	// Contested reports two successors at one of its keys: neither is
	// honored, and its new events deserve suspicion.
	Contested bool `json:"contested"`

	// Revoked is every revocation of one of its keys.
	Revoked []Revocation `json:"revoked"`
}


// Repository reads the identities.
type Repository struct {
	queries *sqlc.Queries
}


// NewRepository returns a repository over db.
func NewRepository(db sqlc.DBTX) *Repository {
	return &Repository{queries: sqlc.New(db)}
}


// Of returns the identity key belongs to. Every key is an identity, its own
// when it never rotated.
func (r *Repository) Of(ctx context.Context, key string) (Identity, error) {
	id := Identity{Identity: key, Keys: []string{key}, Current: key}

	assigned, err := r.queries.AssignmentOf(ctx, key)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return Identity{}, err
	default:
		rows, err := r.queries.KeysOfIdentity(ctx, assigned.Identity)
		if err != nil {
			return Identity{}, err
		}

		id.Identity = assigned.Identity
		id.Contested = assigned.Contested
		id.Keys = id.Keys[:0]

		for _, row := range rows {
			id.Keys = append(id.Keys, row.Key)
		}

		id.Current = id.Keys[len(id.Keys)-1]
	}

	revocations, err := r.queries.RevocationsOf(ctx, id.Keys)
	if err != nil {
		return Identity{}, err
	}

	id.Revoked = make([]Revocation, 0, len(revocations))

	for _, row := range revocations {
		id.Revoked = append(id.Revoked, Revocation{
			Key:     row.Key,
			EventID: row.EventID,
			Reason:  row.Reason,
		})
	}

	return id, nil
}
