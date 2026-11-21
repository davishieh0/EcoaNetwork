package review


import (
	"crypto/sha256"
	"encoding/base32"
	"strings"
)


// --- CODE ---

// evt returns a well-formed eventId named after name.
func evt(name string) string {
	digest := sha256.Sum256([]byte(name))
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).
		EncodeToString(digest[:])

	return "evt_" + strings.ToLower(encoded)
}
