package store


import (
	"path/filepath"
	"testing"
)


// --- CODE ---

func TestLogIDIsCreatedWithTheStore(t *testing.T) {
	// Arrange
	s, err := New(filepath.Join(t.TempDir(), "bolt.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()

	// Act
	got := s.LogID()

	// Assert
	if len(got) != 2*logIDSize {
		t.Fatalf("LogID = %q, want %d hex characters", got, 2*logIDSize)
	}
}


func TestLogIDSurvivesReopeningTheStore(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "bolt.db")

	s, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	first := s.LogID()
	s.Close()

	// Act
	reopened, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer reopened.Close()

	// Assert
	if got := reopened.LogID(); got != first {
		t.Fatalf("LogID = %q after reopening, want %q", got, first)
	}
}


func TestLogIDDiffersBetweenStores(t *testing.T) {
	// Arrange
	dir := t.TempDir()

	a, err := New(filepath.Join(dir, "a.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer a.Close()

	b, err := New(filepath.Join(dir, "b.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	// Act
	same := a.LogID() == b.LogID()

	// Assert
	if same {
		t.Fatalf("two stores share the log id %q", a.LogID())
	}
}
