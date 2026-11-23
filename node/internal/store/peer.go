package store


import (
	"encoding/binary"
	"time"

	bolt "go.etcd.io/bbolt"
)


// --- CODE ---

// AddPeer remembers the multiaddr of a peer, seen at the given time.
func (s *Store) AddPeer(addr string, seen time.Time) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		value := binary.BigEndian.AppendUint64(nil, uint64(seen.Unix()))

		return tx.Bucket(peersBucket).Put([]byte(addr), value)
	})
}


// KnownPeers returns the multiaddr of every remembered peer.
func (s *Store) KnownPeers() ([]string, error) {
	var addrs []string

	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(peersBucket).ForEach(func(k, _ []byte) error {
			addrs = append(addrs, string(k))
			return nil
		})
	})

	return addrs, err
}
