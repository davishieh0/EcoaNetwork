// Package profile is the profiles: per identity, its latest profile.updated,
// and GET /profiles/{did} and GET /profiles?ids=.
package profile


import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/profile/sqlc"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/projection"
	"github.com/jackc/pgx/v5"
)


// --- GLOBALS ---

// ErrMalformed is returned for a profile whose payload or createdAt cannot be
// read. The node validated it against the schema, so it should not happen.
var ErrMalformed = errors.New("profile cannot be read")


// --- CODE ---

// ApplyUpdated projects a profile.updated: it becomes its identity's profile
// when it outranks the current one by the author's own ordering (the later
// createdAt, then the lower eventId), whatever order they arrived in, and
// whichever of the identity's keys signed it. A profile is total: what it
// leaves out is cleared.
func ApplyUpdated(ctx context.Context, tx pgx.Tx, env *envelope.Envelope) error {
	q := sqlc.New(tx)

	identity, err := q.IdentityOf(ctx, env.Author)
	if err != nil {
		return err
	}

	outranks, err := outranksCurrent(ctx, q, identity, env)
	if err != nil || !outranks {
		return err
	}

	return save(ctx, q, identity, env)
}


// save makes env identity's profile, replacing every field.
func save(ctx context.Context,
          q *sqlc.Queries,
          identity string,
          env *envelope.Envelope) error {
	var p struct {
		DisplayName *string `json:"displayName"`
		Bio         *string `json:"bio"`
		Lang        *string `json:"lang"`
	}

	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("payload: %w: %w", ErrMalformed, err)
	}

	createdAt, err := time.Parse(time.RFC3339, env.CreatedAt)
	if err != nil {
		return fmt.Errorf("createdAt: %w: %w", ErrMalformed, err)
	}

	return q.SaveProfile(ctx, sqlc.SaveProfileParams{
		Identity:    identity,
		EventID:     env.EventID,
		DisplayName: p.DisplayName,
		Bio:         p.Bio,
		Lang:        p.Lang,
		CreatedAt:   createdAt,
	})
}


// outranksCurrent reports whether env wins over identity's current profile
// under the protocol's rule, or the identity has none.
func outranksCurrent(ctx context.Context,
                     q *sqlc.Queries,
                     identity string,
                     env *envelope.Envelope) (bool, error) {
	raw, err := q.CurrentEnvelope(ctx, identity)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}

	if err != nil {
		return false, err
	}

	current, err := envelope.Decode(raw)
	if err != nil {
		return false, fmt.Errorf("stored envelope: %w", err)
	}

	return eventOf(env).Outranks(eventOf(current)), nil
}


// eventOf is env as the protocol's rules read it.
func eventOf(env *envelope.Envelope) projection.Event {
	return projection.Event{
		ID:        env.EventID,
		Author:    env.Author,
		Type:      env.EventType,
		CreatedAt: env.CreatedAt,
		Payload:   env.Payload,
	}
}
