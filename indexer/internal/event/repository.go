package event


import (
	"context"
	"errors"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/event/sqlc"
	"github.com/jackc/pgx/v5"
)


// --- GLOBALS ---

// ErrNotFound is returned for an eventId the indexer does not hold.
var ErrNotFound = errors.New("event not found")


// --- CODE ---

// Repository reads and writes the events table.
type Repository struct {
	queries *sqlc.Queries
}


// NewRepository returns a repository over db: the pool for a read, or the
// transaction the projector opened for a write.
func NewRepository(db sqlc.DBTX) *Repository {
	return &Repository{queries: sqlc.New(db)}
}


// ByID returns the event with eventId id, or ErrNotFound.
func (r *Repository) ByID(ctx context.Context, id string) (Event, error) {
	row, err := r.queries.EventByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Event{}, ErrNotFound
	}

	if err != nil {
		return Event{}, err
	}

	return Event{
		ID:         row.EventID,
		Envelope:   row.Envelope,
		ReceivedAt: row.ReceivedAt,
	}, nil
}


// Save stores e unless an event with its eventId is already stored, and
// reports whether it was new.
func (r *Repository) Save(ctx context.Context, e Signed) (bool, error) {
	stored, err := r.queries.SaveEvent(ctx, sqlc.SaveEventParams{
		EventID:   e.ID,
		EventType: e.Type,
		Author:    e.Author,
		CreatedAt: e.CreatedAt,
		Envelope:  e.Envelope,
	})

	return stored == 1, err
}


// Observe records that a node had event id at receivedAt, keeping the
// earliest time any node reported.
func (r *Repository) Observe(ctx context.Context,
                             id string,
                             receivedAt time.Time) error {
	return r.queries.ObserveEvent(ctx, sqlc.ObserveEventParams{
		EventID:    id,
		ReceivedAt: receivedAt,
	})
}


// Hold records that node holds event id, at sequence in its log, since
// receivedAt.
func (r *Repository) Hold(ctx context.Context, id string, h Holder) error {
	return r.queries.HoldEvent(ctx, sqlc.HoldEventParams{
		EventID:    id,
		Node:       h.Node,
		Sequence:   h.Sequence,
		ReceivedAt: h.ReceivedAt,
	})
}


// Holders returns the nodes that hold event id, the first to have it first.
func (r *Repository) Holders(ctx context.Context,
                             id string) ([]Holder, error) {
	rows, err := r.queries.HoldersOf(ctx, id)
	if err != nil {
		return nil, err
	}

	holders := make([]Holder, len(rows))
	for i, row := range rows {
		holders[i] = Holder{
			Node:       row.Node,
			Sequence:   row.Sequence,
			ReceivedAt: row.ReceivedAt,
		}
	}

	return holders, nil
}
