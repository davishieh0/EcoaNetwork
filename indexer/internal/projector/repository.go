package projector


import (
	"context"
	"errors"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/projector/sqlc"
	"github.com/jackc/pgx/v5"
)


// --- CODE ---

// Cursor is where the projector stands in one node's log.
type Cursor struct {
	// LogID is the log After counts in, "" before the first event.
	LogID string

	// After is the last sequence applied, 0 before the first event.
	After uint64
}


// Repository reads and writes the projector's cursors.
type Repository struct {
	queries *sqlc.Queries
}


// NewRepository returns a repository over db: the pool for a read, or the
// transaction of the event a cursor moves with.
func NewRepository(db sqlc.DBTX) *Repository {
	return &Repository{queries: sqlc.New(db)}
}


// CursorOf returns where the projector stands in node's log, the zero Cursor
// for a node it never read.
func (r *Repository) CursorOf(ctx context.Context,
                              node string) (Cursor, error) {
	row, err := r.queries.CursorOf(ctx, node)
	if errors.Is(err, pgx.ErrNoRows) {
		return Cursor{}, nil
	}

	if err != nil {
		return Cursor{}, err
	}

	return Cursor{LogID: row.LogID, After: uint64(row.Sequence)}, nil
}


// Save moves node's cursor to c.
func (r *Repository) Save(ctx context.Context, node string, c Cursor) error {
	return r.queries.SaveCursor(ctx, sqlc.SaveCursorParams{
		Node:     node,
		LogID:    c.LogID,
		Sequence: int64(c.After),
	})
}
