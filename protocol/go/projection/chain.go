package projection


import (
	"slices"
)


// --- CONSTANTS ---

// maxChainLinks is the number of links a walk visits before it gives up.
const maxChainLinks = 64


// typeReplaced is the event type that adds a link to an edit chain.
const typeReplaced = "review.replaced"


// typeWithdrawn is the event type that retires a whole edit chain.
const typeWithdrawn = "review.withdrawn"


// --- CODE ---

// Head is the current version of an edit chain.
type Head struct {
	// ID is the eventId of the current version, empty when Truncated
	ID string
	// Truncated reports a chain longer than the walk allows
	Truncated bool
	// Withdrawn reports a withdrawal aimed at any link of the chain
	Withdrawn bool
}


// ResolveHead walks the chain that starts at root to its current version.
func ResolveHead(root string, events []Event) Head {
	next := successors(events)
	id, truncated := walk(root, next)
	members := chainIDs(root, next)

	return Head{
		ID:        id,
		Truncated: truncated,
		Withdrawn: withdrawnAny(members, events),
	}
}


// successors groups the replacements by the event they replace, winner first.
func successors(events []Event) map[string][]Event {
	next := map[string][]Event{}

	for _, e := range events {
		if e.Type == typeReplaced {
			replaced := e.Field("replaces")
			next[replaced] = append(next[replaced], e)
		}
	}

	for _, links := range next {
		slices.SortFunc(links, func(a, b Event) int {
			if a.Outranks(b) {
				return -1
			}

			return 1
		})
	}

	return next
}


// walk follows the winning successor from root, or reports a chain too long.
func walk(root string, next map[string][]Event) (string, bool) {
	head := root

	for visited := 1; len(next[head]) > 0; visited++ {
		if visited == maxChainLinks {
			return "", true
		}

		head = next[head][0].ID
	}

	return head, false
}


// chainIDs is root and every event that replaces a member, losers included.
func chainIDs(root string, next map[string][]Event) map[string]bool {
	members := map[string]bool{root: true}
	pending := []string{root}

	for len(pending) > 0 {
		id := pending[0]
		pending = pending[1:]

		for _, link := range next[id] {
			if !members[link.ID] {
				members[link.ID] = true
				pending = append(pending, link.ID)
			}
		}
	}

	return members
}


// withdrawnAny reports a withdrawal whose target is one of members.
func withdrawnAny(members map[string]bool, events []Event) bool {
	for _, e := range events {
		if e.Type == typeWithdrawn && members[e.Field("target")] {
			return true
		}
	}

	return false
}
