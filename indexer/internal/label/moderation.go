package label


import (
	"context"
	"net/http"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label/sqlc"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/respond"
)


// --- CODE ---

// Applied is one label that counts: by a followed moderator, not revoked.
type Applied struct {
	EventID   string `json:"eventId"`
	Label     string `json:"label"`
	Moderator string `json:"moderator"`
}


// Moderation is what the API needs to act on labels: whom this indexer
// follows, and the labels they published.
type Moderation struct {
	policy  Policy
	queries *sqlc.Queries
}


// NewModeration returns the moderation of policy, reading labels from db.
func NewModeration(policy Policy, db sqlc.DBTX) *Moderation {
	return &Moderation{policy: policy, queries: sqlc.New(db)}
}


// Moderators returns every key whose labels count: each key of every
// identity the moderators file names, so a moderator who rotated is still
// followed. Worked out on each call, so a new rotation counts at once.
func (m *Moderation) Moderators(ctx context.Context) ([]string, error) {
	listed := m.policy.Moderators()
	if len(listed) == 0 {
		return listed, nil
	}

	return m.queries.KeysOfFollowed(ctx, listed)
}


// OnChains returns the labels that count on each of roots.
func (m *Moderation) OnChains(ctx context.Context,
                              roots []string) (map[string][]Applied, error) {
	moderators, err := m.Moderators(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := m.queries.ChainLabels(ctx, sqlc.ChainLabelsParams{
		Roots:      roots,
		Moderators: moderators,
	})
	if err != nil {
		return nil, err
	}

	applied := map[string][]Applied{}

	for _, row := range rows {
		applied[row.ChainRoot] = append(applied[row.ChainRoot], Applied{
			EventID:   row.EventID,
			Label:     row.Label,
			Moderator: row.Moderator,
		})
	}

	return applied, nil
}


// OnResponses returns the labels that count on each of responses.
func (m *Moderation) OnResponses(ctx context.Context,
                                 responses []string) (map[string][]Applied, error) {
	moderators, err := m.Moderators(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := m.queries.ResponseLabels(ctx, sqlc.ResponseLabelsParams{
		Responses:  responses,
		Moderators: moderators,
	})
	if err != nil {
		return nil, err
	}

	applied := map[string][]Applied{}

	for _, row := range rows {
		applied[row.Response] = append(applied[row.Response], Applied{
			EventID:   row.EventID,
			Label:     row.Label,
			Moderator: row.Moderator,
		})
	}

	return applied, nil
}


// OnProfiles returns the labels that count on each of the profile.updated
// events profiles.
func (m *Moderation) OnProfiles(ctx context.Context,
                                profiles []string) (map[string][]Applied, error) {
	moderators, err := m.Moderators(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := m.queries.ProfileLabels(ctx, sqlc.ProfileLabelsParams{
		Profiles:   profiles,
		Moderators: moderators,
	})
	if err != nil {
		return nil, err
	}

	applied := map[string][]Applied{}

	for _, row := range rows {
		applied[row.Profile] = append(applied[row.Profile], Applied{
			EventID:   row.EventID,
			Label:     row.Label,
			Moderator: row.Moderator,
		})
	}

	return applied, nil
}


// Removed reports whether event id is removed by a followed moderator.
func (m *Moderation) Removed(ctx context.Context, id string) (bool, error) {
	moderators, err := m.Moderators(ctx)
	if err != nil {
		return false, err
	}

	return m.queries.IsRemoved(ctx, sqlc.IsRemovedParams{
		Moderators: moderators,
		Removing:   Removing(),
		EventID:    id,
	})
}


// AnyRemoves reports whether one of applied removes what it labels.
func AnyRemoves(applied []Applied) bool {
	for _, a := range applied {
		if EffectOf(a.Label) == Removed {
			return true
		}
	}

	return false
}


// ShowLabeled reads ?labeled=: "show" asks to see what is hidden by default,
// marked; absent hides it. Anything else answers 400 and reports false.
func ShowLabeled(w http.ResponseWriter, r *http.Request) (bool, bool) {
	switch r.URL.Query().Get("labeled") {
	case "":
		return false, true
	case "show":
		return true, true
	}

	respond.Error(w, http.StatusBadRequest, "invalid_labeled",
	              "labeled is show, or absent.")

	return false, false
}
