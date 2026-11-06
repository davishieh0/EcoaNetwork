package store


import (
	"path/filepath"
	"testing"
)


// --- CODE ---

func TestCursorStartsAtZeroForANewPeer(t *testing.T) {
	// Arrange
	s, err := New(filepath.Join(t.TempDir(), "bolt.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()

	// Act
	got, err := s.Cursor("peer-a")

	// Assert
	if err != nil || got != 0 {
		t.Fatalf("Cursor = %d, %v, want 0", got, err)
	}
}


func TestCursorSurvivesReopeningTheStore(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "bolt.db")

	s, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := s.SetCursor("peer-a", 256); err != nil {
		t.Fatalf("SetCursor: %v", err)
	}
	s.Close()

	// Act
	reopened, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer reopened.Close()

	kept, _ := reopened.Cursor("peer-a")
	other, _ := reopened.Cursor("peer-b")

	// Assert
	if kept != 256 || other != 0 {
		t.Fatalf("cursors = %d and %d, want 256 and 0", kept, other)
	}
}
