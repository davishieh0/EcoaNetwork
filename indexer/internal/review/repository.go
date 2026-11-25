package review


import (
	"context"
	"errors"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/page"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/review/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)


// --- GLOBALS ---

// ErrNotFound is returned for an eventId that is no version of a review.
var ErrNotFound = errors.New("review not found")


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
                                   p page.Page,
                                   hide Hide) ([]Listed, *page.Cursor, error) {
	params := sqlc.HeadsByEntityParams{
		Entity:     entity,
		Moderators: hide.Moderators,
		Hiding:     hide.Labels,

		// one more than asked for tells whether a next page exists
		RowLimit: int32(p.Limit + 1),
	}

	if p.After != nil {
		params.AfterCreatedAt = &p.After.At
		params.AfterRoot = &p.After.ID
	}

	rows, err := r.queries.HeadsByEntity(ctx, params)
	if err != nil {
		return nil, nil, err
	}

	var next *page.Cursor

	if len(rows) > p.Limit {
		rows = rows[:p.Limit]

		last := rows[len(rows)-1]
		next = &page.Cursor{At: last.ChainCreatedAt, ID: last.ChainRoot}
	}

	listed := make([]Listed, 0, len(rows))

	for _, row := range rows {
		listed = append(listed, Listed{
			Version: Version{
				EventID:      row.Head,
				Envelope:     row.Envelope,
				ReceivedAt:   row.ReceivedAt,
				Endorsements: row.Endorsements,
				Disputes:     row.Disputes,
			},
			ChainRoot: row.ChainRoot,
			Edited:    row.Versions > 1,
		})
	}

	return listed, next, nil
}


// HeadsByScore returns one page of the heads of entity's visible chains,
// ranked by the head's score, then newest original first, and the cursor of
// the next page.
func (r *Repository) HeadsByScore(ctx context.Context,
                                  entity string,
                                  p page.Page,
                                  hide Hide) ([]Listed, *page.Cursor, error) {
	params := sqlc.HeadsByScoreParams{
		Entity:     entity,
		Moderators: hide.Moderators,
		Hiding:     hide.Labels,
		RowLimit:   int32(p.Limit + 1),
	}

	// a cursor from the time-ordered list has no score: start over
	if p.After != nil && p.After.Score != nil {
		params.AfterScore = pgtype.Int8{Int64: *p.After.Score, Valid: true}
		params.AfterCreatedAt = &p.After.At
		params.AfterRoot = &p.After.ID
	}

	rows, err := r.queries.HeadsByScore(ctx, params)
	if err != nil {
		return nil, nil, err
	}

	var next *page.Cursor

	if len(rows) > p.Limit {
		rows = rows[:p.Limit]

		last := rows[len(rows)-1]
		score := last.Score
		next = &page.Cursor{Score: &score, At: last.ChainCreatedAt,
		                    ID: last.ChainRoot}
	}

	listed := make([]Listed, 0, len(rows))

	for _, row := range rows {
		listed = append(listed, Listed{
			Version: Version{
				EventID:      row.Head,
				Envelope:     row.Envelope,
				ReceivedAt:   row.ReceivedAt,
				Endorsements: row.Endorsements,
				Disputes:     row.Disputes,
			},
			ChainRoot: row.ChainRoot,
			Edited:    row.Versions > 1,
		})
	}

	return listed, next, nil
}


// ByID returns the version with eventId id and its chain's state, or
// ErrNotFound.
func (r *Repository) ByID(ctx context.Context, id string) (Review, error) {
	row, err := r.queries.ReviewByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Review{}, ErrNotFound
	}

	if err != nil {
		return Review{}, err
	}

	review := Review{
		Version: Version{
			EventID:      row.EventID,
			Envelope:     row.Envelope,
			ReceivedAt:   row.ReceivedAt,
			Endorsements: row.Endorsements,
			Disputes:     row.Disputes,
		},
		ChainRoot: row.ChainRoot,
		Withdrawn: row.Withdrawn,
		Truncated: row.Truncated,
	}

	if row.Head != nil {
		review.Head = *row.Head
	}

	return review, nil
}


// Versions returns every version of the chain that starts at root, forks
// included, oldest first.
func (r *Repository) Versions(ctx context.Context,
                              root string) ([]Version, error) {
	rows, err := r.queries.ChainVersions(ctx, root)
	if err != nil {
		return nil, err
	}

	versions := make([]Version, 0, len(rows))

	for _, row := range rows {
		versions = append(versions, Version{
			EventID:      row.EventID,
			Envelope:     row.Envelope,
			ReceivedAt:   row.ReceivedAt,
			Endorsements: row.Endorsements,
			Disputes:     row.Disputes,
		})
	}

	return versions, nil
}
