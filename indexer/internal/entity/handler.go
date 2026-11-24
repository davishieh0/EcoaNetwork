package entity


import (
	"errors"
	"net/http"
	"unicode/utf8"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/ids"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/page"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/respond"
)


// --- CONSTANTS ---

// maxQuery is the longest search text accepted: a name's own ceiling.
const maxQuery = 120


// --- GLOBALS ---

// categories is the protocol's fixed vocabulary (docs/protocol.md).
var categories = map[string]bool{
	"retail": true, "food-and-drink": true, "financial-services": true,
	"telecom": true, "utilities": true, "transport": true,
	"travel-and-lodging": true, "health": true, "education": true,
	"government": true, "real-estate": true, "professional-services": true,
	"entertainment": true, "technology": true, "manufacturing": true,
	"nonprofit": true, "other": true,
}


// --- CODE ---

// Handler answers the routes of the entities.
type Handler struct {
	repo       *Repository
	moderation *label.Moderation
}


// listing is the answer of GET /entities.
type listing struct {
	Items []Entity `json:"items"`
	Next  *string  `json:"next"`
}


// NewHandler returns the handler of the entity routes.
func NewHandler(repo *Repository, moderation *label.Moderation) *Handler {
	return &Handler{repo: repo, moderation: moderation}
}


// Get answers GET /entities/{entity}: the entity and the value of each of its
// fields.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	moderation, ok := h.moderation.ForRequest(w, r)
	if !ok {
		return
	}

	id := r.PathValue("entity")

	if !ids.IsEntityID(id) {
		respond.Error(w, http.StatusBadRequest, "invalid_entity_id_error",
		              "Not an entity identifier the protocol allows.")
		return
	}

	moderators, err := moderation.Moderators(r.Context())
	if err != nil {
		respond.Failed(w, err)
		return
	}

	e, err := h.repo.ByID(r.Context(), id, moderators)
	if errors.Is(err, ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "entity_not_found_error",
		              "No review or proposal names this entity.")
		return
	}

	if err != nil {
		respond.Failed(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, e)
}


// Search answers GET /entities?q=: the entities with a name that holds q,
// ignoring case and accents, newest first, a page at a time; every entity
// when q is absent.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	moderation, ok := h.moderation.ForRequest(w, r)
	if !ok {
		return
	}

	q := r.URL.Query().Get("q")

	if utf8.RuneCountInString(q) > maxQuery {
		respond.Error(w, http.StatusBadRequest, "invalid_query_error",
		              "q is at most 120 characters.")
		return
	}

	p, ok := page.FromRequest(w, r)
	if !ok {
		return
	}

	moderators, err := moderation.Moderators(r.Context())
	if err != nil {
		respond.Failed(w, err)
		return
	}

	var category *string
	if c := r.URL.Query().Get("category"); c != "" {
		if !categories[c] {
			respond.Error(w, http.StatusBadRequest, "invalid_category_error",
			              "category is one of the protocol's categories.")
			return
		}
		category = &c
	}

	found, next, err := h.repo.Search(r.Context(), q, moderators, p, category)
	if err != nil {
		respond.Failed(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, listing{Items: found, Next: page.Next(next)})
}
