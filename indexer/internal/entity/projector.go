package entity


import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/entity/sqlc"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/jackc/pgx/v5"
)


// --- GLOBALS ---

// ErrMalformed is returned for a proposal whose payload or createdAt cannot
// be read. The node validated it against the schema, so it should not happen.
var ErrMalformed = errors.New("entity proposal cannot be read")


// --- CODE ---

// Ensure records that an event written at seen named entity. An entity needs
// nothing else to exist.
func Ensure(ctx context.Context, tx pgx.Tx, entity string, seen time.Time) error {
	return sqlc.New(tx).EnsureEntity(ctx, sqlc.EnsureEntityParams{
		Entity:    entity,
		FirstSeen: seen,
	})
}


// ApplyUpdated projects an entity.updated: a proposal for the entity's
// metadata, stored as one. Which proposal sets each field is worked out when
// the API answers.
func ApplyUpdated(ctx context.Context, tx pgx.Tx, env *envelope.Envelope) error {
	var p struct {
		Entity      string          `json:"entity"`
		Name        *string         `json:"name"`
		Category    *string         `json:"category"`
		Description *string         `json:"description"`
		Descriptor  json.RawMessage `json:"descriptor"`
	}

	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("payload: %w: %w", ErrMalformed, err)
	}

	createdAt, err := time.Parse(time.RFC3339, env.CreatedAt)
	if err != nil {
		return fmt.Errorf("createdAt: %w: %w", ErrMalformed, err)
	}

	if err := Ensure(ctx, tx, p.Entity, createdAt); err != nil {
		return err
	}

	params := sqlc.SaveProposalParams{
		EventID:     env.EventID,
		Entity:      p.Entity,
		Author:      env.Author,
		CreatedAt:   createdAt,
		Name:        p.Name,
		Category:    p.Category,
		Description: p.Description,
	}

	if len(p.Descriptor) > 0 {
		params.Descriptor = p.Descriptor
	}

	if p.Name != nil {
		folded := fold(*p.Name)
		params.SearchName = &folded
	}

	return sqlc.New(tx).SaveProposal(ctx, params)
}
