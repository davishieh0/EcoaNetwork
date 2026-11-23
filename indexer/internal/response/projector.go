package response


import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/response/sqlc"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/jackc/pgx/v5"
)


// --- GLOBALS ---

// ErrMalformed is returned for a response whose payload or createdAt cannot
// be read. The node validated it against the schema, so it should not happen.
var ErrMalformed = errors.New("response cannot be read")


// --- CODE ---

// Apply projects a response: stored in the chain of what it answers, a
// version of a review or another response. A response to anything else is
// valid and has no effect: it stays in events, and nothing more.
func Apply(ctx context.Context, tx pgx.Tx, env *envelope.Envelope) error {
	var p struct {
		Target string  `json:"target"`
		Body   string  `json:"body"`
		Lang   *string `json:"lang"`
	}

	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("payload: %w: %w", ErrMalformed, err)
	}

	createdAt, err := time.Parse(time.RFC3339, env.CreatedAt)
	if err != nil {
		return fmt.Errorf("createdAt: %w: %w", ErrMalformed, err)
	}

	q := sqlc.New(tx)

	// a target always comes before what answers it, so it is already here
	place, err := q.PlaceOfTarget(ctx, p.Target)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}

	if err != nil {
		return err
	}

	return q.SaveResponse(ctx, sqlc.SaveResponseParams{
		SortAt:           sortAt(createdAt, place.SortAt),
		EventID:          env.EventID,
		ChainRoot:        place.ChainRoot,
		Target:           p.Target,
		RespondedVersion: place.RespondedVersion,
		Depth:            place.Depth + 1,
		Author:           env.Author,
		Body:             p.Body,
		Lang:             p.Lang,
		CreatedAt:        createdAt,
	})
}


// sortAt is where a response written at createdAt sorts: never before what
// it answers, whose place is parent, nil for a version of the review.
func sortAt(createdAt time.Time, parent *time.Time) time.Time {
	if parent != nil && !createdAt.After(*parent) {
		return parent.Add(time.Microsecond)
	}

	return createdAt
}
