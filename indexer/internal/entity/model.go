// Package entity is the entities and their metadata: entity.updated, a
// proposal per event, resolved field by field when the API answers, and
// GET /entities/{entity} and GET /entities.
package entity


import (
	"context"
	"encoding/json"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/projection"
)


// --- CODE ---

// Field is the value that holds for one field, and the proposal that set it.
type Field struct {
	Value    string `json:"value"`
	Proposal string `json:"proposal"`

	// 1 for a verified representative, 2 for a weighted proposal
	Tier int `json:"tier"`
}


// Metadata is the value of each field, nil where no proposal sets it.
type Metadata struct {
	Name        *Field `json:"name"`
	Category    *Field `json:"category"`
	Description *Field `json:"description"`

	// Representative is the one identity with authority over the entity,
	// by a domain claim this indexer verified; absent when none.
	Representative *Representative `json:"representative,omitempty"`

	// Summary is this indexer's summary of the ratings, on GET
	// /entities/{entity} only. The protocol defines one rating, never how to
	// add them up: another indexer may summarize differently.
	Summary *Summary `json:"summary,omitempty"`
}


// Summary is the ratings of an entity's reviews that count: the head of each
// chain, leaving out withdrawn chains, chains a followed moderator hid or
// removed, and reviews by an identity holding a claim on the entity itself.
type Summary struct {
	// Count is how many reviews count.
	Count int `json:"count"`

	// Mean is their average rating, absent when none counts.
	Mean *float64 `json:"mean,omitempty"`

	// Distribution is how many count with each rating, 1 to 5.
	Distribution [5]int `json:"distribution"`

	// Conflicted is how many reviews were left out as a conflict of
	// interest: shown, marked, not counted.
	Conflicted int `json:"conflicted"`
}


// Representative is an entity's verified representative.
type Representative struct {
	Identity string     `json:"identity"`
	EventID  string     `json:"eventId"`
	Since    *time.Time `json:"since,omitempty"`
}


// Judge is what resolving an entity needs to know about the authors of its
// proposals: the claims that count for them, and who represents an entity.
type Judge interface {
	Identities(ctx context.Context,
	           authors []string) (map[string]projection.Identity, error)
	RepresentativeOf(ctx context.Context,
	                 entity string) (*Representative, error)
	Conflicted(ctx context.Context,
	           authors []string,
	           entity string) (map[string]bool, error)
}


// Entity is one entity as the API returns it.
type Entity struct {
	Entity   string   `json:"entity"`
	Derived  derived  `json:"derived"`
	Observed Metadata `json:"observed"`
}


// derived is what any honest indexer must agree on about an entity.
type derived struct {
	// Proposals counts every entity.updated for it, labeled or not.
	Proposals int32 `json:"proposals"`

	// Descriptor is what an entity: identifier is the hash of, and
	// DescriptorMatches whether the hash of it is the identifier; both
	// absent for other identifiers.
	Descriptor        json.RawMessage `json:"descriptor,omitempty"`
	DescriptorMatches *bool           `json:"descriptorMatches,omitempty"`
}
