package node


import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/internal/admission"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/fixturetest"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/testevent"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)


// --- CODE ---

// limited makes n admit one event per author per hour.
func limited(n *Node) {
	n.admission = admission.New(1, time.Hour, clock)
}


// numbered returns the fields of a valid review with a body of its own.
func numbered(i int) testevent.Fields {
	fields := review()
	fields.Payload = fmt.Sprintf(`{"entity":"cnpj:12345678000199",`+
	                             `"rating":4,"body":"review %d"}`, i)

	return fields
}


func TestReceiveRefusesAnAuthorOverTheLimit(t *testing.T) {
	// Arrange
	n, path := newNode(t)
	limited(n)

	n.Receive(sign(t, 1, numbered(1)), gateway)

	// Act
	outcome, err := n.Receive(sign(t, 1, numbered(2)), gateway)

	// Assert
	var over *OverLimitError
	if outcome != OverLimit || !errors.As(err, &over) {
		t.Fatalf("Receive = %s, %v, want over the limit", outcome, err)
	}

	if got := len(stored(t, n, path)); got != 1 {
		t.Fatalf("stored %d events, want only the first", got)
	}
}


func TestReceiveCountsAuthorsSeparately(t *testing.T) {
	// Arrange
	n, _ := newNode(t)
	limited(n)

	n.Receive(sign(t, 1, numbered(1)), gateway)

	// Act
	outcome, err := n.Receive(sign(t, 2, numbered(1)), gateway)

	// Assert
	if outcome != Stored {
		t.Fatalf("Receive = %s, %v, want stored", outcome, err)
	}
}


func TestReceiveDoesNotLimitWhatSyncDelivers(t *testing.T) {
	// Arrange
	n, _ := newNode(t)
	limited(n)

	n.Receive(sign(t, 1, numbered(1)), gateway)

	// Act
	outcome, err := n.Receive(sign(t, 1, numbered(2)),
	                          Arrival{Kind: Sync, Peer: "bob"})

	// Assert
	if outcome != Stored {
		t.Fatalf("Receive = %s, %v, want stored", outcome, err)
	}
}


func TestValidateIgnoresAnAuthorOverTheLimit(t *testing.T) {
	// Arrange
	n, _ := newNode(t)
	limited(n)

	n.Receive(sign(t, 1, numbered(1)), gateway)

	// Act
	result := n.validate(context.Background(), "bob",
	                     sign(t, 1, numbered(2)))

	// Assert
	if result != pubsub.ValidationIgnore {
		t.Fatalf("validate = %v, want ignore, never reject", result)
	}
}


func TestReceiveDoesNotObserveARotationOverTheLimit(t *testing.T) {
	// Arrange
	n, _ := newNode(t)
	limited(n)

	n.Receive(sign(t, 1, numbered(1)), gateway)

	// Act
	outcome, _ := n.Receive(
		sign(t, 1, rotation(2, "2026-01-01T10:00:00Z")), gateway,
	)

	// Assert
	current, _ := n.identities.Resolve(did(1))
	if outcome != OverLimit || current != did(1) {
		t.Fatalf("Receive = %s resolving to %s, want the rotation unseen",
		         outcome, current)
	}
}


// orphanOf returns an endorsement of a review, and the review.
func orphanOf(t *testing.T) ([]byte, []byte) {
	t.Helper()

	target := sign(t, 1, review())

	return sign(t, 2, endorsement(field(t, target, "eventId"))), target
}


func TestReleaseKeepsTheLimitOffAnOrphanThatCameBySync(t *testing.T) {
	// Arrange
	n, _ := newNode(t)
	limited(n)

	child, target := orphanOf(t)

	n.Receive(sign(t, 2, numbered(1)), gateway)
	n.Receive(child, Arrival{Kind: Sync, Peer: "bob"})

	// Act
	n.Receive(target, gateway)

	// Assert
	stored, err := n.store.GetEvent(field(t, child, "eventId"))
	if err != nil || stored == nil {
		t.Fatalf("GetEvent = %q, %v, want the orphan stored", stored, err)
	}
}


func TestReleaseAppliesTheLimitToAnOrphanThatCameByTheGateway(t *testing.T) {
	// Arrange
	n, _ := newNode(t)
	limited(n)

	child, target := orphanOf(t)

	n.Receive(sign(t, 2, numbered(1)), gateway)
	n.Receive(child, gateway)

	// Act
	n.Receive(target, gateway)

	// Assert
	stored, err := n.store.GetEvent(field(t, child, "eventId"))
	if err != nil || stored != nil {
		t.Fatalf("GetEvent = %q, %v, want the orphan dropped", stored, err)
	}
}


func TestReceiveCountsAReleasedOrphanOnce(t *testing.T) {
	// Arrange
	n, _ := newNode(t)
	limited(n)

	child, target := orphanOf(t)

	n.Receive(child, gateway)

	// Act
	n.Receive(target, gateway)
	outcome, _ := n.Receive(sign(t, 2, numbered(1)), gateway)

	// Assert
	if outcome != OverLimit {
		t.Fatalf("Receive = %s, want the released orphan to have used "+
		         "the one slot", outcome)
	}
}


func TestReceiveRebroadcastsWhatTheGatewayStored(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a := startNode(t, ctx)
	b := startNode(t, ctx, a.Addrs()...)

	raw := sign(t, 1, review())

	// Act
	a.Receive(raw, gateway)

	// Assert
	waitStored(t, ctx, b, []string{field(t, raw, "eventId")})
}


func TestReleaseRebroadcastsAnOrphanTheGatewayHeld(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a := startNode(t, ctx)
	b := startNode(t, ctx, a.Addrs()...)

	child, target := orphanOf(t)
	a.Receive(child, gateway)

	// Act
	a.Receive(target, gateway)

	// Assert
	waitStored(t, ctx, b, []string{field(t, child, "eventId")})
}


func TestReceiveHoldsAnOrphanBeforeTheLimitApplies(t *testing.T) {
	// Arrange
	n, _ := newNode(t)
	limited(n)

	// Act
	outcome, _ := n.Receive(
		sign(t, 1, endorsement(fixturetest.EventID("e1"))), gateway,
	)

	// Assert
	if outcome != Orphaned {
		t.Fatalf("Receive = %s, want the orphan held before the limit",
		         outcome)
	}
}
