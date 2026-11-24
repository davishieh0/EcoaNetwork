package identities


import (
	"net/http"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/claim"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/respond"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
)


// --- CODE ---

// Handler answers the routes of the identities.
type Handler struct {
	repo        *Repository
	recognition *claim.Recognition
}


// answer is the identity, with the claims that count for any of its keys.
type answer struct {
	Identity

	Claims []claim.Recognized `json:"claims"`
}


// NewHandler returns the handler of the identity routes.
func NewHandler(repo *Repository, recognition *claim.Recognition) *Handler {
	return &Handler{repo: repo, recognition: recognition}
}


// Get answers GET /identities/{did}: the identity any of its keys belongs
// to, its keys in order, whether it is contested, and its revocations.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	did := r.PathValue("did")

	if _, err := identity.Parse(did); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_identity_error",
		              "Not a did:key for an Ed25519 key.")
		return
	}

	id, err := h.repo.Of(r.Context(), did)
	if err != nil {
		respond.Failed(w, err)
		return
	}

	held, err := h.recognition.Of(r.Context(), id.Keys)
	if err != nil {
		respond.Failed(w, err)
		return
	}

	claims := []claim.Recognized{}
	for _, key := range id.Keys {
		claims = append(claims, held[key]...)
	}

	respond.JSON(w, http.StatusOK, answer{Identity: id, Claims: claims})
}
