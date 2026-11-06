package p2p


import (
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)


// --- CONSTANTS ---

// servedWindow is the period over which a peer's served events are counted.
const servedWindow = time.Minute


// --- CODE ---

// SyncLimits are the ceilings a node holds sync peers to.
type SyncLimits struct {
	// MaxStreams is the most sync streams open at once, from all peers.
	MaxStreams int

	// MaxStreamsPerPeer is the most sync streams open at once from one peer.
	MaxStreamsPerPeer int

	// EventsPerPeer is the most events served to one peer per window.
	EventsPerPeer int
}


// servedCount is the events served to one peer since its window opened.
type servedCount struct {
	opened time.Time
	events int
}


// syncLimiter decides which sync streams the node still serves.
type syncLimiter struct {
	limits SyncLimits

	// now is the clock the window is read from
	now func() time.Time

	mu     sync.Mutex
	open   int
	byPeer map[peer.ID]int
	served map[peer.ID]servedCount
}


// newSyncLimiter returns a limiter holding peers to limits.
func newSyncLimiter(limits SyncLimits, now func() time.Time) *syncLimiter {
	return &syncLimiter{
		limits: limits,
		now:    now,
		byPeer: map[peer.ID]int{},
		served: map[peer.ID]servedCount{},
	}
}


// acquire counts a stream from p and reports whether it may be served.
func (l *syncLimiter) acquire(p peer.ID) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.open >= l.limits.MaxStreams ||
	   l.byPeer[p] >= l.limits.MaxStreamsPerPeer ||
	   l.servedIn(p) >= l.limits.EventsPerPeer {
		return false
	}

	l.open++
	l.byPeer[p]++

	return true
}


// release uncounts a stream from p that acquire allowed.
func (l *syncLimiter) release(p peer.ID) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.open--
	l.byPeer[p]--

	if l.byPeer[p] == 0 {
		delete(l.byPeer, p)
	}
}


// record adds events served to p in its current window.
func (l *syncLimiter) record(p peer.ID, events int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	count := l.served[p]
	if l.servedIn(p) == 0 {
		count = servedCount{opened: l.now()}
	}

	count.events += events
	l.served[p] = count
}


// servedIn returns the events served to p in its window, 0 once it expired.
func (l *syncLimiter) servedIn(p peer.ID) int {
	count := l.served[p]

	if l.now().Sub(count.opened) >= servedWindow {
		delete(l.served, p)
		return 0
	}

	return count.events
}
