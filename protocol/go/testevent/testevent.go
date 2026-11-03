// Package testevent builds validly signed events for tests.
package testevent


import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/event"
)


// --- CODE ---

// Fields is what an author decides about an event.
type Fields struct {
	Protocol  string
	Version   int
	DeviceKey *string
	EventType string
	CreatedAt string
	Payload   string
}


// Sign returns the envelope bytes of fields, signed by key.
func Sign(key ed25519.PrivateKey, fields Fields) ([]byte, error) {
	public, _ := key.Public().(ed25519.PublicKey)

	signed := map[string]any{
		"protocol":  fields.Protocol,
		"version":   fields.Version,
		"author":    identity.FromKey(public).String(),
		"deviceKey": fields.DeviceKey,
		"eventType": fields.EventType,
		"createdAt": fields.CreatedAt,
		"payload":   json.RawMessage(fields.Payload),
	}

	stripped, err := json.Marshal(signed)
	if err != nil {
		return nil, err
	}

	canonical, err := event.SigningBytes(stripped)
	if err != nil {
		return nil, err
	}

	input := event.SigningInput(canonical)
	signature := ed25519.Sign(key, input)

	signed["eventId"] = event.ID(input)
	signed["signature"] = base64.RawURLEncoding.EncodeToString(signature)

	return json.Marshal(signed)
}
