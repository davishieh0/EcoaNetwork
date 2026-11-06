// Package references resolves the events an event names and checks authorship.
package references


import (
	"encoding/json"
	"fmt"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
)


// --- CONSTANTS ---

const (
	// Resolved is an event whose references are all held and allowed.
	Resolved Kind = iota + 1

	// Unresolved is an event naming a target this node does not hold.
	Unresolved

	// Rejected is an event the identity of its author may not send.
	Rejected
)


// --- GLOBALS ---

// rules maps an eventType to the reference its payload carries.
var rules = map[string]rule{
	"review.replaced":            {field: "replaces", sameAuthor: true},
	"review.withdrawn":           {field: "target", sameAuthor: true},
	"moderation.label_revoked":   {field: "target", sameAuthor: true},
	"verification.claim_revoked": {field: "target", sameAuthor: true},
	"review.response_created":    {field: "target"},
	"review.endorsed":            {field: "target"},
	"review.disputed":            {field: "target"},
	"moderation.label_created":   {field: "target"},
	"moderation.report_created":  {field: "target"},
}


// --- CODE ---

// Kind is how far the references of an event resolve.
type Kind int


// Result is the verdict on the references of one event.
type Result struct {
	// Kind is the verdict.
	Kind Kind

	// Missing holds the eventIds this node lacks, when Kind is Unresolved.
	Missing []string

	// Err is the broken rule, when Kind is Rejected.
	Err error
}


// Lookup is what Check asks of the node.
type Lookup interface {
	// Author returns who signed the event, from the log or its tombstone.
	Author(eventID string) (author string, found bool, err error)

	// Resolve returns the current key of the identity of key.
	Resolve(key string) (identity string, contested bool)
}


// rule is the reference of one eventType.
type rule struct {
	// field is the payload field holding the eventId of the target
	field string

	// sameAuthor is true when the target must have the same author
	sameAuthor bool
}


// Check resolves the target of env and, where it acts on it, its authorship.
func Check(env *envelope.Envelope, l Lookup) (Result, error) {
	r, known := rules[env.EventType]
	if !known {
		return Result{Kind: Resolved}, nil
	}

	target, err := targetOf(env, r.field)
	if err != nil {
		return Result{}, err
	}

	author, found, err := l.Author(target)
	if err != nil {
		return Result{}, err
	}

	if !found {
		return Result{Kind: Unresolved, Missing: []string{target}}, nil
	}

	if r.sameAuthor && !sameIdentity(l, env.Author, author) {
		return Result{Kind: Rejected, Err: &AuthorMismatchError{
			EventType: env.EventType,
			Target:    target,
			Author:    env.Author,
			Owner:     author,
		}}, nil
	}

	return Result{Kind: Resolved}, nil
}


// sameIdentity reports whether two keys resolve to one identity.
func sameIdentity(l Lookup, a string, b string) bool {
	first, _ := l.Resolve(a)
	second, _ := l.Resolve(b)

	return first == second
}


// targetOf returns the eventId held by the payload field name.
func targetOf(env *envelope.Envelope, name string) (string, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return "", fmt.Errorf("references: payload: %w", err)
	}

	var target string
	if err := json.Unmarshal(payload[name], &target); err != nil {
		return "", fmt.Errorf("references: payload field %s: %w", name, err)
	}

	return target, nil
}
