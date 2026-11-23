package label


import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label/sqlc"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/jackc/pgx/v5"
)


// --- GLOBALS ---

// ErrMalformed is returned for a label whose payload cannot be read. The
// node validated it against the schema, so it should not happen.
var ErrMalformed = errors.New("label cannot be read")


// --- CODE ---

// ApplyCreated projects a moderation.label_created: stored against a review's
// whole chain when it names one of its versions, so an edit does not escape
// it, or against the response, the entity proposal or the profile it names.
// A label on anything else stays in events, and nothing more.
func ApplyCreated(ctx context.Context, tx pgx.Tx, env *envelope.Envelope) error {
	var p struct {
		Target string  `json:"target"`
		Label  string  `json:"label"`
		Note   *string `json:"note"`
	}

	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("payload: %w: %w", ErrMalformed, err)
	}

	q := sqlc.New(tx)

	params := sqlc.SaveLabelParams{
		EventID:   env.EventID,
		Target:    p.Target,
		Moderator: env.Author,
		Label:     p.Label,
		Note:      p.Note,
	}

	root, err := q.ChainOfVersion(ctx, p.Target)

	switch {
	case err == nil:
		params.ChainRoot = &root
	case !errors.Is(err, pgx.ErrNoRows):
		return err
	default:
		ok, err := placeOther(ctx, q, &params)
		if err != nil || !ok {
			return err
		}
	}

	return q.SaveLabel(ctx, params)
}


// placeOther places a label on a response, a proposal or a profile, and
// reports false for anything else.
func placeOther(ctx context.Context,
                q *sqlc.Queries,
                params *sqlc.SaveLabelParams) (bool, error) {
	response, err := q.IsResponse(ctx, params.Target)
	if err != nil {
		return false, err
	}

	if response {
		params.Response = &params.Target
		return true, nil
	}

	proposal, err := q.IsProposal(ctx, params.Target)
	if err != nil {
		return false, err
	}

	if proposal {
		params.Proposal = &params.Target
		return true, nil
	}

	profile, err := q.IsProfile(ctx, params.Target)
	if err != nil || !profile {
		return false, err
	}

	params.Profile = &params.Target

	return true, nil
}


// ApplyRevoked projects a moderation.label_revoked: the label it names stops
// counting. Only the moderator who published it can take it back; the node
// checked that, and the update checks it again.
func ApplyRevoked(ctx context.Context, tx pgx.Tx, env *envelope.Envelope) error {
	var p struct {
		Target string `json:"target"`
	}

	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("payload: %w: %w", ErrMalformed, err)
	}

	return sqlc.New(tx).RevokeLabel(ctx, sqlc.RevokeLabelParams{
		Label:     p.Target,
		Moderator: env.Author,
	})
}
