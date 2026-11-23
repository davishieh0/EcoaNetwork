package store


import (
	"crypto/rand"
	"encoding/hex"

	bolt "go.etcd.io/bbolt"
)


// --- CONSTANTS ---

// logIDSize is the number of random bytes in a log id.
const logIDSize = 16


// --- CODE ---

// LogID names the log the sequence numbers of this store count in.
func (s *Store) LogID() string {
	return s.logID
}


// ensureLogID returns the log id of tx, creating it on a store without one.
func ensureLogID(tx *bolt.Tx) (string, error) {
	info := tx.Bucket(infoBucket)

	if held := info.Get(logIDKey); held != nil {
		return string(held), nil
	}

	id := hex.EncodeToString(randomBytes(logIDSize))

	return id, info.Put(logIDKey, []byte(id))
}


// randomBytes returns n bytes from the system's secure random source.
func randomBytes(n int) []byte {
	b := make([]byte, n)

	// never fails: crypto/rand panics rather than return short
	rand.Read(b)

	return b
}
