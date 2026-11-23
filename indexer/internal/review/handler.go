package review


import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/ids"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/respond"
)


// --- CONSTANTS ---

const (
	// defaultLimit is a page's size when ?limit= is absent.
	defaultLimit = 20

	// maxLimit is the largest ?limit= accepted.
	maxLimit = 100
)


// --- CODE ---

// Handler answers the routes of the reviews.
type Handler struct {
	repo *Repository
}


// derived is what the indexer worked out about a version, from the events
// alone: what any honest indexer must agree on.
type derived struct {
	ChainRoot string `json:"chainRoot"`
	Head      bool   `json:"head"`
	Withdrawn bool   `json:"withdrawn"`
	Edited    *bool  `json:"edited,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}


// observed is what the indexer observed of a version.
type observed struct {
	ReceivedAt *time.Time `json:"receivedAt,omitempty"`
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


// NewHandler returns the handler of the review routes, reading from repo.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
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

	page, ok := pageOf(w, r)
	if !ok {
		return
	}

	listed, next, err := h.repo.HeadsByEntity(r.Context(), entity, page)
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
		}))
	}

	if next != nil {
		cursor := next.Encode()
		answer.Next = &cursor
	}

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

	versions, err := h.repo.Versions(r.Context(), review.ChainRoot)
	if err != nil {
		respond.Failed(w, err)
		return
	}

	answer := detail{
		Review:   item(review.Version, derivedOf(review, review.EventID)),
		Versions: make([]respond.Item, 0, len(versions)),
	}

	for _, v := range versions {
		answer.Versions = append(answer.Versions,
		                         item(v, derivedOf(review, v.EventID)))
	}

	respond.JSON(w, http.StatusOK, answer)
}


// pageOf reads ?after= and ?limit=, or answers 400 and reports false.
func pageOf(w http.ResponseWriter, r *http.Request) (Page, bool) {
	page := Page{Limit: defaultLimit}

	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > maxLimit {
			respond.Error(w, http.StatusBadRequest, "invalid_limit",
			              "limit is a whole number from 1 to 100.")
			return Page{}, false
		}

		page.Limit = limit
	}

	if raw := r.URL.Query().Get("after"); raw != "" {
		cursor, err := DecodeCursor(raw)
		if err != nil {
			respond.Error(w, http.StatusBadRequest, "invalid_cursor",
			              "after is the next of a previous page.")
			return Page{}, false
		}

		page.After = &cursor
	}

	return page, true
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


// item is v as the API returns it.
func item(v Version, d derived) respond.Item {
	return respond.Item{
		Envelope: string(v.Envelope),
		Derived:  d,
		Observed: observed{ReceivedAt: utc(v.ReceivedAt)},
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
