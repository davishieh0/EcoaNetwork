package profile


import (
	"context"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/profile/sqlc"
)


// --- CODE ---

// Profile is one identity's current profile, as signed.
type Profile struct {
	// Identity names the identity it describes: the first key of its chain,
	// whichever of its keys signed it.
	Identity string

	// EventID is the profile.updated that holds now.
	EventID string

	// Envelope is that event byte for byte as it was signed.
	Envelope []byte

	// ReceivedAt is when the network had it, nil when no node reported it.
	ReceivedAt *time.Time
}


// Repository reads the profiles.
type Repository struct {
	queries *sqlc.Queries
}


// NewRepository returns a repository over db.
func NewRepository(db sqlc.DBTX) *Repository {
	return &Repository{queries: sqlc.New(db)}
}


// Of returns the current profile of each of dids, by did, whichever key of
// its identity it is; one whose identity never published a profile is absent.
func (r *Repository) Of(ctx context.Context,
                        dids []string) (map[string]Profile, error) {
	resolved, err := r.queries.IdentitiesOf(ctx, dids)
	if err != nil {
		return nil, err
	}

	identities := make([]string, 0, len(resolved))
	for _, row := range resolved {
		identities = append(identities, row.Identity)
	}

	rows, err := r.queries.ProfilesOf(ctx, identities)
	if err != nil {
		return nil, err
	}

	byIdentity := map[string]Profile{}

	for _, row := range rows {
		byIdentity[row.Identity] = Profile{
			Identity:   row.Identity,
			EventID:    row.EventID,
			Envelope:   row.Envelope,
			ReceivedAt: row.ReceivedAt,
		}
	}

	profiles := map[string]Profile{}

	for _, row := range resolved {
		if p, ok := byIdentity[row.Identity]; ok {
			profiles[row.Did] = p
		}
	}

	return profiles, nil
}
