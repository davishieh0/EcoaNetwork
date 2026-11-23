package review


import (
	"errors"
	"net/http"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/ids"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/page"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/respond"
)


// --- CODE ---

// Handler answers the routes of the reviews.
type Handler struct {
	repo       *Repository
	moderation *label.Moderation
}


// derived is what the indexer worked out about a version, from the events
// alone: what any honest indexer must agree on.
type derived struct {
	ChainRoot string `json:"chainRoot"`
	Head      bool   `json:"head"`
	Withdrawn bool   `json:"withdrawn"`
	Edited    *bool  `json:"edited,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`

	// the votes that count on this very version
	Endorsements int32 `json:"endorsements"`
	Disputes     int32 `json:"disputes"`
}


// observed is what the indexer observed of a version.
type observed struct {
	ReceivedAt *time.Time `json:"receivedAt,omitempty"`

	// the labels that count, by the moderators this indexer follows: another
	// indexer, following others, shows others
	Labels []label.Applied `json:"labels,omitempty"`
}


// listing is the answer of GET /entities/{entity}/reviews.
type listing struct {
	Items []respond.Item `json:"items"`

	// Next is the ?after= of the next page, null on the last one.
	Next *string `json:"next"`
}


// detail is the answer of GET /reviews/{eventId}.
type detail struct {
	Review   respond.Item   `json:"review"`
	Versions []respond.Item `json:"versions"`
}


// NewHandler returns the handler of the review routes, reading from repo and
// acting on labels by moderation.
func NewHandler(repo *Repository, moderation *label.Moderation) *Handler {
	return &Handler{repo: repo, moderation: moderation}
}


// ListByEntity answers GET /entities/{entity}/reviews: the head of each of
// the entity's chains, newest original first, a page at a time.
func (h *Handler) ListByEntity(w http.ResponseWriter, r *http.Request) {
	entity := r.PathValue("entity")

	if !ids.IsEntityID(entity) {
		respond.Error(w, http.StatusBadRequest, "invalid_entity_id",
		              "Not an entity identifier the protocol allows.")
		return
	}

	p, ok := page.FromRequest(w, r)
	if !ok {
		return
	}

	show, ok := label.ShowLabeled(w, r)
	if !ok {
		return
	}

	moderators, err := h.moderation.Moderators(r.Context())
	if err != nil {
		respond.Failed(w, err)
		return
	}

	hide := Hide{
		Moderators: moderators,
		Labels:     label.Hiding(show),
	}

	listed, next, err := h.repo.HeadsByEntity(r.Context(), entity, p, hide)
	if err != nil {
		respond.Failed(w, err)
		return
	}

	roots := make([]string, 0, len(listed))
	for _, l := range listed {
		roots = append(roots, l.ChainRoot)
	}

	labels, err := h.moderation.OnChains(r.Context(), roots)
	if err != nil {
		respond.Failed(w, err)
		return
	}

	answer := listing{Items: make([]respond.Item, 0, len(listed))}

	for _, l := range listed {
		edited := l.Edited

		answer.Items = append(answer.Items, item(l.Version, derived{
			ChainRoot: l.ChainRoot,
			Head:      true,
			Edited:    &edited,
		}, labels[l.ChainRoot]))
	}

	answer.Next = page.Next(next)

	respond.JSON(w, http.StatusOK, answer)
}


// Get answers GET /reviews/{eventId}: one version, with every version of its
// chain.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("eventId")

	if !ids.IsEventID(id) {
		respond.Error(w, http.StatusBadRequest, "invalid_event_id",
		              "An eventId is evt_ and 52 base32 characters.")
		return
	}

	review, err := h.repo.ByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "review_not_found",
		              "No review with this id.")
		return
	}

	if err != nil {
		respond.Failed(w, err)
		return
	}

	// the author withdrew it: what it said leaves the reading path
	if review.Withdrawn {
		respond.Error(w, http.StatusGone, "review_withdrawn",
		              "This review was withdrawn by its author.")
		return
	}

	labels, err := h.moderation.OnChains(r.Context(),
	                                     []string{review.ChainRoot})
	if err != nil {
		respond.Failed(w, err)
		return
	}

	// a followed moderator removed it: nothing of it comes back
	applied := labels[review.ChainRoot]
	if label.AnyRemoves(applied) {
		respond.Removed(w)
		return
	}

	versions, err := h.repo.Versions(r.Context(), review.ChainRoot)
	if err != nil {
		respond.Failed(w, err)
		return
	}

	answer := detail{
		Review: item(review.Version, derivedOf(review, review.EventID),
		             applied),
		Versions: make([]respond.Item, 0, len(versions)),
	}

	for _, v := range versions {
		answer.Versions = append(answer.Versions,
		                         item(v, derivedOf(review, v.EventID), applied))
	}

	respond.JSON(w, http.StatusOK, answer)
}


// derivedOf is what version id of review's chain is, in that chain.
func derivedOf(review Review, id string) derived {
	return derived{
		ChainRoot: review.ChainRoot,
		Head:      id == review.Head,
		Withdrawn: review.Withdrawn,
		Truncated: review.Truncated,
	}
}


// item is v as the API returns it, with the labels that count on its chain.
func item(v Version, d derived, labels []label.Applied) respond.Item {
	d.Endorsements, d.Disputes = v.Endorsements, v.Disputes

	return respond.Item{
		Envelope: string(v.Envelope),
		Derived:  d,
		Observed: observed{ReceivedAt: utc(v.ReceivedAt), Labels: labels},
	}
}


// utc returns t in UTC, or nil.
func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}

	u := t.UTC()

	return &u
}
