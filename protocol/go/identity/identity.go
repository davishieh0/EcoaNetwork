// Package identity is the Ed25519 identity that signs an event, as did:key.
package identity


import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"fmt"
	"strings"

	"filippo.io/edwards25519"
	"github.com/mr-tron/base58"
)


// --- CONSTANTS ---

// prefix is the did:key method plus the multibase z of base58btc.
const prefix = "did:key:z"


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
