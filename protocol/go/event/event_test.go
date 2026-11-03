package event


import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
)


// --- CONSTANTS ---

// canonicalCount is how many cases vectors/canonicalization.json holds.
const canonicalCount = 11


// --- GLOBALS ---

var (
	// malformed tells whether an error is a MalformedJSONError
	malformed = is[*envelope.MalformedJSONError]

	// badID tells whether an error is an EventIDMismatchError
	badID = is[*EventIDMismatchError]
)


// --- CODE ---

// canonicalVector is one case of vectors/canonicalization.json.
type canonicalVector struct {
	Name         string          `json:"name"`
	Envelope     json.RawMessage `json:"envelope"`
	Canonical    string          `json:"canonical"`
	SigningInput string          `json:"signingInput"`
	EventID      string          `json:"eventId"`
	SigningKey   string          `json:"signingKey"`
	Signature    string          `json:"signature"`
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


func TestSigningMatchesEveryVector(t *testing.T) {
	for _, v := range canonicalVectors(t) {
		t.Run(v.Name, func(t *testing.T) {
			// Arrange
			raw := v.Envelope

			// Act
			canonical, err := SigningBytes(raw)

			// Assert
			if err != nil {
				t.Fatalf("SigningBytes: %v", err)
			}

			if string(canonical) != v.Canonical {
				t.Fatalf("canonical = %s, want %s", canonical, v.Canonical)
			}

			input := SigningInput(canonical)
			if hex.EncodeToString(input) != v.SigningInput {
				t.Fatalf("signing input = %x, want %s", input, v.SigningInput)
			}

			if id := ID(input); id != v.EventID {
				t.Fatalf("EventID = %s, want %s", id, v.EventID)
			}
		})
	}
}


func TestSigningBytesDropsAnEscapedExclusion(t *testing.T) {
	// Arrange
	raw := []byte(`{"\u0065ventId":"x","\u0073ignature":"y","a":1}`)

	// Act
	canonical, err := SigningBytes(raw)

	// Assert
	if err != nil {
		t.Fatalf("SigningBytes: %v", err)
	}

	if string(canonical) != `{"a":1}` {
		t.Fatalf("canonical = %s", canonical)
	}
}


func TestSigningBytesRejectsALoneSurrogate(t *testing.T) {
	// Arrange
	raw := []byte(`{"a":"\ud800"}`)

	// Act
	_, err := SigningBytes(raw)

	// Assert
	if !malformed(err) {
		t.Fatalf("SigningBytes = %v, want malformed", err)
	}
}


func TestSigningBytesKeepsTextOutOfNFC(t *testing.T) {
	// Arrange
	raw := []byte("{\"a\":\"Ótimo\"}")

	// Act
	canonical, err := SigningBytes(raw)

	// Assert
	if err != nil {
		t.Fatalf("SigningBytes: %v", err)
	}

	if !strings.Contains(string(canonical), "Ótimo") {
		t.Fatalf("canonical = %s, normalized the text", canonical)
	}
}


func TestSigningBytesRefusesWhatIsNotJSON(t *testing.T) {
	// Arrange
	raw := []byte("not json")

	// Act
	_, err := SigningBytes(raw)

	// Assert
	if !malformed(err) {
		t.Fatalf("SigningBytes = %v, want malformed", err)
	}
}


func TestCheckID(t *testing.T) {
	// Arrange
	input := SigningInput([]byte(`{"a":1}`))
	id := ID(input)
	swapped := "evt_" + strings.Repeat("a", 52)
	last := "a"
	if strings.HasSuffix(id, "a") {
		last = "b"
	}

	lastChanged := id[:len(id)-1] + last

	// Act
	same := checkID(&envelope.Envelope{EventID: id}, input)
	other := checkID(&envelope.Envelope{EventID: swapped}, input)
	almost := checkID(&envelope.Envelope{EventID: lastChanged}, input)

	// Assert
	if same != nil {
		t.Fatalf("same id: %v", same)
	}

	if !badID(other) {
		t.Fatalf("swapped id = %v, want badID", other)
	}

	if !badID(almost) {
		t.Fatalf("last character = %v, want badID", almost)
	}
}


// loadCases reads the want cases of the vector file at path into a slice.
func loadCases[T any](t *testing.T, path string, want int) []T {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var file struct {
		Cases []T `json:"cases"`
	}

	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	if len(file.Cases) != want {
		t.Fatalf("%s holds %d cases, want %d", path, len(file.Cases), want)
	}

	return file.Cases
}


// canonicalVectors reads vectors/canonicalization.json.
func canonicalVectors(t *testing.T) []canonicalVector {
	t.Helper()

	path := "../../../vectors/canonicalization.json"

	return loadCases[canonicalVector](t, path, canonicalCount)
}
