package store


import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)


// --- GLOBALS ---

// now is the time the tests read the known peers at.
var now = time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)


// --- CODE ---

// addr is a multiaddr of the peer named id.
func addr(port int, id string) string {
	return fmt.Sprintf("/ip4/1.1.1.1/tcp/%d/p2p/%s", port, id)
}


func TestKnownPeersSurvivesAReopen(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "bolt.db")

	s, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for range 2 {
		if err := s.AddPeer(addr(1, "a"), now); err != nil {
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
	want := []string{addr(1, "a")}
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
