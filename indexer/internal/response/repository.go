package response


import (
	"context"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/page"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/review"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/response/sqlc"
)


// --- CODE ---

// Repository reads and writes the responses.
type Repository struct {
	queries *sqlc.Queries
}


// NewRepository returns a repository over db: the pool for a read, or the
// transaction the projector opened for a write.
func NewRepository(db sqlc.DBTX) *Repository {
	return &Repository{queries: sqlc.New(db)}
}


// ByChain returns one page of a chain's responses, oldest first, and the
// cursor of the next page, nil when this one is the last.
func (r *Repository) ByChain(ctx context.Context,
                             root string,
                             p page.Page,
                             hide review.Hide) ([]Response, *page.Cursor, error) {
	params := sqlc.ResponsesByChainParams{
		ChainRoot:  root,
		Moderators: hide.Moderators,
		Hiding:     hide.Labels,

		// one more than asked for tells whether a next page exists
		RowLimit: int32(p.Limit + 1),
	}

	if p.After != nil {
		params.AfterSortAt = &p.After.At
		params.AfterID = &p.After.ID
	}

	rows, err := r.queries.ResponsesByChain(ctx, params)
	if err != nil {
		return nil, nil, err
	}

	var next *page.Cursor

	if len(rows) > p.Limit {
		rows = rows[:p.Limit]

		last := rows[len(rows)-1]
		next = &page.Cursor{At: last.SortAt, ID: last.EventID}
	}

	responses := make([]Response, 0, len(rows))

	for _, row := range rows {
		responses = append(responses, Response{
			EventID:          row.EventID,
			Envelope:         row.Envelope,
			ReceivedAt:       row.ReceivedAt,
			ReplyTo:          row.Target,
			RespondedVersion: row.RespondedVersion,
			Head:             row.Head,
			Depth:            row.Depth,
		})
	}

	return responses, next, nil
}
