package store


import bolt "go.etcd.io/bbolt"


// --- CONSTANTS ---

// prunedAtSize is the byte length of the pruning time ahead of the author.
const prunedAtSize = 8


// --- CODE ---

// Tombstone returns the author of the pruned event id, if it has a tombstone.
func (s *Store) Tombstone(id string) (author string, found bool, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		value := tx.Bucket(tombstonesBucket).Get([]byte(id))
		if len(value) < prunedAtSize {
			return nil
		}

		author = string(value[prunedAtSize:])
		found = true

		return nil
	})

	return author, found, err
}
