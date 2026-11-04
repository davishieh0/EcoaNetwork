package store


import (
	"bytes"
	"path/filepath"
	"testing"
)


// --- CODE ---

func TestGetEventReturnsTheStoredBytes(t *testing.T) {
	// Arrange
	s, err := New(filepath.Join(t.TempDir(), "bolt.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()

	raw := []byte(`{"eventId":"evt_a"}`)
	if err := s.AddEvent("evt_a", raw); err != nil {
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
