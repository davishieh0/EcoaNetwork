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


// Holder is a node that holds an event, as the indexer saw it.
type Holder struct {
	// Node is the node's address, as configured.
	Node string

	// Sequence is where the event sits in that node's log.
	Sequence int64

	// ReceivedAt is when that node had it, nil when it did not say.
	ReceivedAt *time.Time
}


// Signed is an event as it arrives from a node, to be stored.
type Signed struct {
	// ID is the eventId.
	ID string

	// Type is the eventType.
	Type string

	// Author is the did:key that signed it.
	Author string

	// CreatedAt is the time the author wrote it.
	CreatedAt time.Time

	// Envelope is the event byte for byte as it was signed.
	Envelope []byte
}
