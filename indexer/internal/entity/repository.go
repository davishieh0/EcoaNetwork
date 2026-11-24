package entity


import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/entity/sqlc"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/page"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/projection"
	"github.com/jackc/pgx/v5"
)


// --- GLOBALS ---

// ErrNotFound is returned for an entity no event the indexer holds named.
var ErrNotFound = errors.New("entity not found")


// --- CODE ---

// Repository reads the entities and resolves their metadata.
type Repository struct {
	queries *sqlc.Queries
}


// NewRepository returns a repository over db.
func NewRepository(db sqlc.DBTX) *Repository {
	return &Repository{queries: sqlc.New(db)}
}


// ByID returns entity id with its metadata resolved from the proposals no
// moderator among moderators labeled, or ErrNotFound.
func (r *Repository) ByID(ctx context.Context,
                          id string,
                          moderators []string,
                          judge Judge) (Entity, error) {
	row, err := r.queries.EntityExists(ctx, id)
	if err != nil {
		return Entity{}, err
	}

	if !row.Found {
		return Entity{}, ErrNotFound
	}

	resolved, err := r.resolve(ctx, []string{id}, moderators, judge)
	if err != nil {
		return Entity{}, err
	}

	metadata := resolved[id]
	if judge != nil {
		metadata.Representative, err = judge.RepresentativeOf(ctx, id)
		if err != nil {
			return Entity{}, err
		}
	}

	metadata.Summary, err = r.summarize(ctx, id, moderators, judge)
	if err != nil {
		return Entity{}, err
	}

	e := Entity{
		Entity:   id,
		Derived:  derived{Proposals: row.Proposals},
		Observed: metadata,
	}

	if strings.HasPrefix(id, "entity:") {
		if err := r.describe(ctx, &e); err != nil {
			return Entity{}, err
		}
	}

	return e, nil
}


// Search returns one page of the entities with a proposal whose name holds
// text, newest first, every entity when text is empty, and the cursor of the
// next page.
func (r *Repository) Search(ctx context.Context,
                            text string,
                            moderators []string,
                            p page.Page,
                            category *string,
                            judge Judge) ([]Entity, *page.Cursor, error) {
	params := sqlc.SearchEntitiesParams{
		Query:      likePattern(fold(text)),
		Moderators: moderators,
		Category:   category,
		RowLimit:   int32(p.Limit + 1),
	}

	if p.After != nil {
		params.AfterSeen = &p.After.At
		params.AfterEntity = &p.After.ID
	}

	rows, err := r.queries.SearchEntities(ctx, params)
	if err != nil {
		return nil, nil, err
	}

	var next *page.Cursor

	if len(rows) > p.Limit {
		rows = rows[:p.Limit]

		last := rows[len(rows)-1]
		next = &page.Cursor{At: last.FirstSeen, ID: last.Entity}
	}

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.Entity)
	}

	resolved, err := r.resolve(ctx, ids, moderators, judge)
	if err != nil {
		return nil, nil, err
	}

	found := make([]Entity, 0, len(ids))
	for _, id := range ids {
		// a proposal said the category, but another holds: not this one
		held := resolved[id].Category
		if category != nil && (held == nil || held.Value != *category) {
			continue
		}

		found = append(found, Entity{Entity: id, Observed: resolved[id]})
	}

	return found, next, nil
}


// resolve works out each entity's metadata with the protocol's rule, from
// the proposals still in the contest: a verified representative's latest
// value first (tier 1), otherwise the first value (tier 2). Weights wait on
// reputation, so no challenger outweighs the first yet.
func (r *Repository) resolve(ctx context.Context,
                             ids, moderators []string,
                             judge Judge) (map[string]Metadata, error) {
	rows, err := r.queries.ProposalEnvelopes(ctx, sqlc.ProposalEnvelopesParams{
		Entities:   ids,
		Moderators: moderators,
	})
	if err != nil {
		return nil, err
	}

	proposals := map[string][]projection.Event{}

	for _, row := range rows {
		env, err := envelope.Decode(row.Envelope)
		if err != nil {
			return nil, fmt.Errorf("stored envelope: %w", err)
		}

		proposals[row.Entity] = append(proposals[row.Entity], projection.Event{
			ID:        env.EventID,
			Author:    env.Author,
			Type:      env.EventType,
			CreatedAt: env.CreatedAt,
			Payload:   env.Payload,
		})
	}

	identities := map[string]projection.Identity{}

	if judge != nil {
		var authors []string
		for _, list := range proposals {
			for _, p := range list {
				authors = append(authors, p.Author)
			}
		}

		identities, err = judge.Identities(ctx, authors)
		if err != nil {
			return nil, err
		}
	}

	resolved := map[string]Metadata{}

	for _, id := range ids {
		fields := projection.ResolveFields(proposals[id], identities)

		resolved[id] = Metadata{
			Name:        fieldOf(fields, "name"),
			Category:    fieldOf(fields, "category"),
			Description: fieldOf(fields, "description"),
		}
	}

	return resolved, nil
}


// describe adds an entity: identifier's descriptor, and whether its hash is
// the identifier.
func (r *Repository) describe(ctx context.Context, e *Entity) error {
	raw, err := r.queries.FirstDescriptor(ctx, e.Entity)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}

	if err != nil {
		return err
	}

	var d projection.Descriptor
	matches := false

	if json.Unmarshal([]byte(raw), &d) == nil {
		derivedID, err := projection.DeriveEntityID(d)
		matches = err == nil && derivedID == e.Entity
	}

	e.Derived.Descriptor = json.RawMessage(raw)
	e.Derived.DescriptorMatches = &matches

	return nil
}


// summarize works out id's summary of ratings, leaving out what a moderator
// among moderators hid and what an author with a claim on id wrote.
func (r *Repository) summarize(ctx context.Context,
                               id string,
                               moderators []string,
                               judge Judge) (*Summary, error) {
	rows, err := r.queries.HeadRatings(ctx, sqlc.HeadRatingsParams{
		Entity:     id,
		Moderators: moderators,
		Hiding:     label.Hiding(false),
	})
	if err != nil {
		return nil, err
	}

	conflicted := map[string]bool{}
	if judge != nil {
		authors := make([]string, 0, len(rows))
		for _, row := range rows {
			authors = append(authors, row.Author)
		}

		conflicted, err = judge.Conflicted(ctx, authors, id)
		if err != nil {
			return nil, err
		}
	}

	s := Summary{}
	total := 0

	for _, row := range rows {
		if conflicted[row.Author] {
			s.Conflicted++
			continue
		}

		if row.Rating < 1 || row.Rating > 5 {
			continue
		}

		s.Distribution[row.Rating-1]++
		s.Count++
		total += int(row.Rating)
	}

	if s.Count > 0 {
		// two decimals: a summary, not a measurement
		mean := math.Round(float64(total)/float64(s.Count)*100) / 100
		s.Mean = &mean
	}

	return &s, nil
}


// fieldOf is field name of fields, nil when no proposal sets it.
func fieldOf(fields map[string]projection.Field, name string) *Field {
	f, ok := fields[name]
	if !ok {
		return nil
	}

	return &Field{Value: f.Value, Proposal: f.Proposal, Tier: f.Tier}
}
