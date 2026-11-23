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

	// Endorsements and Disputes are the votes that count on this version:
	// one per voter, the latest, and never a vote on another version.
	Endorsements, Disputes int32
}


// Hide is what leaves a chain out of a listing: a label among Labels, by a
// moderator among Moderators.
type Hide struct {
	Moderators []string
	Labels     []string
}


// Listed is the head of a chain, as an entity's listing shows it.
type Listed struct {
	Version

	// ChainRoot is the original of the chain.
	ChainRoot string

	// Edited reports a chain with more than one version.
	Edited bool
}


// Review is one version with the state of its chain.
type Review struct {
	Version

	// ChainRoot is the original of the chain.
	ChainRoot string

	// Head is the chain's current version, "" when the chain is truncated.
	Head string

	// Withdrawn reports a withdrawal aimed at any version of the chain.
	Withdrawn bool

	// Truncated reports a chain longer than the walk to its head allows.
	Truncated bool
}

