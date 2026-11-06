package node


import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/internal/p2p"
	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
)


// --- CODE ---

// reopen closes the store of n and opens the file at path again.
func reopen(t *testing.T, n *Node, path string) *store.Store {
	t.Helper()

	if err := n.store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	s, err := store.New(path)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	t.Cleanup(func() { s.Close() })

	return s
}


func TestRebuildIdentityRestoresRotationsAndRevocations(t *testing.T) {
	// Arrange
	n, path := newNode(t)

	revocation := review()
	revocation.EventType = "identity.key_revoked"
	revocation.Payload = `{"reason":"compromised"}`

	n.Receive(sign(t, 1, rotation(2, "2026-01-01T10:00:00Z")), gateway)
	n.Receive(sign(t, 3, revocation), gateway)

	s := reopen(t, n, path)
	rebuilt := identity.NewTracker()

	// Act
	err := rebuildIdentity(s, rebuilt)

	// Assert
	if err != nil {
		t.Fatalf("rebuildIdentity: %v", err)
	}

	if current, _ := rebuilt.Resolve(did(1)); current != did(2) {
		t.Errorf("Resolve = %s, want %s", current, did(2))
	}

	if !rebuilt.IsRevoked(did(3)) {
		t.Error("the revocation was lost")
	}
}


func TestRebuildIdentityKeepsTheFirstRotationInSequenceOrder(t *testing.T) {
	// Arrange
	n, path := newNode(t)

	// the later createdAt arrives first, so it is the one observed first
	n.Receive(sign(t, 1, rotation(3, "2026-07-01T10:00:00Z")), gateway)
	n.Receive(sign(t, 1, rotation(2, "2026-01-01T10:00:00Z")), gateway)

	s := reopen(t, n, path)
	rebuilt := identity.NewTracker()

	// Act
	err := rebuildIdentity(s, rebuilt)

	// Assert
	if err != nil {
		t.Fatalf("rebuildIdentity: %v", err)
	}

	if current, _ := rebuilt.Resolve(did(1)); current != did(3) {
		t.Errorf("Resolve = %s, want %s", current, did(3))
	}
}


func TestNewRestoresTheIdentitiesOfTheStore(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	first, path := newNode(t)
	first.Receive(sign(t, 1, rotation(2, "2026-01-01T10:00:00Z")), gateway)

	restarted, err := New(ctx, reopen(t, first, path), p2p.Config{
		Network: network,
		Listen:  loopback,
		Sync:    roomy,
	}, envelope.MaxSize, 0)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer restarted.Close()

	// Act
	outcome, err := restarted.Receive(
		sign(t, 1, rotation(3, "2026-07-01T10:00:00Z")), gateway,
	)

	// Assert
	var observed *identity.SuccessorObservedError
	if outcome != Rejected || !errors.As(err, &observed) {
		t.Fatalf("Receive = %s, %v, want a successor already observed",
		         outcome, err)
	}
}
