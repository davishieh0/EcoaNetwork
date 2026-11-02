package identity


import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"testing"

	"github.com/mr-tron/base58"
)


// --- CONSTANTS ---

// known is the did:key of the seed of 32 zero bytes, from an outside encoder.
const known = "did:key:z6MkiTBz1ymuepAQ4HEHYSF1H8quG5GLVVQR3djdX3mDooWp"


// --- CODE ---

// keyBytes returns a valid public key.
func keyBytes() ed25519.PublicKey {
	private := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	public, _ := private.Public().(ed25519.PublicKey)

	return public
}


func TestParseReturnsTheKey(t *testing.T) {
	// Arrange
	want := keyBytes()

	// Act
	got, err := Parse(known)

	// Assert
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if !bytes.Equal(got.PublicKey(), want) {
		t.Fatalf("Parse = %x, want %x", got.PublicKey(), want)
	}
}


func TestFromKeyThenParseIsTheSameKey(t *testing.T) {
	// Arrange
	key := keyBytes()

	// Act
	got, err := Parse(FromKey(key).String())

	// Assert
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if !bytes.Equal(got.PublicKey(), key) {
		t.Fatalf("round trip = %x, want %x", got, key)
	}
}


func TestParseRejects(t *testing.T) {
	// y = 2 is not on the curve
	notAPoint := append(make([]byte, 0, 32), 2)
	notAPoint = append(notAPoint, make([]byte, 31)...)

	multicodec := append([]byte{0xed, 0x01}, keyBytes()...)
	otherCodec := append([]byte{0xec, 0x01}, keyBytes()...)
	offCurve := append([]byte{0xed, 0x01}, notAPoint...)
	shortKey := []byte{0xed, 0x01, 1, 2, 3}

	cases := []struct {
		name string
		text string
	}{
		{"empty", ""},
		{"another method", "did:web:example.com"},
		{"another multibase", "did:key:m" + base58.Encode(multicodec)},
		{"method in capitals", "DID:KEY:z" + base58.Encode(multicodec)},
		{"not base58", "did:key:z0OIl"},
		{"another multicodec", "did:key:z" + base58.Encode(otherCodec)},
		{"short key", "did:key:z" + base58.Encode(shortKey)},
		{"leading base58 zero", "did:key:z1" + base58.Encode(multicodec)},
		{"not a curve point", "did:key:z" + base58.Encode(offCurve)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			text := c.text

			// Act
			_, err := Parse(text)

			// Assert
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("Parse(%q) = %v, want another error", text, err)
			}
		})
	}
}
