package event


import (
	"errors"
	"net/http"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/ids"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/respond"
)


// --- CODE ---

// Handler answers the routes of the events table.
type Handler struct {
	repo *Repository
}


// observed is what the indexer observed of an event.
type observed struct {
	ReceivedAt *time.Time `json:"receivedAt,omitempty"`
}


// NewHandler returns the handler of the events routes, reading from repo.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}


// Get answers GET /events/{eventId}: the envelope alone.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("eventId")

	if !ids.IsEventID(id) {
		respond.Error(w, http.StatusBadRequest, "invalid_event_id",
		              "An eventId is evt_ and 52 base32 characters.")
		return
	}

	e, err := h.repo.ByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "event_not_found",
		              "No event with this id.")
		return
	}

	if err != nil {
		respond.Failed(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, respond.Item{
		Envelope: string(e.Envelope),
		Derived:  struct{}{},
		Observed: observed{ReceivedAt: utc(e.ReceivedAt)},
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
