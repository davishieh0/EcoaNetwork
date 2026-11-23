package projector


import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/entity"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/event"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/response"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/review"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/stream"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/vote"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)


// --- CONSTANTS ---

// applyTimeout bounds one event's transaction, which runs to its end even
// when the projector is stopping, so a stop never leaves one half done.
const applyTimeout = 10 * time.Second


// --- GLOBALS ---

// subjects maps each event type to the subject that projects it: the one
// list of the types the indexer understands. A type missing here is stored
// in events and nothing more; a rebuild picks it up once it is added.
var subjects = map[string]func(context.Context, pgx.Tx, *envelope.Envelope) error{
	"review.created":   review.ApplyCreated,
	"review.replaced":  review.ApplyReplaced,
	"review.withdrawn": review.ApplyWithdrawn,
	"review.endorsed":  vote.Apply,
	"review.disputed":  vote.Apply,

	"review.response_created": response.Apply,

	"moderation.label_created": label.ApplyCreated,
	"moderation.label_revoked": label.ApplyRevoked,

	"entity.updated": entity.ApplyUpdated,
}


// skippable are the subjects' errors that mean the event has no place in the
// projection: trying it again would fail the same way.
var skippable = []error{
	review.ErrUnplaced,
	review.ErrMalformed,
	vote.ErrMalformed,
	response.ErrMalformed,
	label.ErrMalformed,
	entity.ErrMalformed,
}


// --- CODE ---

// applier writes each event into the database, one at a time.
type applier struct {
	pool *pgxpool.Pool
}


// apply stores r's event, its observation, what its subject makes of it and
// its node's cursor, in one transaction: a crash before the commit loses all
// of them, and the event comes again from the cursor that was saved.
func (a applier) apply(r stream.Received) error {
	ctx, cancel := context.WithTimeout(context.Background(), applyTimeout)
	defer cancel()

	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return err
	}

	// a no-op once the transaction is committed
	defer tx.Rollback(ctx)

	env, signed, err := read(r)
	if err != nil {
		// the node validated it, so this should not happen; skipping it keeps
		// the projector from stopping on it forever
		slog.Warn("skipped an event that cannot be read",
		          "node", r.Node,
		          "sequence", r.Event.GetSequence(),
		          "err", err)
	} else if err := projectOne(ctx, tx, env, signed, r); err != nil {
		return err
	}

	cursor := Cursor{LogID: r.Event.GetLogId(), After: r.Event.GetSequence()}
	if err := NewRepository(tx).Save(ctx, r.Node, cursor); err != nil {
		return err
	}

	return tx.Commit(ctx)
}


// projectOne stores the event once, when its node had it, and, the first time
// the indexer sees it, what its subject makes of it.
func projectOne(ctx context.Context,
                tx pgx.Tx,
                env *envelope.Envelope,
                signed event.Signed,
                r stream.Received) error {
	events := event.NewRepository(tx)

	isNew, err := events.Save(ctx, signed)
	if err != nil {
		return err
	}

	if received := r.Event.GetReceivedAt(); received != 0 {
		err := events.Observe(ctx, signed.ID, time.Unix(0, received))
		if err != nil {
			return err
		}
	}

	// a copy from another node, or the same event read again
	if !isNew {
		return nil
	}

	subject, known := subjects[env.EventType]
	if !known {
		return nil
	}

	err = subject(ctx, tx, env)
	if skipped(err) {
		slog.Warn("event has no place in the projection",
		          "node", r.Node,
		          "eventId", env.EventID,
		          "type", env.EventType,
		          "err", err)
		return nil
	}

	return err
}


// read decodes r's envelope and takes from it what the events table holds.
func read(r stream.Received) (*envelope.Envelope, event.Signed, error) {
	env, err := envelope.Decode(r.Event.GetEnvelope())
	if err != nil {
		return nil, event.Signed{}, err
	}

	createdAt, err := time.Parse(time.RFC3339, env.CreatedAt)
	if err != nil {
		return nil, event.Signed{},
		       fmt.Errorf("createdAt %q: %w", env.CreatedAt, err)
	}

	return env, event.Signed{
		ID:        env.EventID,
		Type:      env.EventType,
		Author:    env.Author,
		CreatedAt: createdAt,
		Envelope:  r.Event.GetEnvelope(),
	}, nil
}


// skipped reports an error that means the event has no place, not a failure.
func skipped(err error) bool {
	for _, target := range skippable {
		if errors.Is(err, target) {
			return true
		}
	}

	return false
}

