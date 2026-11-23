package store


import (
	"cmp"
	"encoding/binary"
	"slices"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)


// --- CONSTANTS ---

const (
	// maxKnownPeers is the most peers remembered, the most recently seen
	// kept: anyone can connect to a public node and leave addresses behind.
	maxKnownPeers = 64

	// peerExpiry is how long a peer not seen again is remembered.
	peerExpiry = 30 * 24 * time.Hour
)


// --- CODE ---

// knownAddr is one remembered multiaddr and when its peer was last seen.
type knownAddr struct {
	addr string
	seen time.Time

	// stamp reports an entry written before times were kept
	stamp bool
}


// AddPeer remembers the multiaddr of a peer, seen at the given time.
func (s *Store) AddPeer(addr string, seen time.Time) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		value := binary.BigEndian.AppendUint64(nil, uint64(seen.Unix()))

		return tx.Bucket(peersBucket).Put([]byte(addr), value)
	})
}


// KnownPeers returns the remembered multiaddrs, the most recently seen
// first, and forgets the peers not seen within peerExpiry of now and those
// beyond the maxKnownPeers most recent.
func (s *Store) KnownPeers(now time.Time) ([]string, error) {
	var kept []string

	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(peersBucket)

		known, err := readKnown(bucket, now)
		if err != nil {
			return err
		}

		peers := map[string]bool{}

		for _, k := range known {
			id := peerOf(k.addr)
			fresh := now.Sub(k.seen) <= peerExpiry
			room := peers[id] || len(peers) < maxKnownPeers

			if !fresh || !room {
				if err := bucket.Delete([]byte(k.addr)); err != nil {
					return err
				}

				continue
			}

			peers[id] = true
			kept = append(kept, k.addr)

			// it ages from now on, rather than stay new for ever
			if k.stamp {
				value := binary.BigEndian.AppendUint64(nil, uint64(now.Unix()))
				if err := bucket.Put([]byte(k.addr), value); err != nil {
					return err
				}
			}
		}

		return nil
	})

	return kept, err
}


// readKnown reads every remembered address, the most recently seen first.
// An address remembered before times were kept counts as seen now, once.
func readKnown(bucket *bolt.Bucket, now time.Time) ([]knownAddr, error) {
	var known []knownAddr

	err := bucket.ForEach(func(k, v []byte) error {
		entry := knownAddr{addr: string(k), seen: now, stamp: len(v) != 8}
		if !entry.stamp {
			entry.seen = time.Unix(int64(binary.BigEndian.Uint64(v)), 0)
		}

		known = append(known, entry)

		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.SortFunc(known, func(a, b knownAddr) int {
		return cmp.Or(b.seen.Compare(a.seen), cmp.Compare(a.addr, b.addr))
	})

	return known, nil
}


// peerOf returns the peer ID a multiaddr ends with, or the whole address.
func peerOf(addr string) string {
	if i := strings.LastIndex(addr, "/p2p/"); i >= 0 {
		return addr[i+len("/p2p/"):]
	}

	return addr
}
