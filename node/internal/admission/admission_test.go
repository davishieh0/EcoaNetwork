package admission


import (
	"testing"
	"time"
)


// --- GLOBALS ---

// start is the instant every test clock begins at.
var start = time.Date(2026, 8, 5, 17, 0, 0, 0, time.UTC)


// --- CODE ---

// clock is a manual time source.
type clock struct {
	at time.Time
}


// now returns the current instant of the clock.
func (c *clock) now() time.Time {
	return c.at
}


// advance moves the clock forward by d.
func (c *clock) advance(d time.Duration) {
	c.at = c.at.Add(d)
}


// fill asks Allow n times for author and returns how many it granted.
func fill(l *Limiter, author string, n int) int {
	granted := 0

	for range n {
		if l.Allow(author) {
			granted++
		}
	}

	return granted
}


func TestAllowGrantsTheLimitAndDeniesTheNext(t *testing.T) {
	// Arrange
	c := &clock{at: start}
	l := New(100, time.Hour, c.now)

	// Act
	granted := fill(l, "did:key:a", 100)
	next := l.Allow("did:key:a")

	// Assert
	if granted != 100 {
		t.Fatalf("granted %d of the first 100, want all", granted)
	}

	if next {
		t.Fatal("Allow granted the 101st event, want it denied")
	}
}


func TestAllowGrantsAgainOnceTheWindowHasPassed(t *testing.T) {
	// Arrange
	c := &clock{at: start}
	l := New(2, time.Hour, c.now)
	fill(l, "did:key:a", 2)

	// Act
	c.advance(time.Hour)
	again := l.Allow("did:key:a")

	// Assert
	if !again {
		t.Fatal("Allow denied after the window, want granted")
	}
}


func TestAllowDeniesWithinTheWindow(t *testing.T) {
	// Arrange
	c := &clock{at: start}
	l := New(2, time.Hour, c.now)
	fill(l, "did:key:a", 2)

	// Act
	c.advance(time.Hour - time.Second)
	again := l.Allow("did:key:a")

	// Assert
	if again {
		t.Fatal("Allow granted inside the window, want denied")
	}
}


func TestAllowCountsEachAuthorApart(t *testing.T) {
	// Arrange
	c := &clock{at: start}
	l := New(1, time.Hour, c.now)
	l.Allow("did:key:a")

	// Act
	other := l.Allow("did:key:b")

	// Assert
	if !other {
		t.Fatal("Allow denied a second author, want granted")
	}
}


func TestAllowNeverDeniesWhenTheLimitIsZero(t *testing.T) {
	// Arrange
	c := &clock{at: start}
	l := New(0, time.Hour, c.now)

	// Act
	granted := fill(l, "did:key:a", 1000)

	// Assert
	if granted != 1000 {
		t.Fatalf("granted %d of 1000, want all", granted)
	}
}


func TestAllowForgetsTheAuthorsWhoseWindowHasPassed(t *testing.T) {
	// Arrange
	c := &clock{at: start}
	l := New(1, time.Hour, c.now)
	l.Allow("did:key:a")

	// Act
	c.advance(2 * time.Hour)
	l.Allow("did:key:b")

	// Assert
	if len(l.tallies) != 1 {
		t.Fatalf("limiter holds %d authors, want only the current one",
		         len(l.tallies))
	}
}
