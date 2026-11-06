package store


import (
	"encoding/binary"

	bolt "go.etcd.io/bbolt"
)


// --- CODE ---

// Cursor returns the last sync cursor received from peer, 0 for a new peer.
func (s *Store) Cursor(peer string) (uint64, error) {
	var cursor uint64

	err := s.db.View(func(tx *bolt.Tx) error {
		raw := tx.Bucket(cursorsBucket).Get([]byte(peer))
		if raw != nil {
			cursor = binary.BigEndian.Uint64(raw)
		}

		return nil
	})

	return cursor, err
}


// SetCursor records c as the last sync cursor received from peer.
func (s *Store) SetCursor(peer string, c uint64) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		raw := binary.BigEndian.AppendUint64(nil, c)

		return tx.Bucket(cursorsBucket).Put([]byte(peer), raw)
	})
}
