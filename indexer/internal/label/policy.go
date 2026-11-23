// Package label is moderation: the labels anyone may publish, and which of
// them this indexer acts on, by whom it follows and what each label does.
package label


import (
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
)


// --- CODE ---

// Effect is what this indexer does with something a followed moderator
// labeled. The protocol says what something is, never what to do about it.
type Effect int


const (
	// Marked is shown, with its label.
	Marked Effect = iota + 1

	// Hidden is left out of every list unless a client asks to see it, and
	// then shown with its label.
	Hidden

	// Removed is left out of every answer, with no way to show it: the cases
	// a court order or an erasure request is about.
	Removed
)


// effects is what each label of the protocol's vocabulary does here.
var effects = map[string]Effect{
	"illegal":       Removed,
	"personal-data": Removed,
	"spam":          Hidden,
	"harassment":    Hidden,
	"impersonation": Hidden,
	"off-topic":     Marked,
	"other":         Marked,
}


// Policy is whose labels count. Every label is stored, whoever signed it, so
// changing who is followed rebuilds nothing.
type Policy struct {
	moderators []string
}


// file is the shape of the moderators file.
type file struct {
	Moderators []string `json:"moderators"`
}


// LoadPolicy reads the moderators file at path; "" follows nobody, so labels
// are stored and have no effect.
func LoadPolicy(path string) (Policy, error) {
	if path == "" {
		return Policy{}, nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("MODERATORS_FILE: %w", err)
	}

	var f file
	if err := json.Unmarshal(raw, &f); err != nil {
		return Policy{}, fmt.Errorf("MODERATORS_FILE %s: %w", path, err)
	}

	for _, did := range f.Moderators {
		if _, err := identity.Parse(did); err != nil {
			return Policy{}, fmt.Errorf("MODERATORS_FILE %s: %q is not a "+
			                            "did:key", path, did)
		}
	}

	return Follow(f.Moderators...), nil
}


// Follow returns a policy that follows moderators.
func Follow(moderators ...string) Policy {
	return Policy{moderators: slices.Clone(moderators)}
}


// Moderators returns whom this indexer follows, never nil.
func (p Policy) Moderators() []string {
	if p.moderators == nil {
		return []string{}
	}

	return slices.Clone(p.moderators)
}


// Hiding returns the labels that leave something out of a list: the removed
// ones always, the hidden ones unless show.
func Hiding(show bool) []string {
	return labelsWith(func(e Effect) bool {
		return e == Removed || (e == Hidden && !show)
	})
}


// Removing returns the labels that leave something out of every answer.
func Removing() []string {
	return labelsWith(func(e Effect) bool { return e == Removed })
}


// EffectOf returns what label does here.
func EffectOf(label string) Effect {
	return effects[label]
}


// labelsWith returns the labels whose effect matches, in a fixed order.
func labelsWith(match func(Effect) bool) []string {
	var labels []string

	for label, effect := range effects {
		if match(effect) {
			labels = append(labels, label)
		}
	}

	slices.Sort(labels)

	return labels
}
