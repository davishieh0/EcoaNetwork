// Package claim is verification: the claims anyone may publish, the DNS
// lookup that verifies a domain claim, and which claims count here.
package claim


import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/claim/sqlc"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/jackc/pgx/v5"
)


// --- GLOBALS ---

// ErrMalformed is returned for a claim whose payload cannot be read. The
// node validated it against the schema, so it should not happen.
var ErrMalformed = errors.New("claim cannot be read")


// --- CODE ---

// ApplyCreated projects a verification.claim_created: stored as signed,
// whoever signed it. Whether it counts is decided when the API answers.
func ApplyCreated(ctx context.Context, tx pgx.Tx, env *envelope.Envelope) error {
	var p struct {
		ClaimType   string  `json:"claimType"`
		Subject     string  `json:"subject"`
		Entity      *string `json:"entity"`
		EvidenceRef *string `json:"evidenceRef"`
		ExpiresAt   string  `json:"expiresAt"`
	}

	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("payload: %w: %w", ErrMalformed, err)
	}

	expires, err := time.Parse(time.RFC3339, p.ExpiresAt)
	if err != nil {
		return fmt.Errorf("expiresAt: %w: %w", ErrMalformed, err)
	}

	return sqlc.New(tx).SaveClaim(ctx, sqlc.SaveClaimParams{
		EventID:     env.EventID,
		ClaimType:   p.ClaimType,
		Issuer:      env.Author,
		Subject:     p.Subject,
		Entity:      p.Entity,
		EvidenceRef: p.EvidenceRef,
		ExpiresAt:   expires,
	})
}


// ApplyRevoked projects a verification.claim_revoked: the claim it names
// stops counting, only when its issuer signed the revocation.
func ApplyRevoked(ctx context.Context, tx pgx.Tx, env *envelope.Envelope) error {
	var p struct {
		Target string `json:"target"`
	}

	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("payload: %w: %w", ErrMalformed, err)
	}

	return sqlc.New(tx).RevokeClaim(ctx, sqlc.RevokeClaimParams{
		Claim:  p.Target,
		Issuer: env.Author,
	})
}
