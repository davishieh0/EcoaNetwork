// Package review is the reviews: their versions, the chain each belongs to,
// GET /entities/{entity}/reviews and GET /reviews/{eventId}.
package review


import (
	"time"
)


// --- CODE ---

// Version is one version of a review, as signed.
type Version struct {
	// EventID is the version's eventId.
	EventID string

	// Envelope is the version byte for byte as it was signed.
	Envelope []byte

	// ReceivedAt is when the network had it, nil when no node reported it.
	ReceivedAt *time.Time
}


// Listed is the head of a chain, as an entity's listing shows it.
type Listed struct {
	Version

	// ChainRoot is the original of the chain.
	ChainRoot string

	// Edited reports a chain with more than one version.
	Edited bool
}


// Page is where a listing starts and how much of it to return.
type Page struct {
	// After is the cursor the previous page ended on, nil for the first.
	After *Cursor

	// Limit is the most items to return.
	Limit int
}
