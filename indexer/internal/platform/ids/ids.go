// Package ids tells whether a string is a well-formed protocol identifier,
// before anything goes to the database with it.
package ids


import (
	"regexp"
)


// --- GLOBALS ---

// eventID is evt_ and the 52 base32 characters of a SHA-256.
var eventID = regexp.MustCompile(`^evt_[a-z2-7]{52}$`)


// --- CODE ---

// IsEventID reports whether s has the shape of an eventId.
func IsEventID(s string) bool {
	return eventID.MatchString(s)
}
