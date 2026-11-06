package store


import (
	"path/filepath"
	"reflect"
	"testing"
)


// --- CODE ---

func TestKnownPeersSurvivesAReopen(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "bolt.db")

	s, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for _, addr := range []string{"/ip4/1.1.1.1/tcp/1", "/ip4/1.1.1.1/tcp/1"} {
		if err := s.AddPeer(addr); err != nil {
			t.Fatalf("AddPeer: %v", err)
		}
	}

	s.Close()

	reopened, err := New(path)
	if err != nil {
		t.Fatalf("New after reopen: %v", err)
	}
	defer reopened.Close()

	// Act
	got, err := reopened.KnownPeers()

	// Assert
	want := []string{"/ip4/1.1.1.1/tcp/1"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("KnownPeers = %v, %v, want %v", got, err, want)
	}
}


func TestKnownPeersIsEmptyForANewStore(t *testing.T) {
	// Arrange
	s, err := New(filepath.Join(t.TempDir(), "bolt.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()

	// Act
	got, err := s.KnownPeers()

	// Assert
	if err != nil || len(got) != 0 {
		t.Fatalf("KnownPeers = %v, %v, want none", got, err)
	}
}
