// Package event is the events table: every envelope the indexer holds, and
// GET /events/{eventId}.
package event


import (
	"time"
)


// --- CODE ---

// Event is one stored event, as signed, with what the indexer observed of it.
type Event struct {
	// ID is the eventId.
	ID string

	// Envelope is the event byte for byte as it was signed.
	Envelope []byte

	// ReceivedAt is when the network had it, nil when no node reported it.
	ReceivedAt *time.Time
}
