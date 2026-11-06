// Package projection derives what a node shows from the events it holds.
package projection


import (
	"encoding/json"
)


// --- CODE ---

// Event is the part of a verified event that a projection rule reads.
type Event struct {
	ID        string
	Author    string
	Type      string
	CreatedAt string
	Payload   json.RawMessage
}


// Field returns the string payload field name, or "" when it is absent.
func (e Event) Field(name string) string {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(e.Payload, &fields); err != nil {
		return ""
	}

	var value string
	if err := json.Unmarshal(fields[name], &value); err != nil {
		return ""
	}

	return value
}


// Outranks reports whether e wins over other under the createdAt ordering.
func (e Event) Outranks(other Event) bool {
	// one spelling per instant, so comparing the text compares the time
	if e.CreatedAt != other.CreatedAt {
		return e.CreatedAt > other.CreatedAt
	}

	return e.ID < other.ID
}
