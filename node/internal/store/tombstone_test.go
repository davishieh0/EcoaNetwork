package store


import (
	"encoding/binary"
	"testing"

	bolt "go.etcd.io/bbolt"
)


func TestNewCreatesAnEmptyTombstonesBucket(t *testing.T) {
	// Arrange
	s, _ := open(t)

	// Act
	var keys int

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(tombstonesBucket)
		if bucket == nil {
			t.Fatal("no tombstones bucket")
		}

		keys = bucket.Stats().KeyN
		return nil
	})

	// Assert
	if err != nil || keys != 0 {
		t.Fatalf("tombstones hold %d keys, %v, want none", keys, err)
	}
}


func TestTombstoneIsNotFoundForAnIdWithoutOne(t *testing.T) {
	// Arrange
	s, _ := open(t)

	// Act
	author, found, err := s.Tombstone("evt_a")

	// Assert
	if err != nil || found || author != "" {
		t.Fatalf("Tombstone = %q, %t, %v, want not found",
		         author, found, err)
	}
}


func TestTombstoneReturnsTheAuthorOfAPrunedEvent(t *testing.T) {
	// Arrange
	s, _ := open(t)

	// written straight into the bucket: the node itself never prunes
	err := s.db.Update(func(tx *bolt.Tx) error {
		prunedAt := uint64(receivedAt.UnixNano())
		value := binary.BigEndian.AppendUint64(nil, prunedAt)
		value = append(value, "did:key:z6MkAuthor"...)

		return tx.Bucket(tombstonesBucket).Put([]byte("evt_a"), value)
	})
	if err != nil {
		t.Fatalf("write tombstone: %v", err)
	}

	// Act
	author, found, err := s.Tombstone("evt_a")

	// Assert
	if err != nil || !found || author != "did:key:z6MkAuthor" {
		t.Fatalf("Tombstone = %q, %t, %v, want did:key:z6MkAuthor",
		         author, found, err)
	}
}
