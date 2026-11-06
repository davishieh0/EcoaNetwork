package projection


import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"strings"

	"github.com/gowebpki/jcs"
	"golang.org/x/text/unicode/norm"
)


// --- GLOBALS ---

// hashEncoding is base32 without padding; the hash is spelled lowercase.
var hashEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)


// --- CODE ---

// Descriptor is the fixed object an entity: identifier is derived from.
type Descriptor struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Country  string `json:"country"`
	Locality string `json:"locality,omitempty"`
}


// DeriveEntityID is entity: plus the base32 of the SHA-256 of the JCS form.
func DeriveEntityID(d Descriptor) (string, error) {
	normalized := Descriptor{
		Name:     normalize(d.Name),
		Kind:     normalize(d.Kind),
		Country:  normalize(d.Country),
		Locality: normalize(d.Locality),
	}

	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}

	canonical, err := jcs.Transform(raw)
	if err != nil {
		return "", err
	}

	digest := sha256.Sum256(canonical)
	hash := hashEncoding.EncodeToString(digest[:])

	return "entity:" + strings.ToLower(hash), nil
}


// normalize trims, lowercases, collapses whitespace and applies NFC to s.
func normalize(s string) string {
	collapsed := strings.Join(strings.Fields(s), " ")

	return norm.NFC.String(strings.ToLower(collapsed))
}
