// Package entity is the entities and their metadata: entity.updated, a
// proposal per event, resolved field by field when the API answers, and
// GET /entities/{entity} and GET /entities.
package entity


import (
	"encoding/json"
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
