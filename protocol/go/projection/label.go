package projection


// --- CONSTANTS ---

// typeLabelCreated is the event type that puts a moderation label on a target.
const typeLabelCreated = "moderation.label_created"


// typeLabelRevoked is the event type that takes a label back.
const typeLabelRevoked = "moderation.label_revoked"


// typeResponseCreated is the event type of a reply to a review.
const typeResponseCreated = "review.response_created"


// --- CODE ---

// LabelsOnChain returns the unrevoked labels aimed at any event of chain.
func LabelsOnChain(chain, events []Event) []Event {
	revoked := map[string]bool{}

	for _, e := range events {
		if e.Type == typeLabelRevoked {
			revoked[e.Field("target")] = true
		}
	}

	var labels []Event

	for _, label := range aimedAt(chain, events, typeLabelCreated) {
		if !revoked[label.ID] {
			labels = append(labels, label)
		}
	}

	return labels
}


// ResponsesOnChain returns the responses aimed at any event of chain.
func ResponsesOnChain(chain, events []Event) []Event {
	return aimedAt(chain, events, typeResponseCreated)
}


// aimedAt returns the events of eventType whose target is in chain, by eventId.
func aimedAt(chain, events []Event, eventType string) []Event {
	members := map[string]bool{}

	for _, e := range chain {
		members[e.ID] = true
	}

	var found []Event

	for _, e := range events {
		if e.Type == eventType && members[e.Field("target")] {
			found = append(found, e)
		}
	}

	return sortedByID(found)
}
