package event


import (
	"errors"
	"net/http"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/ids"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/respond"
)


// --- CODE ---

// Handler answers the routes of the events table.
type Handler struct {
	repo       *Repository
	moderation *label.Moderation
}


// observed is what the indexer observed of an event.
type observed struct {
	ReceivedAt *time.Time `json:"receivedAt,omitempty"`

	// the nodes this indexer saw holding it; another, following other
	// nodes, sees others
	Nodes []holder `json:"nodes,omitempty"`

	// the labels that count, by the moderators this indexer follows
	Labels []label.Applied `json:"labels,omitempty"`
}


// holder is one node that holds an event, as the API names it.
type holder struct {
	Node       string     `json:"node"`
	Sequence   int64      `json:"sequence"`
	ReceivedAt *time.Time `json:"receivedAt,omitempty"`
}


// NewHandler returns the handler of the events routes, reading from repo
// and leaving out what a followed moderator removed.
func NewHandler(repo *Repository, moderation *label.Moderation) *Handler {
	return &Handler{repo: repo, moderation: moderation}
}


// Get answers GET /events/{eventId}: the envelope alone.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("eventId")

	if !ids.IsEventID(id) {
		respond.Error(w, http.StatusBadRequest, "invalid_event_id_error",
		              "An eventId is evt_ and 52 base32 characters.")
		return
	}

	e, err := h.repo.ByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "event_not_found_error",
		              "No event with this id.")
		return
	}

	if err != nil {
		respond.Failed(w, err)
		return
	}

	// the envelope too: the indexer is where removed content leaves the
	// reading path, so it must not leave by this route either
	removed, err := h.moderation.Removed(r.Context(), id)
	if err != nil {
		respond.Failed(w, err)
		return
	}

	if removed {
		respond.Removed(w)
		return
	}

	holders, err := h.repo.Holders(r.Context(), id)
	if err != nil {
		respond.Failed(w, err)
		return
	}

	labels, err := h.moderation.OnEvent(r.Context(), id)
	if err != nil {
		respond.Failed(w, err)
		return
	}

	nodes := make([]holder, len(holders))
	for i, held := range holders {
		nodes[i] = holder{
			Node:       held.Node,
			Sequence:   held.Sequence,
			ReceivedAt: utc(held.ReceivedAt),
		}
	}

	respond.JSON(w, http.StatusOK, respond.Item{
		Envelope: string(e.Envelope),
		Derived:  struct{}{},
		Observed: observed{
			ReceivedAt: utc(e.ReceivedAt),
			Nodes:      nodes,
			Labels:     labels,
		},
	})
}


// utc returns t in UTC, or nil.
func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}

	u := t.UTC()

	return &u
}
