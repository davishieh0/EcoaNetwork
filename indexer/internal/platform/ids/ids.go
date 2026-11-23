// Package ids tells whether a string is a well-formed protocol identifier,
// before anything goes to the database with it.
package ids


import (
	"regexp"
)


// --- GLOBALS ---

// eventID is evt_ and the 52 base32 characters of a SHA-256.
var eventID = regexp.MustCompile(`^evt_[a-z2-7]{52}$`)


// entityID is every form protocol/schemas/common.schema.json allows.
var entityID = regexp.MustCompile(`^(` +
	`cnpj:[0-9]{14}` +
	`|gtin:[0-9]{14}` +
	`|domain:[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+` +
	`|place:osm:[nwr][1-9][0-9]*` +
	`|entity:[a-z2-7]{52}` +
	`)$`)


// --- CODE ---

// IsEventID reports whether s has the shape of an eventId.
func IsEventID(s string) bool {
	return eventID.MatchString(s)
}


// IsEntityID reports whether s has the shape of an entity identifier.
func IsEntityID(s string) bool {
	return entityID.MatchString(s)
}
