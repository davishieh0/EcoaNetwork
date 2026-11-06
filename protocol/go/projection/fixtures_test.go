package projection


import (
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/fixturetest"
)


// --- CODE ---

func TestFixtures(t *testing.T) {
	names := []string{
		"vote-and-label-binding",
		"entity-metadata",
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
