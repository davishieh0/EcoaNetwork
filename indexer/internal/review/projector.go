package review


import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/entity"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/review/sqlc"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/projection"
	"github.com/jackc/pgx/v5"
)


// --- GLOBALS ---

// ErrUnplaced is returned for an edit or a withdrawal whose target is no
// version of a review the indexer holds. The node checked the reference, so
// it should not happen; the projector skips the event rather than stop.
var ErrUnplaced = errors.New("target is no review this indexer holds")


// ErrMalformed is returned for a review whose payload or createdAt cannot be
// read. The node validated it against the schema, so it should not happen.
var ErrMalformed = errors.New("review cannot be read")


// --- CODE ---

// payload is every field the three review events carry, each where it does.
type payload struct {
	Entity   string  `json:"entity"`
	Replaces string  `json:"replaces"`
	Target   string  `json:"target"`
	Rating   int16   `json:"rating"`
	Title    *string `json:"title"`
	Body     string  `json:"body"`
	Lang     *string `json:"lang"`
}


// ApplyCreated projects a review.created: a version, and the chain it opens.
func ApplyCreated(ctx context.Context, tx pgx.Tx, env *envelope.Envelope) error {
	p, createdAt, err := read(env)
	if err != nil {
		return err
	}

	// the first review that names an entity is enough for it to exist
	if err := entity.Ensure(ctx, tx, p.Entity, createdAt); err != nil {
		return err
	}

	q := sqlc.New(tx)

	err = q.SaveVersion(ctx, versionRow(env, p, env.EventID, p.Entity, createdAt))
	if err != nil {
		return err
	}

	return q.SaveChain(ctx, sqlc.SaveChainParams{
		ChainRoot: env.EventID,
		Entity:    p.Entity,
		CreatedAt: createdAt,
	})
}


// ApplyReplaced projects a review.replaced: a version in the chain of the one
// it edits, and the chain resolved again.
func ApplyReplaced(ctx context.Context, tx pgx.Tx, env *envelope.Envelope) error {
	p, createdAt, err := read(env)
	if err != nil {
		return err
	}

	q := sqlc.New(tx)

	place, err := placeOf(ctx, q, p.Replaces)
	if err != nil {
		return err
	}

	// the entity is the root's: an edit carries none
	v := versionRow(env, p, place.ChainRoot, place.Entity, createdAt)
	v.Replaces = &p.Replaces

	if err := q.SaveVersion(ctx, v); err != nil {
		return err
	}

	return resolve(ctx, q, place.ChainRoot)
}


// ApplyWithdrawn projects a review.withdrawn: the withdrawal, and the chain
// of the version it named resolved again.
func ApplyWithdrawn(ctx context.Context, tx pgx.Tx, env *envelope.Envelope) error {
	var p payload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("payload: %w: %w", ErrMalformed, err)
	}

	q := sqlc.New(tx)

	place, err := placeOf(ctx, q, p.Target)
	if err != nil {
		return err
	}

	err = q.SaveWithdrawal(ctx, sqlc.SaveWithdrawalParams{
		EventID:   env.EventID,
		ChainRoot: place.ChainRoot,
		Target:    p.Target,
	})
	if err != nil {
		return err
	}

	return resolve(ctx, q, place.ChainRoot)
}


// resolve works out the chain's head again from every event that decides it,
// with the protocol's rule, and records it.
func resolve(ctx context.Context, q *sqlc.Queries, root string) error {
	envelopes, err := q.ChainEnvelopes(ctx, root)
	if err != nil {
		return err
	}

	events := make([]projection.Event, 0, len(envelopes))

	for _, raw := range envelopes {
		env, err := envelope.Decode(raw)
		if err != nil {
			return fmt.Errorf("stored envelope: %w", err)
		}

		events = append(events, projection.Event{
			ID:        env.EventID,
			Author:    env.Author,
			Type:      env.EventType,
			CreatedAt: env.CreatedAt,
			Payload:   env.Payload,
		})
	}

	head := projection.ResolveHead(root, events)

	params := sqlc.UpdateChainParams{
		ChainRoot: root,
		Withdrawn: head.Withdrawn,
		Truncated: head.Truncated,
	}

	if head.ID != "" {
		params.Head = &head.ID
	}

	return q.UpdateChain(ctx, params)
}


// placeOf returns the chain of version id, or ErrUnplaced.
func placeOf(ctx context.Context,
             q *sqlc.Queries,
             id string) (sqlc.PlaceOfRow, error) {
	place, err := q.PlaceOf(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return place, fmt.Errorf("%s: %w", id, ErrUnplaced)
	}

	return place, err
}


// read decodes a version's payload and its createdAt.
func read(env *envelope.Envelope) (payload, time.Time, error) {
	var p payload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return p, time.Time{}, fmt.Errorf("payload: %w: %w", ErrMalformed, err)
	}

	createdAt, err := time.Parse(time.RFC3339, env.CreatedAt)
	if err != nil {
		return p, time.Time{}, fmt.Errorf("createdAt: %w: %w", ErrMalformed, err)
	}

	return p, createdAt, nil
}


// versionRow is the row of one version of a review.
func versionRow(env *envelope.Envelope,
             p payload,
             root, entity string,
             createdAt time.Time) sqlc.SaveVersionParams {
	return sqlc.SaveVersionParams{
		EventID:   env.EventID,
		ChainRoot: root,
		Author:    env.Author,
		Entity:    entity,
		Rating:    p.Rating,
		Title:     p.Title,
		Body:      p.Body,
		Lang:      p.Lang,
		CreatedAt: createdAt,
	}
}
