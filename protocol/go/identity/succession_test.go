package identity_test


import (
	"errors"
	"slices"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/fixturetest"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
)


// --- GLOBALS ---

// errRevoked is the verdict on an event signed by a revoked key.
var errRevoked = errors.New("signing key revoked")


// --- CODE ---

// replay is what a tracker concluded after the arrivals of one case.
type replay struct {
	tracker  *identity.Tracker
	accepted []string
	rejected []string
}


// judge refuses a revoked key, else observes a rotation or a revocation.
func judge(tracker *identity.Tracker, env *envelope.Envelope) error {
	if tracker.IsRevoked(env.Author) {
		return errRevoked
	}

	if !identity.Moves(env.EventType) {
		return nil
	}

	return tracker.Observe(env)
}


// run hands the arrivals of c to a tracker, one by one.
func run(t *testing.T, c fixturetest.Case) replay {
	t.Helper()

	r := replay{tracker: identity.NewTracker()}

	for _, e := range c.Events(t) {
		if judge(r.tracker, e.Env) != nil {
			r.rejected = append(r.rejected, e.ID)
			continue
		}

		r.accepted = append(r.accepted, e.ID)
	}

	return r
}


// runBackfill hands the arrivals of c to a tracker as one batch.
func runBackfill(t *testing.T, c fixturetest.Case) *identity.Tracker {
	t.Helper()

	tracker := identity.NewTracker()

	var batch []*envelope.Envelope
	for _, e := range c.Events(t) {
		batch = append(batch, e.Env)
	}

	if err := tracker.ObserveBackfill(batch); err != nil {
		t.Fatalf("ObserveBackfill: %v", err)
	}

	return tracker
}


// expectVerdicts stops the test when r differs from the verdicts of c.
func expectVerdicts(t *testing.T, c fixturetest.Case, r replay) {
	t.Helper()

	if want := c.List("accepted"); want != nil &&
	   !slices.Equal(r.accepted, want) {
		t.Fatalf("accepted = %v, want %v", r.accepted, want)
	}

	if want := c.Rejected(); want != nil &&
	   !slices.Equal(r.rejected, want) {
		t.Fatalf("rejected = %v, want %v", r.rejected, want)
	}
}


// expectSuccessor stops the test when subject resolves other than c expects.
func expectSuccessor(t *testing.T,
                     c fixturetest.Case,
                     tracker *identity.Tracker,
                     subject string) {
	t.Helper()

	head, contested := tracker.Resolve(subject)

	if want, ok := c.Expect["contested"].(bool); ok && contested != want {
		t.Fatalf("contested = %v, want %v", contested, want)
	}

	if want, ok := c.Expect["successor"].(string); ok &&
	   head != fixturetest.Author(want) {
		t.Fatalf("successor = %s, want %s", head, want)
	}

	if want, ok := c.Expect["successor"]; ok && want == nil &&
	   head != subject {
		t.Fatalf("successor = %s, want none", head)
	}
}


func TestFixtures(t *testing.T) {
	file := fixturetest.Load(t, "identity-succession")

	fixturetest.Run(t, file, func(t *testing.T, c fixturetest.Case) {
		// Arrange
		subject := fixturetest.Author(c.Arrival[0].Author)

		// Act
		var r replay
		if c.Expect["backfill"] == true {
			r.tracker = runBackfill(t, c)
		} else {
			r = run(t, c)
		}

		// Assert
		expectVerdicts(t, c, r)
		expectSuccessor(t, c, r.tracker, subject)
	})
}


func TestObserveAcceptsTheSameSuccessorTwice(t *testing.T) {
	// Arrange
	c := fixturetest.Case{Arrival: []fixturetest.Arrival{
		{ID: "a", Author: "k1", Type: "identity.key_rotated",
		 Payload: []byte(`{"successor":"k2"}`)},
		{ID: "b", Author: "k1", Type: "identity.key_rotated",
		 At: "2026-02-01T00:00:00Z", Payload: []byte(`{"successor":"k2"}`)},
	}}

	// Act
	r := run(t, c)

	// Assert
	if len(r.rejected) != 0 {
		t.Fatalf("rejected = %v, want none", r.rejected)
	}
}


func TestObserveRefusesAnEventItDoesNotTrack(t *testing.T) {
	// Arrange
	c := fixturetest.Case{Arrival: []fixturetest.Arrival{
		{ID: "a", Author: "k1", Type: "review.created",
		 Payload: []byte(`{"entity":"cnpj:1","rating":2,"body":"x"}`)},
	}}
	tracker := identity.NewTracker()

	// Act
	err := tracker.Observe(c.Events(t)[0].Env)

	// Assert
	var unsupported *identity.UnsupportedEventError
	if !errors.As(err, &unsupported) {
		t.Fatalf("Observe = %v, want UnsupportedEventError", err)
	}
}


func TestObserveBackfillKeepsASuccessorAlreadyObserved(t *testing.T) {
	// Arrange
	first := fixturetest.Case{Arrival: []fixturetest.Arrival{
		{ID: "a", Author: "k1", Type: "identity.key_rotated",
		 Payload: []byte(`{"successor":"k2"}`)},
	}}
	batch := fixturetest.Case{Arrival: []fixturetest.Arrival{
		{ID: "b", Author: "k1", Type: "identity.key_rotated",
		 Payload: []byte(`{"successor":"k3"}`)},
		{ID: "c", Author: "k1", Type: "identity.key_rotated",
		 Payload: []byte(`{"successor":"k4"}`)},
	}}
	tracker := identity.NewTracker()

	if err := tracker.Observe(first.Events(t)[0].Env); err != nil {
		t.Fatalf("Observe: %v", err)
	}

	var envs []*envelope.Envelope
	for _, e := range batch.Events(t) {
		envs = append(envs, e.Env)
	}

	// Act
	err := tracker.ObserveBackfill(envs)

	// Assert
	head, contested := tracker.Resolve(fixturetest.Author("k1"))
	if err == nil || contested || head != fixturetest.Author("k2") {
		t.Fatalf("Resolve = %s, %v after %v", head, contested, err)
	}
}
