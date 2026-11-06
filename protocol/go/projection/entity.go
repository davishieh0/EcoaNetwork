package projection


import (
	"cmp"
	"slices"
)


// --- CONSTANTS ---

// incumbencyMargin is the factor a challenger must exceed the incumbent by.
const incumbencyMargin = 1.2


// --- GLOBALS ---

// fieldNames are the metadata fields an entity.updated can propose.
var fieldNames = []string{"name", "category", "description"}


// --- CODE ---

// Claim is a verification claim an identity holds, as the indexer judged it.
type Claim struct {
	Entity         string `json:"entity"`
	Recognized     bool   `json:"recognized"`
	SelfVerifiable bool   `json:"selfVerifiable"`
}


// Identity is what the indexer observed about an author: weight and claims.
type Identity struct {
	Weight float64 `json:"weight"`
	Claims []Claim `json:"claims"`
}


// Field is the canonical value of one metadata field.
type Field struct {
	// Value is the canonical text
	Value string
	// Proposal is the eventId of the entity.updated that set it
	Proposal string
	// Tier is 1 for a verified representative and 2 for a weighted proposal
	Tier int
}


// ResolveFields picks the canonical value of each field of one entity.
func ResolveFields(
	proposals []Event, identities map[string]Identity,
) map[string]Field {
	ordered := slices.Clone(proposals)
	slices.SortFunc(ordered, func(a, b Event) int {
		return cmp.Or(cmp.Compare(a.CreatedAt, b.CreatedAt),
		              cmp.Compare(a.ID, b.ID))
	})

	fields := map[string]Field{}

	for _, name := range fieldNames {
		proposing := asserting(name, ordered)

		if field, ok := authoritative(name, proposing, identities); ok {
			fields[name] = field
		} else if field, ok := incumbent(name, proposing, identities); ok {
			fields[name] = field
		}
	}

	return fields
}


// asserting keeps the proposals that say something about field name.
func asserting(name string, proposals []Event) []Event {
	var kept []Event

	for _, p := range proposals {
		if p.Field(name) != "" {
			kept = append(kept, p)
		}
	}

	return kept
}


// authoritative is the latest value a verified representative set for name.
func authoritative(
	name string, proposals []Event, identities map[string]Identity,
) (Field, bool) {
	var best *Event

	for _, p := range proposals {
		holder := representative(p.Field("entity"), identities)
		if holder == p.Author && (best == nil || p.Outranks(*best)) {
			best = &p
		}
	}

	if best == nil {
		return Field{}, false
	}

	return Field{Value: best.Field(name), Proposal: best.ID, Tier: 1}, true
}


// incumbent keeps the first value unless a challenger clears the margin.
func incumbent(
	name string, proposals []Event, identities map[string]Identity,
) (Field, bool) {
	if len(proposals) == 0 {
		return Field{}, false
	}

	held := proposals[0]

	for _, challenger := range proposals[1:] {
		heldWeight := identities[held.Author].Weight
		weight := identities[challenger.Author].Weight

		if weight > heldWeight*incumbencyMargin {
			held = challenger
		}
	}

	return Field{Value: held.Field(name), Proposal: held.ID, Tier: 2}, true
}


// representative is the one author with authority over entity, or "".
func representative(entity string, identities map[string]Identity) string {
	var holders, selfVerifiable []string

	for author, identity := range identities {
		for _, claim := range identity.Claims {
			if claim.Entity != entity || !claim.Recognized {
				continue
			}

			holders = append(holders, author)

			if claim.SelfVerifiable {
				selfVerifiable = append(selfVerifiable, author)
			}
		}
	}

	if len(holders) == 1 {
		return holders[0]
	}

	// self-verifiable beats delegated; a tie it cannot break is tier 2
	if len(selfVerifiable) == 1 {
		return selfVerifiable[0]
	}

	return ""
}
