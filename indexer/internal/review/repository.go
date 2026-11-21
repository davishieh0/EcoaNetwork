package review


import (
	"context"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/review/sqlc"
)


// --- CODE ---

// Repository reads and writes the reviews and their chains.
type Repository struct {
	queries *sqlc.Queries
}


// NewRepository returns a repository over db: the pool for a read, or the
// transaction the projector opened for a write.
func NewRepository(db sqlc.DBTX) *Repository {
	return &Repository{queries: sqlc.New(db)}
}


// HeadsByEntity returns one page of an entity's listing, and the cursor of
// the next page, nil when this one is the last.
func (r *Repository) HeadsByEntity(ctx context.Context,
                                   entity string,
                                   page Page) ([]Listed, *Cursor, error) {
	params := sqlc.HeadsByEntityParams{
		Entity: entity,

		// one more than asked for tells whether a next page exists
		RowLimit: int32(page.Limit + 1),
	}

	if page.After != nil {
		params.AfterCreatedAt = &page.After.CreatedAt
		params.AfterRoot = &page.After.ChainRoot
	}

	rows, err := r.queries.HeadsByEntity(ctx, params)
	if err != nil {
		return nil, nil, err
	}

	var next *Cursor

	if len(rows) > page.Limit {
		rows = rows[:page.Limit]

		last := rows[len(rows)-1]
		next = &Cursor{CreatedAt: last.ChainCreatedAt, ChainRoot: last.ChainRoot}
	}

	listed := make([]Listed, 0, len(rows))

	for _, row := range rows {
		listed = append(listed, Listed{
			Version: Version{
				EventID:    row.Head,
				Envelope:   row.Envelope,
				ReceivedAt: row.ReceivedAt,
			},
			ChainRoot: row.ChainRoot,
			Edited:    row.Versions > 1,
		})
	}

	return listed, next, nil
}
