// Package orphan holds the events whose references have not resolved yet.
package orphan


import (
	"container/list"
	"slices"
	"sync"
	"time"
)


// --- CODE ---

// Entry is one event waiting for the events it names.
type Entry struct {
	// Raw is the event, byte for byte as received.
	Raw []byte

	// Missing holds the eventIds this node lacks.
	Missing []string

	// Origin is the peer that delivered it, or the gateway.
	Origin string

	// Kind is the path it came in by, which the buffer hands back unread.
	Kind int
}


// Buffer is a bounded, oldest-first set of entries, with a share per origin.
type Buffer struct {
	mu sync.Mutex

	// maxOrphans bounds the whole buffer
	maxOrphans int

	// orphanShare bounds the entries of one origin
	orphanShare int

	// orphanAge is how long an entry may wait
	orphanAge time.Duration

	// now reads the clock
	now func() time.Time

	// all holds every entry, oldest at the front
	all *list.List

	// byOrigin holds the entries of each origin, oldest at the front
	byOrigin map[string]*list.List

	// waiting holds the entries waiting on each missing eventId
	waiting map[string][]*held
}


// held is an entry and where the buffer keeps it.
type held struct {
	entry Entry

	// at is when the entry was added
	at time.Time

	// inAll and inOrigin are its places in the two lists
	inAll    *list.Element
	inOrigin *list.Element
}


// New returns an empty buffer with the given bounds.
func New(maxOrphans int,
         orphanShare int,
         orphanAge time.Duration,
         now func() time.Time) *Buffer {
	return &Buffer{
		maxOrphans:  maxOrphans,
		orphanShare: orphanShare,
		orphanAge:   orphanAge,
		now:         now,
		all:         list.New(),
		byOrigin:    map[string]*list.List{},
		waiting:     map[string][]*held{},
	}
}


// Add holds e, evicting the oldest of its origin and then the oldest overall.
func (b *Buffer) Add(e Entry) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.expire()

	mine := b.byOrigin[e.Origin]
	if mine != nil && mine.Len() >= b.orphanShare {
		b.remove(oldest(mine))
	}

	if b.all.Len() >= b.maxOrphans {
		b.remove(oldest(b.all))
	}

	h := &held{entry: e, at: b.now()}
	h.inAll = b.all.PushBack(h)

	if b.byOrigin[e.Origin] == nil {
		b.byOrigin[e.Origin] = list.New()
	}

	h.inOrigin = b.byOrigin[e.Origin].PushBack(h)

	for _, id := range e.Missing {
		b.waiting[id] = append(b.waiting[id], h)
	}
}


// Release removes and returns the entries waiting for targetID, oldest first.
func (b *Buffer) Release(targetID string) []Entry {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.expire()

	var released []Entry

	for _, h := range slices.Clone(b.waiting[targetID]) {
		released = append(released, h.entry)
		b.remove(h)
	}

	return released
}


// HasRoom reports whether origin can add an entry and keep all of its own.
func (b *Buffer) HasRoom(origin string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.expire()

	mine := b.byOrigin[origin]

	return mine == nil || mine.Len() < b.orphanShare
}


// expire removes the entries older than orphanAge.
func (b *Buffer) expire() {
	for b.all.Len() > 0 {
		h := oldest(b.all)
		if b.now().Sub(h.at) < b.orphanAge {
			return
		}

		b.remove(h)
	}
}


// remove takes h out of every index.
func (b *Buffer) remove(h *held) {
	b.all.Remove(h.inAll)

	mine := b.byOrigin[h.entry.Origin]
	mine.Remove(h.inOrigin)

	if mine.Len() == 0 {
		delete(b.byOrigin, h.entry.Origin)
	}

	for _, id := range h.entry.Missing {
		b.waiting[id] = slices.DeleteFunc(b.waiting[id],
		                                  func(w *held) bool { return w == h })

		if len(b.waiting[id]) == 0 {
			delete(b.waiting, id)
		}
	}
}


// oldest returns the entry at the front of l.
func oldest(l *list.List) *held {
	h, _ := l.Front().Value.(*held)
	return h
}
