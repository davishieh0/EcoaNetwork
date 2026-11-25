package response


import (
	"errors"
	"net/http"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/claim"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/ids"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/page"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/respond"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/review"
)


// --- CONSTANTS ---

// maxDepth is the deepest a response is shown. Deeper ones are shown at this
// depth, still saying what they answer, so none is hidden and a client never
// indents more than twice. The stored depth is the real one.
const maxDepth = 2


// --- CODE ---

// Handler answers the routes of the responses.
type Handler struct {
	responses   *Repository
	reviews     *review.Repository
	moderation  *label.Moderation
	recognition *claim.Recognition
}


// derived is what the indexer worked out about a response, from the events
// alone: what any honest indexer must agree on.
type derived struct {
	ChainRoot        string `json:"chainRoot"`
	ReplyTo          string `json:"replyTo"`
	RespondedVersion string `json:"respondedVersion"`

	// the version it answered is no longer the review's current one
	TextChanged bool `json:"textChanged"`

	Depth int32 `json:"depth"`
}


// observed is what the indexer observed of a response.
type observed struct {
	ReceivedAt *time.Time `json:"receivedAt,omitempty"`

	// the labels that count on this response, by followed moderators
	Labels []label.Applied `json:"labels,omitempty"`

	// the claims that count for its author
	Claims []claim.Recognized `json:"claims,omitempty"`

	// its author represents the entity the review is about, by a domain
	// claim this indexer verified: the entity's own answer
	Official bool `json:"official,omitempty"`
}


// listing is the answer of GET /reviews/{eventId}/responses.
type listing struct {
	Items []respond.Item `json:"items"`
	Next  *string        `json:"next"`
}


// NewHandler returns the handler of the response routes.
func NewHandler(responses *Repository,
                reviews *review.Repository,
                moderation *label.Moderation,
                recognition *claim.Recognition) *Handler {
	return &Handler{
		responses:   responses,
		reviews:     reviews,
		moderation:  moderation,
		recognition: recognition,
	}
}


// ListByReview answers GET /reviews/{eventId}/responses: the responses of
// the chain any version of the review belongs to, oldest first, a page at a
// time.
func (h *Handler) ListByReview(w http.ResponseWriter, r *http.Request) {
	moderation, ok := h.moderation.ForRequest(w, r)
	if !ok {
		return
	}

	id := r.PathValue("eventId")

	if !ids.IsEventID(id) {
		respond.Error(w, http.StatusBadRequest, "invalid_event_id_error",
		              "An eventId is evt_ and 52 base32 characters.")
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

	rev, err := h.reviews.ByID(r.Context(), id)
	if errors.Is(err, review.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "review_not_found_error",
		              "No review with this id.")
		return
	}

	if err != nil {
		respond.Failed(w, err)
		return
	}

	// the review left the reading path, and its conversation with it
	if rev.Withdrawn {
		respond.Error(w, http.StatusGone, "review_withdrawn_error",
		              "This review was withdrawn by its author.")
		return
	}

	chainLabels, err := moderation.OnChains(r.Context(),
	                                          []string{rev.ChainRoot})
	if err != nil {
		respond.Failed(w, err)
		return
	}

	// a followed moderator removed the review: its conversation goes with it
	if label.AnyRemoves(chainLabels[rev.ChainRoot]) {
		respond.Removed(w)
		return
	}

	moderators, err := moderation.Moderators(r.Context())
	if err != nil {
		respond.Failed(w, err)
		return
	}

	hide := review.Hide{
		Moderators: moderators,
		Labels:     label.Hiding(show),
	}

	responses, next, err := h.responses.ByChain(r.Context(), rev.ChainRoot,
	                                            p, hide)
	if err != nil {
		respond.Failed(w, err)
		return
	}

	ids := make([]string, 0, len(responses))
	for _, res := range responses {
		ids = append(ids, res.EventID)
	}

	labels, err := moderation.OnResponses(r.Context(), ids)
	if err != nil {
		respond.Failed(w, err)
		return
	}

	authors := make([]string, 0, len(responses))
	for _, res := range responses {
		authors = append(authors, claim.AuthorOf(res.Envelope))
	}

	held, err := h.recognition.Of(r.Context(), authors)
	if err != nil {
		respond.Failed(w, err)
		return
	}

	entity, err := h.recognition.EntityOfChain(r.Context(), rev.ChainRoot)
	if err != nil {
		respond.Failed(w, err)
		return
	}

	reps, err := h.recognition.Representatives(r.Context(), []string{entity})
	if err != nil {
		respond.Failed(w, err)
		return
	}
	rep, hasRep := reps[entity]

	answer := listing{
		Items: make([]respond.Item, 0, len(responses)),
		Next:  page.Next(next),
	}

	for _, res := range responses {
		author := claim.AuthorOf(res.Envelope)
		answer.Items = append(answer.Items, respond.Item{
			Envelope: string(res.Envelope),
			Derived: derived{
				ChainRoot:        rev.ChainRoot,
				ReplyTo:          res.ReplyTo,
				RespondedVersion: res.RespondedVersion,
				TextChanged:      res.RespondedVersion != res.Head,
				Depth:            min(res.Depth, maxDepth),
			},
			Observed: observed{
				ReceivedAt: utc(res.ReceivedAt),
				Labels:     labels[res.EventID],
				Claims:     held[author],
				Official:   hasRep && rep.Identity == author,
			},
		})
	}

	respond.JSON(w, http.StatusOK, answer)
}


// utc returns t in UTC, or nil.
func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}

	u := t.UTC()

	return &u
}
