package event_test


import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/event"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/schema"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/testevent"
)


// --- CONSTANTS ---

// network is the network of every test event.
const network = "ecoa-dev"


// createdAt is the createdAt of every test event and the local clock.
const createdAt = "2026-08-05T17:00:00Z"


// review is a payload the review.created schema accepts.
const review = `{"entity":"cnpj:12345678000199","rating":5,"body":"ok"}`


// rejectionCount is how many cases protocol/vectors/rejections.json holds.
const rejectionCount = 15


// --- GLOBALS ---

// reasons maps the reason of a rejection vector to the error that says it.
var reasons = map[string]func(error) bool{
	"malformed-json":       malformed,
	"too-large":            tooLarge,
	"too-deep":             tooDeep,
	"number-out-of-domain": badNumber,
	"bad-signature":        badSignature,
	"bad-field":            badField,
	"unknown-field":        unknownField,
	"duplicate-key":        duplicateKey,
}


var (
	// tooLarge tells whether an error is a TooLargeError
	tooLarge = is[*envelope.TooLargeError]

	// malformed tells whether an error is a MalformedJSONError
	malformed = is[*envelope.MalformedJSONError]

	// duplicateKey tells whether an error is a DuplicateKeyError
	duplicateKey = is[*envelope.DuplicateKeyError]

	// tooDeep tells whether an error is a TooDeepError
	tooDeep = is[*envelope.TooDeepError]

	// badNumber tells whether an error is an UnsafeNumberError
	badNumber = is[*envelope.UnsafeNumberError]

	// unknownField tells whether an error is a FieldError of that fault
	unknownField = isField(envelope.FieldOutsideEnvelope)

	// badField tells whether an error is a FieldError of that fault
	badField = isField(envelope.FieldBreaksItsRule)

	// badID tells whether an error is an EventIDMismatchError
	badID = is[*event.EventIDMismatchError]

	// badSignature tells whether an error is an InvalidSignatureError
	badSignature = is[*event.InvalidSignatureError]

	// signedByDevice tells whether an error is a DeviceKeyNotSupportedError
	signedByDevice = is[*event.DeviceKeyNotSupportedError]

	// badPayload tells whether an error is an InvalidPayloadError
	badPayload = is[*schema.InvalidPayloadError]
)


// --- CODE ---

// rejectionFile is protocol/vectors/rejections.json.
type rejectionFile struct {
	Network string `json:"network"`
	Now     string `json:"now"`
	Cases   []struct {
		Name   string `json:"name"`
		Reason string `json:"reason"`
		Raw    string `json:"raw"`
	} `json:"cases"`
}


// clock returns a clock stopped at at.
func clock(t *testing.T, at string) func() time.Time {
	t.Helper()

	now, err := time.Parse(time.RFC3339, at)
	if err != nil {
		t.Fatalf("parse clock: %v", err)
	}

	return func() time.Time { return now }
}


// schemas returns the embedded payload schemas.
func schemas(t *testing.T) *schema.Set {
	t.Helper()

	set, err := schema.Load()
	if err != nil {
		t.Fatalf("schema.Load: %v", err)
	}

	return set
}


// receive opens raw and verifies the envelope.
func receive(
	t *testing.T, raw []byte, node envelope.Rules,
) (event.Event, error) {
	t.Helper()

	env, err := envelope.Open(raw, node)
	if err != nil {
		return event.Event{}, err
	}

	return event.Verify(env, schemas(t))
}


// rules returns the rules of a node of network at createdAt.
func rules(t *testing.T) envelope.Rules {
	t.Helper()

	return envelope.Rules{
		Network: network,
		MaxSize: envelope.MaxSize,
		Now:     clock(t, createdAt),
	}
}


// fields returns an event that passes every check.
func fields() testevent.Fields {
	return testevent.Fields{
		Protocol:  network,
		Version:   1,
		EventType: "review.created",
		CreatedAt: createdAt,
		Payload:   review,
	}
}


// sign signs f with a fixed key.
func sign(t *testing.T, f testevent.Fields) []byte {
	t.Helper()

	key := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))

	raw, err := testevent.Sign(key, f)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	return raw
}


// edit changes one field of an envelope and leaves the rest as signed.
func edit(t *testing.T, raw []byte, name, value string) []byte {
	t.Helper()

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("parse: %v", err)
	}

	parsed[name] = json.RawMessage(value)

	edited, err := json.Marshal(parsed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	return edited
}


