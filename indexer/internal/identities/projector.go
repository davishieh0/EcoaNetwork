package identities


import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/identities/sqlc"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/jackc/pgx/v5"
)


// --- GLOBALS ---

// ErrMalformed is returned for a rotation or revocation whose payload or
// createdAt cannot be read. The node validated it, so it should not happen.
var ErrMalformed = errors.New("rotation or revocation cannot be read")


// --- CODE ---

// Changed is what a rotation moved: every key whose identity or contest
// changed, and every identity, before and after, that one of them is part of.
type Changed struct {
	Keys       []string
	Identities []string
}


// ApplyRotated projects an identity.key_rotated: stored, and every identity
// worked out again from all the rotations, since one rotation can contest an
// identity or join two keys. Rotations are rare, so working them all out is
// cheap, and it makes the result the rule's whatever order they came in.
func ApplyRotated(ctx context.Context,
                  tx pgx.Tx,
                  env *envelope.Envelope) (Changed, error) {
	var p struct {
		Successor string `json:"successor"`
	}

	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return Changed{}, fmt.Errorf("payload: %w: %w", ErrMalformed, err)
	}

	q := sqlc.New(tx)

	err := q.SaveRotation(ctx, sqlc.SaveRotationParams{
		EventID:   env.EventID,
		Key:       env.Author,
		Successor: p.Successor,
		CreatedAt: env.CreatedAt,
	})
	if err != nil {
		return Changed{}, err
	}

	return reassign(ctx, q)
}


// ApplyRevoked projects an identity.key_revoked: stored and shown. Which
// events fall after it is each node's observation, so none is hidden.
func ApplyRevoked(ctx context.Context, tx pgx.Tx, env *envelope.Envelope) error {
	var p struct {
		Reason *string `json:"reason"`
	}

	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("payload: %w: %w", ErrMalformed, err)
	}

	createdAt, err := time.Parse(time.RFC3339, env.CreatedAt)
	if err != nil {
		return fmt.Errorf("createdAt: %w: %w", ErrMalformed, err)
	}

	return sqlc.New(tx).SaveRevocation(ctx, sqlc.SaveRevocationParams{
		EventID:   env.EventID,
		Key:       env.Author,
		Reason:    p.Reason,
		CreatedAt: createdAt,
	})
}


// reassign works every identity out again, writes it, and reports what
// changed from what was written before.
func reassign(ctx context.Context, q *sqlc.Queries) (Changed, error) {
	before, err := assignments(ctx, q)
	if err != nil {
		return Changed{}, err
	}

	rows, err := q.AllRotations(ctx)
	if err != nil {
		return Changed{}, err
	}

	rotations := make([]Rotation, 0, len(rows))
	for _, row := range rows {
		rotations = append(rotations, Rotation{
			EventID:   row.EventID,
			Key:       row.Key,
			Successor: row.Successor,
			CreatedAt: row.CreatedAt,
		})
	}

	after := Resolve(rotations)

	if err := q.ClearAssignments(ctx); err != nil {
		return Changed{}, err
	}

	for key, a := range after {
		err := q.SaveAssignment(ctx, sqlc.SaveAssignmentParams{
			Key:       key,
			Identity:  a.Identity,
			Position:  int32(a.Position),
			Contested: a.Contested,
		})
		if err != nil {
			return Changed{}, err
		}
	}

	return diff(before, after), nil
}


// assignments reads what was last written.
func assignments(ctx context.Context,
                 q *sqlc.Queries) (map[string]Assignment, error) {
	rows, err := q.AllAssignments(ctx)
	if err != nil {
		return nil, err
	}

	written := map[string]Assignment{}

	for _, row := range rows {
		written[row.Key] = Assignment{
			Identity:  row.Identity,
			Position:  int(row.Position),
			Contested: row.Contested,
		}
	}

	return written, nil
}


// diff returns the keys whose identity or contest differs, and the
// identities they were and are part of. An absent key is its own identity.
func diff(before, after map[string]Assignment) Changed {
	var changed Changed

	seen := map[string]bool{}
	identity := map[string]bool{}

	for _, assigned := range []map[string]Assignment{before, after} {
		for key := range assigned {
			if seen[key] {
				continue
			}

			seen[key] = true

			was, is := own(before, key), own(after, key)
			if was.Identity == is.Identity && was.Contested == is.Contested {
				continue
			}

			changed.Keys = append(changed.Keys, key)
			identity[was.Identity] = true
			identity[is.Identity] = true
		}
	}

	for id := range identity {
		changed.Identities = append(changed.Identities, id)
	}

	slices.Sort(changed.Keys)
	slices.Sort(changed.Identities)

	return changed
}


// own is key's assignment, itself when it has none.
func own(assigned map[string]Assignment, key string) Assignment {
	if a, ok := assigned[key]; ok {
		return a
	}

	return Assignment{Identity: key}
}
