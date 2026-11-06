// Package admission is the node's local policy on which events it accepts.
package admission


import (
	"sync"
	"time"
)


// --- CODE ---

// Limiter caps how many events one author may have accepted per window.
type Limiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	// mu guards the fields below; Gossipsub validates concurrently
	mu sync.Mutex

	// tallies is the window of each author, open or expired but not swept
	tallies map[string]tally

	// swept is the last time the expired windows were dropped
	swept time.Time
}


// tally is the events one author had accepted since start.
type tally struct {
	start time.Time
	count int
}


// New returns a limiter of limit events per author per window, 0 for none.
func New(limit int, window time.Duration, now func() time.Time) *Limiter {
	return &Limiter{
		limit:   limit,
		window:  window,
		now:     now,
		tallies: map[string]tally{},
	}
}


// Allow reports whether author may have one more event now, and counts it.
func (l *Limiter) Allow(author string) bool {
	if l.limit <= 0 {
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	t, found := l.tallies[author]
	if !found || now.Sub(t.start) >= l.window {
		t = tally{start: now}
	}

	if t.count >= l.limit {
		return false
	}

	t.count++
	l.tallies[author] = t

	return true
}


// sweep forgets the authors whose window has passed, at most once per window.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.swept) < l.window {
		return
	}

	for author, t := range l.tallies {
		if now.Sub(t.start) >= l.window {
			delete(l.tallies, author)
		}
	}

	l.swept = now
}
