package profile


import (
	"net/http"
	"strings"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/respond"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
)


// --- CONSTANTS ---

// maxIDs is the most identities GET /profiles?ids= takes: a full page.
const maxIDs = 100


// --- CODE ---

// Handler answers the routes of the profiles.
type Handler struct {
	repo       *Repository
	moderation *label.Moderation
}


// derived is what any honest indexer must agree on about a profile.
type derived struct {
	Identity string `json:"identity"`
}


// observed is what this indexer observed of a profile.
type observed struct {
	ReceivedAt *time.Time      `json:"receivedAt,omitempty"`
	Labels     []label.Applied `json:"labels,omitempty"`
}


// listing is the answer of GET /profiles?ids=.
type listing struct {
	Items []respond.Item `json:"items"`
}


// NewHandler returns the handler of the profile routes.
func NewHandler(repo *Repository, moderation *label.Moderation) *Handler {
	return &Handler{repo: repo, moderation: moderation}
}


// Get answers GET /profiles/{did}: the identity's current profile. A display
// name is verified by nothing; a client shows it beside the identity, never
// as if it were the identity.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	did := r.PathValue("did")

	if _, err := identity.Parse(did); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_identity",
		              "Not a did:key for an Ed25519 key.")
		return
	}

	items, removed, err := h.profiles(r, []string{did})
	if err != nil {
		respond.Failed(w, err)
		return
	}

	if removed > 0 {
		respond.Removed(w)
		return
	}

	if len(items) == 0 {
		respond.Error(w, http.StatusNotFound, "profile_not_found",
		              "This identity published no profile.")
		return
	}

	respond.JSON(w, http.StatusOK, items[0])
}


// List answers GET /profiles?ids=: the current profiles of up to 100
// identities, in the order asked, those with none left out, so a client gets
// the authors of a page in one call.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var dids []string

	for _, did := range strings.Split(r.URL.Query().Get("ids"), ",") {
		if did = strings.TrimSpace(did); did != "" {
			dids = append(dids, did)
		}
	}

	if len(dids) == 0 || len(dids) > maxIDs {
		respond.Error(w, http.StatusBadRequest, "invalid_ids",
		              "ids is from 1 to 100 did:key, comma-separated.")
		return
	}

	for _, did := range dids {
		if _, err := identity.Parse(did); err != nil {
			respond.Error(w, http.StatusBadRequest, "invalid_identity",
			              "Not a did:key for an Ed25519 key: "+did)
			return
		}
	}

	items, _, err := h.profiles(r, dids)
	if err != nil {
		respond.Failed(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, listing{Items: items})
}


// profiles returns the current profiles of dids in their order, without
// those a followed moderator removed, and how many it left out for that.
func (h *Handler) profiles(r *http.Request,
                           dids []string) ([]respond.Item, int, error) {
	found, err := h.repo.Of(r.Context(), dids)
	if err != nil {
		return nil, 0, err
	}

	events := make([]string, 0, len(found))
	for _, p := range found {
		events = append(events, p.EventID)
	}

	labels, err := h.moderation.OnProfiles(r.Context(), events)
	if err != nil {
		return nil, 0, err
	}

	items := make([]respond.Item, 0, len(found))
	removed := 0

	for _, did := range dids {
		p, ok := found[did]
		if !ok {
			continue
		}

		applied := labels[p.EventID]
		if label.AnyRemoves(applied) {
			removed++
			continue
		}

		items = append(items, respond.Item{
			Envelope: string(p.Envelope),
			Derived:  derived{Identity: p.Identity},
			Observed: observed{ReceivedAt: utc(p.ReceivedAt), Labels: applied},
		})
	}

	return items, removed, nil
}


// utc returns t in UTC, or nil.
func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}

	u := t.UTC()

	return &u
}
