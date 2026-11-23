package event


import (
	"context"
	"errors"

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
