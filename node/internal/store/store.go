// Package store is the node's event log.
package store


import (
	"time"

	bolt "go.etcd.io/bbolt"
)


// --- CONSTANTS ---

// openTimeout bounds the wait for another process holding the file lock.
const openTimeout = time.Second


// --- CODE ---

// Store is the event log, backed by a single bbolt file.
type Store struct {
	db *bolt.DB
}


// New opens the bbolt file at path and creates the buckets it needs.
func New(path string) (*Store, error) {
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: openTimeout})
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		for _, name := range buckets {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return err
			}
		}

		return nil
	})

	// Closing releases the file lock taken by the open above.
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}


// Close releases the file lock and closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}
