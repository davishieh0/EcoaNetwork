package store


import (
	bolt "go.etcd.io/bbolt"
)


// --- CODE ---

// AddPeer remembers the multiaddr of a peer this node connected to.
func (s *Store) AddPeer(addr string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(peersBucket).Put([]byte(addr), []byte{})
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
