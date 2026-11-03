// Package event turns an envelope into an event the protocol accepts.
package event


import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/gowebpki/jcs"
)


// --- CONSTANTS ---

// domainPrefix is the domain separator that opens every signing input.
const domainPrefix = "ecoa-event-v1\x00"


// --- GLOBALS ---

// idEncoding is base32 lowercase without padding.
var idEncoding = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567").
	WithPadding(base32.NoPadding)


// --- CODE ---

// SigningBytes is the JCS form of the envelope minus eventId and signature.
func SigningBytes(raw []byte) ([]byte, error) {
	// each value keeps its received bytes
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, &envelope.MalformedJSONError{Detail: err.Error()}
	}

	delete(fields, "eventId")
	delete(fields, "signature")

	stripped, err := json.Marshal(fields)
	if err != nil {
		return nil, &envelope.MalformedJSONError{Detail: err.Error()}
	}

	canonical, err := jcs.Transform(stripped)
	if err != nil {
		return nil, &envelope.MalformedJSONError{Detail: err.Error()}
	}

	return canonical, nil
}


// SigningInput prefixes the canonical bytes with the domain separator.
func SigningInput(canonical []byte) []byte {
	input := make([]byte, 0, len(domainPrefix)+len(canonical))
	input = append(input, domainPrefix...)

	return append(input, canonical...)
}


// ID is evt_ plus the base32 of the SHA-256 of the signing input.
func ID(signingInput []byte) string {
	digest := sha256.Sum256(signingInput)

	return "evt_" + idEncoding.EncodeToString(digest[:])
}


// checkID compares the received eventId with the recomputed one.
func checkID(env *envelope.Envelope, signingInput []byte) error {
	if computed := ID(signingInput); env.EventID != computed {
		return &EventIDMismatchError{Received: env.EventID, Computed: computed}
	}

	return nil
}
