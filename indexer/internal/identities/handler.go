package identities


import (
	"net/http"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/respond"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
)


// --- CODE ---

// Handler answers the routes of the identities.
type Handler struct {
	repo *Repository
}


// NewHandler returns the handler of the identity routes.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}


// Get answers GET /identities/{did}: the identity any of its keys belongs
// to, its keys in order, whether it is contested, and its revocations.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	did := r.PathValue("did")

	if _, err := identity.Parse(did); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_identity",
		              "Not a did:key for an Ed25519 key.")
		return
	}

	id, err := h.repo.Of(r.Context(), did)
	if err != nil {
		respond.Failed(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, id)
}
