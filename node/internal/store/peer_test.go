package store


import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
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
	got, err := reopened.KnownPeers(now)

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
	got, err := s.KnownPeers(now)

	// Assert
	if err != nil || len(got) != 0 {
		t.Fatalf("KnownPeers = %v, %v, want none", got, err)
	}
}


func TestKnownPeersComeMostRecentFirst(t *testing.T) {
	// Arrange
	s, _ := open(t)

	s.AddPeer(addr(1, "old"), now.Add(-48*time.Hour))
	s.AddPeer(addr(2, "new"), now.Add(-time.Hour))
	s.AddPeer(addr(3, "mid"), now.Add(-24*time.Hour))

	// Act
	got, err := s.KnownPeers(now)

	// Assert
	want := []string{addr(2, "new"), addr(3, "mid"), addr(1, "old")}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("KnownPeers = %v, %v, want %v", got, err, want)
	}
}


func TestKnownPeersForgetsAPeerNotSeenForThirtyDays(t *testing.T) {
	// Arrange
	s, _ := open(t)

	s.AddPeer(addr(1, "gone"), now.Add(-31*24*time.Hour))
	s.AddPeer(addr(2, "here"), now.Add(-29*24*time.Hour))

	// Act
	first, _ := s.KnownPeers(now)
	again, _ := s.KnownPeers(now.Add(-40 * 24 * time.Hour))

	// Assert: forgotten, not only left out
	if !reflect.DeepEqual(first, []string{addr(2, "here")}) ||
	   !reflect.DeepEqual(again, []string{addr(2, "here")}) {
		t.Fatalf("KnownPeers = %v, then %v, want only here", first, again)
	}
}


func TestKnownPeersKeepsTheSixtyFourMostRecentPeers(t *testing.T) {
	// Arrange: 70 peers, two addresses each, a minute apart
	s, _ := open(t)

	for i := range 70 {
		seen := now.Add(-time.Duration(i) * time.Minute)
		s.AddPeer(addr(1, fmt.Sprintf("p%02d", i)), seen)
		s.AddPeer(addr(2, fmt.Sprintf("p%02d", i)), seen)
	}

	// Act
	got, err := s.KnownPeers(now)

	// Assert: 64 peers, both addresses of each, the oldest six forgotten
	peers := map[string]bool{}
	for _, a := range got {
		peers[peerOf(a)] = true
	}

	if err != nil || len(peers) != 64 || len(got) != 128 || peers["p64"] {
		t.Fatalf("kept %d peers, %d addresses, p64 = %v", len(peers), len(got),
		         peers["p64"])
	}
}


func TestKnownPeersKeepsAnAddressRememberedBeforeTimesWere(t *testing.T) {
	// Arrange: an entry as the store wrote it before, with no time
	s, _ := open(t)

	err := s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(peersBucket).Put([]byte(addr(1, "legacy")), []byte{})
	})
	if err != nil {
		t.Fatalf("write legacy: %v", err)
	}

	// Act
	got, _ := s.KnownPeers(now)

	// Assert: kept, and stamped so it ages from now
	var stamped []byte

	s.db.View(func(tx *bolt.Tx) error {
		stamped = tx.Bucket(peersBucket).Get([]byte(addr(1, "legacy")))
		return nil
	})

	if !reflect.DeepEqual(got, []string{addr(1, "legacy")}) ||
	   len(stamped) != 8 ||
	   int64(binary.BigEndian.Uint64(stamped)) != now.Unix() {
		t.Fatalf("KnownPeers = %v, stamped %v", got, stamped)
	}
}
