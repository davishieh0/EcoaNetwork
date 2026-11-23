// Package vote is the votes on reviews: review.endorsed and review.disputed,
// each bound to the exact version it targeted.
package vote


import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/vote/sqlc"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/projection"
	"github.com/jackc/pgx/v5"
)


// --- GLOBALS ---

// ErrMalformed is returned for a vote whose payload or createdAt cannot be
// read. The node validated it against the schema, so it should not happen.
var ErrMalformed = errors.New("vote cannot be read")


// kinds maps a vote's event type to what it says.
var kinds = map[string]string{
	"review.endorsed": "endorsed",
	"review.disputed": "disputed",
}


// --- CODE ---

// Apply projects a vote: stored against the version it names, and that
// version's tally worked out again. A vote on anything but a review is valid
// and has no effect: it stays in events, and nothing more.
func Apply(ctx context.Context, tx pgx.Tx, env *envelope.Envelope) error {
	var p struct {
		Target string `json:"target"`
	}

	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("payload: %w: %w", ErrMalformed, err)
	}

	createdAt, err := time.Parse(time.RFC3339, env.CreatedAt)
	if err != nil {
		return fmt.Errorf("createdAt: %w: %w", ErrMalformed, err)
	}

	q := sqlc.New(tx)

	review, err := q.IsReview(ctx, p.Target)
	if err != nil || !review {
		return err
	}

	err = q.SaveVote(ctx, sqlc.SaveVoteParams{
		EventID:   env.EventID,
		Target:    p.Target,
		Voter:     env.Author,
		Kind:      kinds[env.EventType],
		CreatedAt: createdAt,
	})
	if err != nil {
		return err
	}

	return tally(ctx, q, p.Target)
}


// Retally works out again the tally of every version keys voted on: after a
// rotation, two of them may now be one voter.
func Retally(ctx context.Context, tx pgx.Tx, keys []string) error {
	q := sqlc.New(tx)

	versions, err := q.VotedBy(ctx, keys)
	if err != nil {
		return err
	}

	for _, version := range versions {
		if err := tally(ctx, q, version); err != nil {
			return err
		}
	}

	return nil
}


// tally works out again which votes count on version, one per voter under
// the protocol's rule, and records how many endorse and dispute it. A voter
// is an identity, not a key: votes cast before and after a rotation are one
// voter's.
func tally(ctx context.Context, q *sqlc.Queries, version string) error {
	envelopes, err := q.VoteEnvelopes(ctx, version)
	if err != nil {
		return err
	}

	votes := make([]projection.Event, 0, len(envelopes))
	voters := map[string]bool{}

	for _, row := range envelopes {
		env, err := envelope.Decode(row.Envelope)
		if err != nil {
			return fmt.Errorf("stored envelope: %w", err)
		}

		votes = append(votes, projection.Event{
			ID:        env.EventID,
			Author:    row.Identity,
			Type:      env.EventType,
			CreatedAt: env.CreatedAt,
			Payload:   env.Payload,
		})

		voters[row.Identity] = true
	}

	var endorsements, disputes int32

	for voter := range voters {
		current, found := projection.CurrentVote(voter, version, votes)
		if !found {
			continue
		}

		if current.Type == "review.endorsed" {
			endorsements++
		} else {
			disputes++
		}
	}

	return q.SaveTally(ctx, sqlc.SaveTallyParams{
		Version:      version,
		Endorsements: endorsements,
		Disputes:     disputes,
	})
}
