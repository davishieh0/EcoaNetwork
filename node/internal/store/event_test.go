package store


import (
	"bytes"
	"encoding/binary"
	"errors"
	"path/filepath"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
)


// --- GLOBALS ---

// receivedAt is the local time every test event is received at.
var receivedAt = time.Date(2026, 8, 5, 17, 0, 0, 0, time.UTC)


// errForced is the failure the tests inject into a transaction.
var errForced = errors.New("forced failure")


// --- CODE ---

// open returns a store over a fresh file, and the file's path.
func open(t *testing.T) (*Store, string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "bolt.db")

	s, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	t.Cleanup(func() { s.Close() })

	return s, path
}


// add stores an event named id, or stops the test.
func add(t *testing.T, s *Store, id string) {
	t.Helper()

	if err := s.AddEvent(id, raw(id), receivedAt); err != nil {
		t.Fatalf("AddEvent %s: %v", id, err)
	}
}


// raw returns the bytes add stores under id.
func raw(id string) []byte {
	return []byte(`{"eventId":"` + id + `"}`)
}


// sequenceOf returns the sequence the log gave to id, or stops the test.
func sequenceOf(t *testing.T, s *Store, id string) uint64 {
	t.Helper()

	found, err := s.EventsSince(0, 100)
	if err != nil {
		t.Fatalf("EventsSince: %v", err)
	}

	for _, e := range found {
		if bytes.Equal(e.Raw, raw(id)) {
			return e.Sequence
		}
	}

	t.Fatalf("%s is not in the log", id)

	return 0
}


// counts returns how many entries events, log and meta hold.
func counts(t *testing.T, s *Store) [3]int {
	t.Helper()

	var sizes [3]int

	err := s.db.View(func(tx *bolt.Tx) error {
		for i, name := range [][]byte{eventsBucket, logBucket, metaBucket} {
			sizes[i] = tx.Bucket(name).Stats().KeyN
		}

		return nil
	})

	if err != nil {
		t.Fatalf("count: %v", err)
	}

	return sizes
}


func TestGetEventReturnsTheStoredBytes(t *testing.T) {
	// Arrange
	s, _ := open(t)

	raw := []byte(`{"eventId":"evt_a"}`)
	if err := s.AddEvent("evt_a", raw, receivedAt); err != nil {
		t.Fatalf("AddEvent: %v", err)
	}

	// Act
	stored, storedErr := s.GetEvent("evt_a")
	absent, absentErr := s.GetEvent("evt_b")

	// Assert
	if storedErr != nil || !bytes.Equal(stored, raw) {
		t.Fatalf("GetEvent stored = %q, %v, want %q", stored, storedErr, raw)
	}

	if absentErr != nil || absent != nil {
		t.Fatalf("GetEvent absent = %q, %v, want nil", absent, absentErr)
	}
}


func TestAddEventRecordsTheSequenceAndTheTimeOfReceipt(t *testing.T) {
	// Arrange
	s, _ := open(t)

	// Act
	add(t, s, "evt_a")

	// Assert
	sequence := sequenceOf(t, s, "evt_a")

	var meta []byte

	err := s.db.View(func(tx *bolt.Tx) error {
		meta = bytes.Clone(tx.Bucket(metaBucket).Get([]byte("evt_a")))
		return nil
	})

	if err != nil || len(meta) != 16 {
		t.Fatalf("meta = %x, %v, want 16 bytes", meta, err)
	}

	gotSequence := binary.BigEndian.Uint64(meta[:8])
	gotTime := time.Unix(0, int64(binary.BigEndian.Uint64(meta[8:])))

	if gotSequence != sequence || !gotTime.Equal(receivedAt) {
		t.Fatalf("meta = %d at %s, want %d at %s",
		         gotSequence, gotTime, sequence, receivedAt)
	}
}


func TestAddEventWritesNothingWhenTheTransactionFails(t *testing.T) {
	// Arrange
	s, _ := open(t)

	// Act
	err := s.db.Update(func(tx *bolt.Tx) error {
		err := addEvent(tx, "evt_a", []byte(`{}`), receivedAt)
		if err != nil {
			return err
		}

		return errForced
	})

	// Assert
	if !errors.Is(err, errForced) {
		t.Fatalf("Update = %v, want the forced failure", err)
	}

	if sizes := counts(t, s); sizes != [3]int{} {
		t.Fatalf("events, log, meta hold %v entries, want none", sizes)
	}
}


func TestAddEventNumbersAreIncreasingAcrossRestarts(t *testing.T) {
	// Arrange
	s, path := open(t)
	add(t, s, "evt_a")
	first := sequenceOf(t, s, "evt_a")

	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reopened, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer reopened.Close()

	// Act
	add(t, reopened, "evt_b")

	// Assert
	if second := sequenceOf(t, reopened, "evt_b"); second <= first {
		t.Fatalf("sequence after restart = %d, want more than %d",
		         second, first)
	}
}


func TestAddEventRefusesADuplicateWithoutUsingASequence(t *testing.T) {
	// Arrange
	s, _ := open(t)
	add(t, s, "evt_a")

	// Act
	err := s.AddEvent("evt_a", []byte(`{}`), receivedAt)
	add(t, s, "evt_b")

	// Assert
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("AddEvent = %v, want ErrDuplicate", err)
	}

	first := sequenceOf(t, s, "evt_a")
	if next := sequenceOf(t, s, "evt_b"); next != first+1 {
		t.Fatalf("sequence after a duplicate = %d, want %d", next, first+1)
	}
}


func TestEventsSinceReturnsTheEventsAfterTheCursorInOrder(t *testing.T) {
	// Arrange
	s, _ := open(t)
	add(t, s, "evt_c")
	add(t, s, "evt_a")
	add(t, s, "evt_b")

	// Act
	found, err := s.EventsSince(1, 10)

	// Assert
	if err != nil || len(found) != 2 {
		t.Fatalf("EventsSince = %v, %v, want 2 events", found, err)
	}

	if !bytes.Equal(found[0].Raw, raw("evt_a")) ||
	   !bytes.Equal(found[1].Raw, raw("evt_b")) {
		t.Fatalf("EventsSince = %q, %q, want evt_a, evt_b",
		         found[0].Raw, found[1].Raw)
	}
}


func TestEventsSinceStopsAtTheLimit(t *testing.T) {
	// Arrange
	s, _ := open(t)
	add(t, s, "evt_a")
	add(t, s, "evt_b")
	add(t, s, "evt_c")

	// Act
	found, err := s.EventsSince(0, 2)

	// Assert
	if err != nil || len(found) != 2 {
		t.Fatalf("EventsSince = %v, %v, want evt_a, evt_b", found, err)
	}

	if !bytes.Equal(found[1].Raw, raw("evt_b")) {
		t.Fatalf("last = %q, want evt_b", found[1].Raw)
	}
}
