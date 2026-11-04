package event


import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
)


// --- CONSTANTS ---

// canonicalCount counts the cases of protocol/vectors/canonicalization.json.
const canonicalCount = 11


// zip215Count is how many cases protocol/vectors/zip215.json holds.
const zip215Count = 208


// --- GLOBALS ---

var (
	// malformed tells whether an error is a MalformedJSONError
	malformed = is[*envelope.MalformedJSONError]

	// badID tells whether an error is an EventIDMismatchError
	badID = is[*EventIDMismatchError]

	// badSignature tells whether an error is an InvalidSignatureError
	badSignature = is[*InvalidSignatureError]

	// signedByDevice tells whether an error is a DeviceKeyNotSupportedError
	signedByDevice = is[*DeviceKeyNotSupportedError]
)


// message is what the signature tests sign.
var message = []byte("ecoa-event-v1\x00{}")


// --- CODE ---

// canonicalVector is one case of protocol/vectors/canonicalization.json.
type canonicalVector struct {
	Name         string          `json:"name"`
	Envelope     json.RawMessage `json:"envelope"`
	Canonical    string          `json:"canonical"`
	SigningInput string          `json:"signingInput"`
	EventID      string          `json:"eventId"`
	SigningKey   string          `json:"signingKey"`
	Signature    string          `json:"signature"`
}


// zip215Vector is one case of protocol/vectors/zip215.json.
type zip215Vector struct {
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
	Message   string `json:"message"`
	Signature string `json:"signature"`
	Expect    string `json:"expect"`
}


// is tells whether err is or wraps an error of type T.
func is[T error](err error) bool {
	var target T
	return errors.As(err, &target)
}


// newSigner returns a fixed key and its author string.
func newSigner(seed byte) (ed25519.PrivateKey, string) {
	key := ed25519.NewKeyFromSeed(append(make([]byte, 31), seed))
	public, _ := key.Public().(ed25519.PublicKey)

	return key, identity.FromKey(public).String()
}


// mustHex decodes hex or stops the test.
func mustHex(t *testing.T, text string) []byte {
	t.Helper()

	decoded, err := hex.DecodeString(text)
	if err != nil {
		t.Fatalf("hex %q: %v", text, err)
	}

	return decoded
}


func TestVerifySignatureMatchesEveryZIP215Vector(t *testing.T) {
	for _, v := range zip215Vectors(t) {
		t.Run(v.Name, func(t *testing.T) {
			// Arrange
			author := identity.FromKey(mustHex(t, v.PublicKey)).String()
			signed := mustHex(t, v.Message)
			signature := base64.RawURLEncoding.EncodeToString(
				mustHex(t, v.Signature),
			)

			if v.Expect != "accept" && v.Expect != "reject" {
				t.Fatalf("expect = %q", v.Expect)
			}

			// Act
			err := verifySignature(author, signature, signed)

			// Assert
			if accepted := err == nil; accepted != (v.Expect == "accept") {
				t.Fatalf("VerifySignature = %v, want %s", err, v.Expect)
			}
		})
	}
}


func TestVerifySignatureMatchesEveryCanonicalVector(t *testing.T) {
	for _, v := range canonicalVectors(t) {
		t.Run(v.Name, func(t *testing.T) {
			// Arrange
			env, err := envelope.Decode(v.Envelope)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}

			// Act
			err = verifySignature(env.Author,
			                      env.Signature,
			                      mustHex(t, v.SigningInput))

			// Assert
			if err != nil {
				t.Fatalf("VerifySignature: %v", err)
			}
		})
	}
}


func TestSigningKeyOfEveryVectorProducesItsAuthorAndSignature(t *testing.T) {
	for _, v := range canonicalVectors(t) {
		t.Run(v.Name, func(t *testing.T) {
			// Arrange
			key := ed25519.NewKeyFromSeed(mustHex(t, v.SigningKey))
			public, _ := key.Public().(ed25519.PublicKey)

			env, err := envelope.Decode(v.Envelope)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}

			// Act
			signature := ed25519.Sign(key, mustHex(t, v.SigningInput))

			// Assert
			if identity.FromKey(public).String() != env.Author {
				t.Fatalf("author = %s", env.Author)
			}

			encoded := base64.RawURLEncoding.EncodeToString(signature)
			if encoded != v.Signature {
				t.Fatalf("signature = %s, want %s", encoded, v.Signature)
			}
		})
	}
}


