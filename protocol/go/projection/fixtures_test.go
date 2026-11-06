package projection


import (
	"encoding/json"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/fixturetest"
)


// --- CODE ---

func TestFixtures(t *testing.T) {
	names := []string{
		"vote-and-label-binding",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			// Arrange
			file := fixturetest.Load(t, name)

			// Act
			fixturetest.Run(t, file, nil)

			// Assert
			t.Fatal("a file without a rule did not skip")
		})
	}
}


func TestChainHeadFixtures(t *testing.T) {
	// Arrange
	file := fixturetest.Load(t, "chain-head")

	// Act
	fixturetest.Run(t, file, func(t *testing.T, c fixturetest.Case) {
		// Arrange
		events := eventsOf(c)

		// Act
		head := ResolveHead(rootOf(c), events)

		// Assert
		checkHead(t, c, head)
	})
}


func TestEntityMetadataFixtures(t *testing.T) {
	// Arrange
	file := fixturetest.Load(t, "entity-metadata")

	// Act
	fixturetest.Run(t, file, func(t *testing.T, c fixturetest.Case) {
		// Arrange
		proposals := unlabelled(eventsOf(c))
		identities := identitiesOf(t, c)

		// Act
		fields := ResolveFields(proposals, identities)

		// Assert
		checkFields(t, c, fields)
	})
}


// eventsOf turns the arrival of c into the events the rules read.
func eventsOf(c fixturetest.Case) []Event {
	events := make([]Event, 0, len(c.Arrival))

	for _, a := range c.Arrival {
		events = append(events, Event{
			ID:        a.ID,
			Author:    a.Author,
			Type:      a.Type,
			CreatedAt: a.At,
			Payload:   a.Payload,
		})
	}

	return events
}


// rootOf returns the id of the review the chain of c starts at.
func rootOf(c fixturetest.Case) string {
	for _, a := range c.Arrival {
		if a.Type == "review.created" {
			return a.ID
		}
	}

	return ""
}


// checkHead compares head with the head, truncated and withdrawn of c.
func checkHead(t *testing.T, c fixturetest.Case, head Head) {
	t.Helper()

	want, _ := c.Expect["head"].(string)
	truncated, _ := c.Expect["truncated"].(bool)
	withdrawn, _ := c.Expect["withdrawn"].(bool)

	if head.ID != want {
		t.Errorf("head = %q, want %q", head.ID, want)
	}

	if head.Truncated != truncated {
		t.Errorf("truncated = %v, want %v", head.Truncated, truncated)
	}

	if head.Withdrawn != withdrawn {
		t.Errorf("withdrawn = %v, want %v", head.Withdrawn, withdrawn)
	}
}


// identitiesOf decodes the identities that c supplies as input.
func identitiesOf(t *testing.T, c fixturetest.Case) map[string]Identity {
	t.Helper()

	identities := map[string]Identity{}

	for name, raw := range c.Identities {
		var identity Identity
		if err := json.Unmarshal(raw, &identity); err != nil {
			t.Fatalf("identity %s: %v", name, err)
		}

		identities[name] = identity
	}

	return identities
}


// unlabelled keeps the entity.updated events no label points at.
func unlabelled(events []Event) []Event {
	labelled := map[string]bool{}

	for _, e := range events {
		if e.Type == "moderation.label_created" {
			labelled[e.Field("target")] = true
		}
	}

	var kept []Event

	for _, e := range events {
		if e.Type == "entity.updated" && !labelled[e.ID] {
			kept = append(kept, e)
		}
	}

	return kept
}


// checkFields compares fields with the canonical values and tier of c.
func checkFields(t *testing.T, c fixturetest.Case, fields map[string]Field) {
	t.Helper()

	canonical, _ := c.Expect["canonical"].(map[string]any)
	tier, hasTier := c.Expect["tier"].(float64)

	if len(fields) != len(canonical) {
		t.Errorf("fields = %+v, want %d of them", fields, len(canonical))
	}

	for name, want := range canonical {
		got := fields[name]
		if got.Value != want {
			t.Errorf("%s = %q, want %q", name, got.Value, want)
		}

		if hasTier && float64(got.Tier) != tier {
			t.Errorf("%s tier = %d, want %v", name, got.Tier, tier)
		}
	}
}
