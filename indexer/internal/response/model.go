// Package response is the responses to reviews: review.response_created,
// attached to the chain of the review they answer, and
// GET /reviews/{eventId}/responses.
package response


import (
	"time"
)


// --- CODE ---

// Response is one response, as signed, with its place in the conversation.
type Response struct {
	// EventID is the response's eventId.
	EventID string

	// Envelope is the response byte for byte as it was signed.
	Envelope []byte

	// ReceivedAt is when the network had it, nil when no node reported it.
	ReceivedAt *time.Time

	// ReplyTo is what it answered: a version of the review, or a response.
	ReplyTo string

	// RespondedVersion is the version of the review the conversation answered.
	RespondedVersion string

	// Head is the chain's current version, "" when the chain is truncated.
	Head string

	// Depth is 1 for an answer to the review, 2 for an answer to that, ...
	Depth int32
}
