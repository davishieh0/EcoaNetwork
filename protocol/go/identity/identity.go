// Package identity is the Ed25519 identity that signs an event, as did:key.
package identity


import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"filippo.io/edwards25519"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/mr-tron/base58"
)


// --- CONSTANTS ---

// prefix is the did:key method plus the multibase z of base58btc.
const prefix = "did:key:z"


// eventType of the two events that move an identity between keys.
const (
	rotatedType = "identity.key_rotated"
	revokedType = "identity.key_revoked"
)


// --- GLOBALS ---

// multicodec is the varint of an Ed25519 public key, 0xed01.
var multicodec = []byte{0xed, 0x01}


// ErrInvalid is a string that is not one canonical did:key for an Ed25519 key.
var ErrInvalid = errors.New("identity: invalid did:key")


// --- CODE ---

// Identity is an Ed25519 public key named by its did:key.
type Identity struct {
	// key is the 32-byte Ed25519 public key
	key ed25519.PublicKey
}


// FromKey returns the identity of an Ed25519 public key.
func FromKey(key ed25519.PublicKey) Identity {
	return Identity{key: bytes.Clone(key)}
}


// Parse returns the identity of a did:key, with one spelling per 32 bytes.
func Parse(s string) (Identity, error) {
	encoded, ok := strings.CutPrefix(s, prefix)
	if !ok {
		return Identity{}, fmt.Errorf("%w: not did:key with multibase z",
		                              ErrInvalid)
	}

	decoded, err := base58.Decode(encoded)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}

	key, ok := bytes.CutPrefix(decoded, multicodec)
	if !ok || len(key) != ed25519.PublicKeySize {
		return Identity{}, fmt.Errorf("%w: not an Ed25519 multicodec key",
		                              ErrInvalid)
	}

	if _, err := new(edwards25519.Point).SetBytes(key); err != nil {
		return Identity{}, fmt.Errorf("%w: not a curve point", ErrInvalid)
	}

	id := Identity{key: key}
	if id.String() != s {
		return Identity{}, fmt.Errorf("%w: not the canonical spelling",
		                              ErrInvalid)
	}

	return id, nil
}


// String returns the did:key of the identity.
func (i Identity) String() string {
	return prefix + base58.Encode(append(bytes.Clone(multicodec), i.key...))
}


// PublicKey returns the Ed25519 public key of the identity.
func (i Identity) PublicKey() ed25519.PublicKey {
	return bytes.Clone(i.key)
}


// Tracker is the rotations and revocations this node observed, in order.
type Tracker struct {
	// successor maps a rotated key to the key it named
	successor map[string]string

	// contested holds the keys with two conflicting rotations in one batch
	contested map[string]bool

	// revoked holds the keys whose revocation this node accepted
	revoked map[string]bool
}


// NewTracker returns a tracker that has observed nothing.
func NewTracker() *Tracker {
	return &Tracker{
		successor: map[string]string{},
		contested: map[string]bool{},
		revoked:   map[string]bool{},
	}
}


// Moves reports whether eventType rotates or revokes a key.
func Moves(eventType string) bool {
	return eventType == rotatedType || eventType == revokedType
}


// Observe records a rotation or revocation, refusing a conflicting rotation.
func (t *Tracker) Observe(env *envelope.Envelope) error {
	switch env.EventType {
	case rotatedType:
		return t.rotate(env)
	case revokedType:
		t.revoked[env.Author] = true
		return nil
	}

	return &UnsupportedEventError{EventType: env.EventType}
}


// ObserveBackfill records a batch, contesting a key with conflicting rotations.
func (t *Tracker) ObserveBackfill(rotations []*envelope.Envelope) error {
	named := map[string]map[string]bool{}

	for _, env := range rotations {
		successor, err := successorOf(env)
		if err != nil {
			return err
		}

		if named[env.Author] == nil {
			named[env.Author] = map[string]bool{}
		}

		named[env.Author][successor] = true
	}

	for author, successors := range named {
		_, rotated := t.successor[author]
		if len(successors) > 1 && !rotated {
			t.contested[author] = true
		}
	}

	return t.observeUncontested(rotations)
}


// Resolve returns the current key of key and whether its identity is contested.
func (t *Tracker) Resolve(key string) (string, bool) {
	for {
		next, rotated := t.successor[key]
		if !rotated {
			return key, t.contested[key]
		}

		key = next
	}
}


// IsRevoked reports whether this node accepted a revocation of key.
func (t *Tracker) IsRevoked(key string) bool {
	return t.revoked[key]
}


// rotate records the rotation of env, the first one observed per key.
func (t *Tracker) rotate(env *envelope.Envelope) error {
	successor, err := successorOf(env)
	if err != nil {
		return err
	}

	current, rotated := t.successor[env.Author]
	if rotated && current == successor {
		return nil
	}

	if rotated || t.contested[env.Author] {
		return &SuccessorObservedError{
			Author:    env.Author,
			Successor: successor,
		}
	}

	if t.reaches(successor, env.Author) {
		return &RotationCycleError{
			Author:    env.Author,
			Successor: successor,
		}
	}

	t.successor[env.Author] = successor
	return nil
}


// observeUncontested observes each rotation whose key is not contested.
func (t *Tracker) observeUncontested(rotations []*envelope.Envelope) error {
	var refused []error

	for _, env := range rotations {
		if t.contested[env.Author] {
			continue
		}

		if err := t.rotate(env); err != nil {
			refused = append(refused, err)
		}
	}

	return errors.Join(refused...)
}


// reaches reports whether the successors of from lead to target.
func (t *Tracker) reaches(from string, target string) bool {
	for key := from; ; key = t.successor[key] {
		if key == target {
			return true
		}

		if _, rotated := t.successor[key]; !rotated {
			return false
		}
	}
}


// successorOf returns the successor a rotation names.
func successorOf(env *envelope.Envelope) (string, error) {
	var payload struct {
		Successor string `json:"successor"`
	}

	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return "", fmt.Errorf("identity: rotation payload: %w", err)
	}

	return payload.Successor, nil
}
