package store


import (
	"bytes"
	"errors"

	bolt "go.etcd.io/bbolt"
)


// ErrDuplicate is returned by AddEvent for an id already stored.
var ErrDuplicate = errors.New("store: event already stored")


// AddEvent stores raw, byte for byte, under id, never overwriting an id.
func (s *Store) AddEvent(id string, raw []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(eventsBucket)

		if bucket.Get([]byte(id)) != nil {
			return ErrDuplicate
		}

		return bucket.Put([]byte(id), raw)
	})
}


// GetEvent returns the event stored under id, or nil when there is none.
func (s *Store) GetEvent(id string) ([]byte, error) {
	var raw []byte

	err := s.db.View(func(tx *bolt.Tx) error {
		// copy of the stored bytes, nil when id is absent
		raw = bytes.Clone(tx.Bucket(eventsBucket).Get([]byte(id)))
		return nil
	})

	return raw, err
}
