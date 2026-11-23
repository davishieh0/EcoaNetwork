package store


import (
	"sync"
)


// --- CODE ---

// bell wakes everyone waiting for the log to grow, all at once.
type bell struct {
	mu sync.Mutex
	ch chan struct{}
}


// newBell returns a bell that has not rung.
func newBell() *bell {
	return &bell{ch: make(chan struct{})}
}


// wait returns the channel the next ring closes.
func (b *bell) wait() <-chan struct{} {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.ch
}


// ring wakes every waiter and arms the bell for the next ring.
func (b *bell) ring() {
	b.mu.Lock()
	defer b.mu.Unlock()

	close(b.ch)
	b.ch = make(chan struct{})
}


// Changed returns a channel closed when the next event is stored.
func (s *Store) Changed() <-chan struct{} {
	return s.changed.wait()
}
