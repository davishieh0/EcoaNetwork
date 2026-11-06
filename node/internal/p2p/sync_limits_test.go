package p2p


import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)


// --- CODE ---

// fakeClock is a clock the test moves by hand.
type fakeClock struct {
	at time.Time
}


func (c *fakeClock) now() time.Time {
	return c.at
}


var tight = SyncLimits{MaxStreams: 3, MaxStreamsPerPeer: 2, EventsPerPeer: 10}


func TestSyncLimiterRefusesAStreamOverThePeerLimit(t *testing.T) {
	// Arrange
	clock := &fakeClock{at: time.Unix(1000, 0)}
	l := newSyncLimiter(tight, clock.now)

	l.acquire(peer.ID("a"))
	l.acquire(peer.ID("a"))

	// Act
	overLimit := l.acquire(peer.ID("a"))
	other := l.acquire(peer.ID("b"))

	// Assert
	if overLimit {
		t.Error("a third stream from one peer should be refused")
	}

	if !other {
		t.Error("another peer should still be served")
	}
}


func TestSyncLimiterRefusesAStreamOverTheTotalLimit(t *testing.T) {
	// Arrange
	clock := &fakeClock{at: time.Unix(1000, 0)}
	l := newSyncLimiter(tight, clock.now)

	l.acquire(peer.ID("a"))
	l.acquire(peer.ID("b"))
	l.acquire(peer.ID("c"))

	// Act
	got := l.acquire(peer.ID("d"))

	// Assert
	if got {
		t.Error("a fourth stream should be refused")
	}
}


func TestSyncLimiterServesAgainAfterRelease(t *testing.T) {
	// Arrange
	clock := &fakeClock{at: time.Unix(1000, 0)}
	l := newSyncLimiter(tight, clock.now)

	l.acquire(peer.ID("a"))
	l.acquire(peer.ID("a"))

	// Act
	l.release(peer.ID("a"))
	got := l.acquire(peer.ID("a"))

	// Assert
	if !got {
		t.Error("a freed stream should let the peer in again")
	}
}


func TestSyncLimiterRefusesUntilTheWindowPasses(t *testing.T) {
	// Arrange
	clock := &fakeClock{at: time.Unix(1000, 0)}
	l := newSyncLimiter(tight, clock.now)

	l.record(peer.ID("a"), 10)

	// Act
	during := l.acquire(peer.ID("a"))
	clock.at = clock.at.Add(servedWindow)
	after := l.acquire(peer.ID("a"))

	// Assert
	if during {
		t.Error("a peer at its event limit should be refused")
	}

	if !after {
		t.Error("the peer should be served once the window passed")
	}
}
