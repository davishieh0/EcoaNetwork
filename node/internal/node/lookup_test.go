package node


import (
	"encoding/binary"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
	bolt "go.etcd.io/bbolt"
)


// --- CODE ---

func TestLookupReadsTheAuthorOfAPrunedEventFromItsTombstone(t *testing.T) {
	// Arrange
	n, path := newNode(t)

	if err := n.store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	db, err := bolt.Open(path, 0o600, nil)
	if err != nil {
		t.Fatalf("bolt.Open: %v", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		value := binary.BigEndian.AppendUint64(nil, 1)
		value = append(value, did(1)...)

		return tx.Bucket([]byte("tombstones")).Put([]byte("evt_a"), value)
	})
	if err != nil {
		t.Fatalf("write tombstone: %v", err)
	}

	db.Close()

	s, err := store.New(path)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	// Act
	author, found, err := lookup{s, identity.NewTracker()}.Author("evt_a")

	// Assert
	if err != nil || !found || author != did(1) {
		t.Fatalf("Author = %q, %t, %v, want %s", author, found, err, did(1))
	}
}