func TestVerifyAcceptsEveryCanonicalizationVector(t *testing.T) {
	path := "../../vectors/canonicalization.json"

	file, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read vectors: %v", err)
	}

	var vectors struct {
		Cases []struct {
			Name     string          `json:"name"`
			Envelope json.RawMessage `json:"envelope"`
			EventID  string          `json:"eventId"`
		} `json:"cases"`
	}

	if err := json.Unmarshal(file, &vectors); err != nil {
		t.Fatalf("parse vectors: %v", err)
	}

	for _, v := range vectors.Cases {
		t.Run(v.Name, func(t *testing.T) {
			// Arrange
			raw := v.Envelope

			// Act
			got, err := receive(t, raw, rules(t))

			// Assert
			if err != nil {
				t.Fatalf("receive: %v", err)
			}

			if got.ID() != v.EventID || !bytes.Equal(got.Raw(), raw) {
				t.Fatalf("event = %s, %q", got.ID(), got.Raw())
			}
		})
	}
}


// loadRejections reads protocol/vectors/rejections.json.
func loadRejections(t *testing.T) rejectionFile {
	t.Helper()

	path := "../../vectors/rejections.json"

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read vectors: %v", err)
	}

	var file rejectionFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("parse vectors: %v", err)
	}

	if len(file.Cases) != rejectionCount {
		t.Fatalf("%d vectors, want %d", len(file.Cases), rejectionCount)
	}

	return file
}


func TestReceiveRefusesEveryRejectionVector(t *testing.T) {
	vectors := loadRejections(t)

	cfg := rules(t)
	cfg.Network = vectors.Network
	cfg.Now = clock(t, vectors.Now)

	for _, v := range vectors.Cases {
		t.Run(v.Name, func(t *testing.T) {
			// Arrange
			raw, err := hex.DecodeString(v.Raw)
			if err != nil {
				t.Fatalf("hex: %v", err)
			}

			want, ok := reasons[v.Reason]
			if !ok {
				t.Fatalf("reason %q is not known", v.Reason)
			}

			// Act
			_, err = receive(t, raw, cfg)

			// Assert
			if !matches(err, want) {
				t.Fatalf("receive = %v, want another error", err)
			}
		})
	}
}


func TestVerifyRejects(t *testing.T) {
	valid := sign(t, fields())

	deviceKey := "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK"
	withDevice := fields()
	withDevice.DeviceKey = &deviceKey

	badRating := fields()
	badRating.Payload = `{"entity":"cnpj:12345678000199","rating":9,"body":"x"}`

	cases := []struct {
		name string
		raw  []byte
		want func(error) bool
	}{
		{
			"payload schema",
			sign(t, badRating),
			badPayload,
		},
		{
			"tampered content",
			edit(t, valid, "payload",
			     `{"entity":"cnpj:12345678000199","rating":1,"body":"ok"}`),
			badID,
		},
		{
			"device key",
			sign(t, withDevice),
			signedByDevice,
		},
		{
			"device key with a bad signature",
			edit(t, sign(t, withDevice), "signature",
			     `"`+strings.Repeat("A", 86)+`"`),
			signedByDevice,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			cfg := rules(t)

			// Act
			_, err := receive(t, c.raw, cfg)

			// Assert
			if !matches(err, c.want) {
				t.Fatalf("receive = %v, want another error", err)
			}
		})
	}
}


func TestVerifyRejectsASignatureFromAnotherKey(t *testing.T) {
	// Arrange
	valid := sign(t, fields())

	other := ed25519.NewKeyFromSeed(append(make([]byte, 31), 1))
	forged, err := testevent.Sign(other, fields())
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(valid, &parsed); err != nil {
		t.Fatalf("parse: %v", err)
	}

	raw := edit(t, forged, "author", string(parsed["author"]))
	raw = edit(t, raw, "eventId", string(parsed["eventId"]))

	// Act
	_, err = receive(t, raw, rules(t))

	// Assert
	if !badSignature(err) {
		t.Fatalf("receive = %v, want another error", err)
	}
}


func TestVerifyAcceptsAnUnknownEventType(t *testing.T) {
	// Arrange
	opaque := fields()
	opaque.EventType = "future.thing"
	opaque.Payload = `{"anything":[1,2]}`

	// Act
	_, err := receive(t, sign(t, opaque), rules(t))

	// Assert
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
}


// is tells whether err is or wraps an error of type T.
func is[T error](err error) bool {
	var target T
	return errors.As(err, &target)
}


// matches tells whether err passes want, or is nil when want is nil.
func matches(err error, want func(error) bool) bool {
	if want == nil {
		return err == nil
	}

	return want(err)
}


// isField returns a check for a field error with fault.
func isField(fault envelope.FieldFault) func(error) bool {
	return func(err error) bool {
		var field *envelope.FieldError
		return errors.As(err, &field) && field.Fault == fault
	}
}
