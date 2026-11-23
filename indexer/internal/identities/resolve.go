// Package identities follows identities through their keys: every
// identity.key_rotated and identity.key_revoked, which keys are one person,
// whether an identity is contested, and GET /identities/{did}.
package identities


import (
	"cmp"
	"slices"
)


// --- CODE ---

// Rotation is one identity.key_rotated: Key names Successor as its next key.
type Rotation struct {
	EventID   string
	Key       string
	Successor string

	// CreatedAt is the signed text, compared as the protocol compares it
	CreatedAt string
}


// Assignment is which identity a key belongs to.
type Assignment struct {
	// Identity is the first key of the chain, which names the identity.
	Identity string

	// Position is the key's place in its chain, 0 for the first.
	Position int

	// Contested reports an identity with two successors at some key.
	Contested bool
}


// Resolve works out, from every rotation any node accepted, which keys are
// one identity. A key that never rotated, and is no successor, is absent: it
// is its own identity.
//
// A node keeps the first rotation of a key it observes and refuses the rest;
// the indexer holds what every node accepted, so it can hold two. The rules:
//
//   - A rotation that would close a cycle is dropped, taking rotations in
//     the author's order (createdAt, then eventId).
//   - A key with one successor passes its identity on to it.
//   - A key with two different successors makes its identity contested, the
//     protocol's rule for a node that receives both at once: neither is
//     honored, and each successor starts an identity of its own.
//   - A key named as successor by two different keys joins neither: anyone
//     can name any key, so honoring one would let a stranger take a key, and
//     contesting would let a stranger contest anyone.
func Resolve(rotations []Rotation) map[string]Assignment {
	successors, predecessors := links(rotations)

	assigned := map[string]Assignment{}

	for _, root := range roots(successors, predecessors) {
		chain, contested := walk(root, successors, predecessors)

		for position, key := range chain {
			assigned[key] = Assignment{
				Identity:  root,
				Position:  position,
				Contested: contested,
			}
		}
	}

	return assigned
}


// links returns each key's distinct successors and predecessors, from the
// rotations that close no cycle.
func links(rotations []Rotation) (map[string][]string, map[string][]string) {
	ordered := slices.Clone(rotations)
	slices.SortFunc(ordered, func(a, b Rotation) int {
		return cmp.Or(cmp.Compare(a.CreatedAt, b.CreatedAt),
		              cmp.Compare(a.EventID, b.EventID))
	})

	successors := map[string][]string{}
	predecessors := map[string][]string{}

	for _, r := range ordered {
		if r.Key == r.Successor || reaches(r.Successor, r.Key, successors) {
			continue
		}

		if !slices.Contains(successors[r.Key], r.Successor) {
			successors[r.Key] = append(successors[r.Key], r.Successor)
		}

		if !slices.Contains(predecessors[r.Successor], r.Key) {
			predecessors[r.Successor] = append(predecessors[r.Successor], r.Key)
		}
	}

	return successors, predecessors
}


// reaches reports whether from leads to target along successors.
func reaches(from, target string, successors map[string][]string) bool {
	seen := map[string]bool{}
	pending := []string{from}

	for len(pending) > 0 {
		key := pending[0]
		pending = pending[1:]

		if key == target {
			return true
		}

		if seen[key] {
			continue
		}

		seen[key] = true
		pending = append(pending, successors[key]...)
	}

	return false
}


// roots returns the keys that start an identity, in a fixed order: those no
// key passes an identity on to.
func roots(successors, predecessors map[string][]string) []string {
	keys := map[string]bool{}

	for key, next := range successors {
		keys[key] = true

		for _, s := range next {
			keys[s] = true
		}
	}

	var found []string

	for key := range keys {
		if inherits(key, successors, predecessors) == "" {
			found = append(found, key)
		}
	}

	slices.Sort(found)

	return found
}


// inherits returns the key that passes its identity on to key, "" for none.
func inherits(key string, successors, predecessors map[string][]string) string {
	from := predecessors[key]
	if len(from) != 1 || len(successors[from[0]]) != 1 {
		return ""
	}

	return from[0]
}


// walk follows root's identity along its successors, and reports whether a
// key of it had two.
func walk(root string,
          successors, predecessors map[string][]string) ([]string, bool) {
	chain := []string{root}

	for key := root; ; {
		next := successors[key]

		if len(next) > 1 {
			return chain, true
		}

		if len(next) == 0 || inherits(next[0], successors, predecessors) != key {
			return chain, false
		}

		key = next[0]
		chain = append(chain, key)
	}
}
