package references


import (
	"errors"
	"slices"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/fixturetest"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
)


// --- CODE ---

// memory is a Lookup over events and tombstones held in a map.
type memory struct {
	authors  map[string]string
	tracker  *identity.Tracker
	asked    int
	failWith error
}


// newMemory returns a lookup that holds nothing.
func newMemory() *memory {
	return &memory{
		authors: map[string]string{},
		tracker: identity.NewTracker(),
	}
}


// Author returns the author held under eventID.
func (m *memory) Author(eventID string) (string, bool, error) {
	m.asked++

	author, found := m.authors[eventID]
	return author, found, m.failWith
}


// Resolve returns the current key of key.
func (m *memory) Resolve(key string) (string, bool) {
	return m.tracker.Resolve(key)
}


// hold records the tombstones a case declares.
func (m *memory) hold(c fixturetest.Case) {
	items, _ := c.Expect["tombstones"].([]any)

	for _, item := range items {
		entry, _ := item.(map[string]any)
		id, _ := entry["id"].(string)
		author, _ := entry["author"].(string)

		m.authors[fixturetest.EventID(id)] = fixturetest.Author(author)
	}
}


// observe records a rotation in the tracker, or stops the test.
func (m *memory) observe(t *testing.T, e fixturetest.Built) {
	t.Helper()

	if err := m.tracker.Observe(e.Env); err != nil {
		t.Fatalf("Observe %s: %v", e.ID, err)
	}
}


// check hands the arrivals of c to Check, learning each accepted event.
func (m *memory) check(
	t *testing.T, c fixturetest.Case,
) (accepted, rejected []string) {
	t.Helper()

	for _, e := range c.Events(t) {
		result, err := Check(e.Env, m)
		if err != nil {
			t.Fatalf("Check %s: %v", e.ID, err)
		}

		if result.Kind != Resolved {
			rejected = append(rejected, e.ID)
			continue
		}

		accepted = append(accepted, e.ID)
		m.authors[e.Env.EventID] = e.Env.Author

		if e.Env.EventType == "identity.key_rotated" {
			m.observe(t, e)
		}
	}

	return accepted, rejected
}


func TestFixtures(t *testing.T) {
	file := fixturetest.Load(t, "authorship")

	fixturetest.Run(t, file, func(t *testing.T, c fixturetest.Case) {
		// Arrange
		lookup := newMemory()
		lookup.hold(c)

		// Act
		accepted, rejected := lookup.check(t, c)

		// Assert
		if want := c.List("accepted"); !slices.Equal(accepted, want) {
			t.Fatalf("accepted = %v, want %v", accepted, want)
		}

		if want := c.Rejected(); !slices.Equal(rejected, want) {
			t.Fatalf("rejected = %v, want %v", rejected, want)
		}
	})
}


func TestCheckReportsATargetTheNodeLacksAsUnresolved(t *testing.T) {
	// Arrange
	file := fixturetest.Load(t, "orphan-buffer")
	events := file.Cases[0].Events(t)

	// Act
	result, err := Check(events[0].Env, newMemory())

	// Assert
	want := []string{fixturetest.EventID("e1")}
	if err != nil || result.Kind != Unresolved ||
	   !slices.Equal(result.Missing, want) {
		t.Fatalf("Check = %+v, %v, want unresolved %v", result, err, want)
	}
}


func TestCheckRejectsAnActionOnAnotherAuthorsEvent(t *testing.T) {
	// Arrange
	file := fixturetest.Load(t, "authorship")
	events := file.Cases[0].Events(t)

	lookup := newMemory()
	lookup.authors[events[0].Env.EventID] = events[0].Env.Author

	// Act
	result, err := Check(events[1].Env, lookup)

	// Assert
	var mismatch *AuthorMismatchError
	if err != nil || result.Kind != Rejected ||
	   !errors.As(result.Err, &mismatch) {
		t.Fatalf("Check = %+v, %v, want an author mismatch", result, err)
	}
}


func TestCheckSkipsAnUnknownEventTypeWithoutAskingTheLookup(t *testing.T) {
	// Arrange
	file := fixturetest.Load(t, "orphan-buffer")
	events := file.Cases[2].Events(t)
	lookup := newMemory()

	// Act
	result, err := Check(events[0].Env, lookup)

	// Assert
	if err != nil || result.Kind != Resolved || lookup.asked != 0 {
		t.Fatalf("Check = %+v, %v after %d lookups, want resolved with none",
		         result, err, lookup.asked)
	}
}


func TestCheckReturnsAFaultOfTheLookup(t *testing.T) {
	// Arrange
	file := fixturetest.Load(t, "orphan-buffer")
	events := file.Cases[0].Events(t)

	broken := errors.New("store closed")
	lookup := newMemory()
	lookup.failWith = broken

	// Act
	_, err := Check(events[0].Env, lookup)

	// Assert
	if !errors.Is(err, broken) {
		t.Fatalf("Check = %v, want %v", err, broken)
	}
}