func TestVerifySignatureRefusesAnotherSpellingOfTheSameBytes(t *testing.T) {
	// Arrange
	key, author := newSigner(1)
	signature := ed25519.Sign(key, message)
	encoded := base64.RawURLEncoding.EncodeToString(signature)

	// sets one of the unused bits of the last digit
	respelled := encoded[:len(encoded)-1] + string(encoded[len(encoded)-1]+1)

	decoded, err := base64.RawURLEncoding.DecodeString(respelled)
	if err != nil || !bytes.Equal(decoded, signature) {
		t.Fatalf("lenient decoding = %x, %v, want the same bytes", decoded, err)
	}

	// Act
	err = verifySignature(author, respelled, message)

	// Assert
	if !badSignature(err) {
		t.Fatalf("VerifySignature = %v, want badSignature", err)
	}
}


func TestVerifySignatureRefusesALineBreak(t *testing.T) {
	// Arrange
	key, author := newSigner(1)
	encoded := base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, message))

	spellings := []string{
		encoded + "\n",
		encoded[:40] + "\r\n" + encoded[40:],
	}

	for _, spelling := range spellings {
		// Act
		err := verifySignature(author, spelling, message)

		// Assert
		if !badSignature(err) {
			t.Fatalf("verifySignature(%q) = %v", spelling, err)
		}
	}
}


func TestVerifySignatureRejects(t *testing.T) {
	key, author := newSigner(1)
	other, _ := newSigner(2)
	signature := ed25519.Sign(key, message)
	encoded := base64.RawURLEncoding.EncodeToString(signature)

	cases := []struct {
		name      string
		author    string
		signature string
	}{
		{
			"signed by another key",
			author,
			base64.RawURLEncoding.EncodeToString(ed25519.Sign(other, message)),
		},
		{"padded", author, encoded + "=="},
		{
			"63 bytes",
			author,
			base64.RawURLEncoding.EncodeToString(signature[:63]),
		},
		{
			"65 bytes",
			author,
			base64.RawURLEncoding.EncodeToString(append(signature, 0)),
		},
		{"bad author", "did:key:nope", encoded},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			input := message

			// Act
			err := verifySignature(c.author, c.signature, input)

			// Assert
			if !badSignature(err) {
				t.Fatalf("VerifySignature = %v, want badSignature", err)
			}
		})
	}
}


func TestVerifySignatureAcceptsItsOwnKey(t *testing.T) {
	// Arrange
	key, author := newSigner(1)
	signature := base64.RawURLEncoding.EncodeToString(
		ed25519.Sign(key, message),
	)

	// Act
	err := verifySignature(author, signature, message)

	// Assert
	if err != nil {
		t.Fatalf("VerifySignature: %v", err)
	}
}


func TestCheckSingleKey(t *testing.T) {
	// Arrange
	device := "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK"

	// Act
	single := checkSingleKey(&envelope.Envelope{DeviceKey: nil})
	filled := checkSingleKey(&envelope.Envelope{DeviceKey: &device})

	// Assert
	if single != nil {
		t.Fatalf("null deviceKey: %v", single)
	}

	if !signedByDevice(filled) {
		t.Fatalf("filled deviceKey = %v", filled)
	}
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


// canonicalVectors reads protocol/vectors/canonicalization.json.
func canonicalVectors(t *testing.T) []canonicalVector {
	t.Helper()

	path := "../../vectors/canonicalization.json"

	return loadCases[canonicalVector](t, path, canonicalCount)
}


// zip215Vectors reads protocol/vectors/zip215.json.
func zip215Vectors(t *testing.T) []zip215Vector {
	t.Helper()

	path := "../../vectors/zip215.json"

	return loadCases[zip215Vector](t, path, zip215Count)
}
