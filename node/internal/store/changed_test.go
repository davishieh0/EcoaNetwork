package store


import (
	"sync"
	"testing"
	"time"
)


// --- CONSTANTS ---

// wakeTimeout bounds the wait for a bell that should ring at once.
const wakeTimeout = time.Second


// --- CODE ---

// rang reports whether ch is closed, without waiting for it.
func rang(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}


func TestChangedRingsWhenAnEventIsStored(t *testing.T) {
	// Arrange
	s, _ := open(t)
	changed := s.Changed()

	// Act
	if err := s.AddEvent("evt_a", []byte(`{}`), receivedAt); err != nil {
		t.Fatalf("AddEvent: %v", err)
	}

	// Assert
	if !rang(changed) {
		t.Fatal("Changed did not ring after AddEvent")
	}
}


func TestChangedDoesNotRingForADuplicate(t *testing.T) {
	// Arrange
	s, _ := open(t)

	if err := s.AddEvent("evt_a", []byte(`{}`), receivedAt); err != nil {
		t.Fatalf("AddEvent: %v", err)
	}

	changed := s.Changed()

	// Act
	err := s.AddEvent("evt_a", []byte(`{}`), receivedAt)

	// Assert
	if err != ErrDuplicate || rang(changed) {
		t.Fatalf("AddEvent = %v, rang = %v, want a silent duplicate",
		         err, rang(changed))
	}
}


func TestChangedIsArmedAgainAfterRinging(t *testing.T) {
	// Arrange
	s, _ := open(t)

	if err := s.AddEvent("evt_a", []byte(`{}`), receivedAt); err != nil {
		t.Fatalf("AddEvent: %v", err)
	}

	// Act
	next := s.Changed()

	// Assert
	if rang(next) {
		t.Fatal("Changed rang before the next event was stored")
	}
}


func TestChangedWakesEveryWaiter(t *testing.T) {
	// Arrange
	s, _ := open(t)

	const waiters = 8

	var ready, woken sync.WaitGroup
	ready.Add(waiters)
	woken.Add(waiters)

	for range waiters {
		go func() {
			changed := s.Changed()
			ready.Done()

			<-changed
			woken.Done()
		}()
	}

	ready.Wait()

	// Act
	if err := s.AddEvent("evt_a", []byte(`{}`), receivedAt); err != nil {
		t.Fatalf("AddEvent: %v", err)
	}

	// Assert
	done := make(chan struct{})
	go func() {
		woken.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(wakeTimeout):
		t.Fatal("not every waiter woke")
	}
}


func TestChangedLetsAWaiterReadWhatWokeIt(t *testing.T) {
	// Arrange
	s, _ := open(t)
	changed := s.Changed()

	// Act
	if err := s.AddEvent("evt_a", []byte(`{}`), receivedAt); err != nil {
		t.Fatalf("AddEvent: %v", err)
	}

	<-changed
	found, err := s.EventsSince(0, 10)

	// Assert
	if err != nil || len(found) != 1 || found[0].Sequence != 1 {
		t.Fatalf("EventsSince = %+v, %v, want the event at sequence 1",
		         found, err)
	}
}
