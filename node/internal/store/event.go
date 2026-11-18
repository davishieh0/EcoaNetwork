package store


import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"

	bolt "go.etcd.io/bbolt"
)


// ErrDuplicate is returned by AddEvent for an id already stored.
var ErrDuplicate = errors.New("store: event already stored")


// StoredEvent is an event of the log with the sequence it was received at.
type StoredEvent struct {
	// Sequence is the local receive sequence number of the event.
	Sequence uint64

	// ID is the eventId the event is stored under.
	ID string

	// ReceivedAt is when this node stored the event.
	ReceivedAt time.Time

	// Raw is the event, byte for byte as it was received.
	Raw []byte
}


// AddEvent stores raw under id with the next sequence, never overwriting an id.
func (s *Store) AddEvent(id string,
                         raw []byte,
                         receivedAt time.Time) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return addEvent(tx, id, raw, receivedAt)
	})
}


// addEvent writes the event into events, log and meta, all within tx.
func addEvent(tx *bolt.Tx,
              id string,
              raw []byte,
              receivedAt time.Time) error {
	events := tx.Bucket(eventsBucket)

	if events.Get([]byte(id)) != nil {
		return ErrDuplicate
	}

	if err := events.Put([]byte(id), raw); err != nil {
		return err
	}

	log := tx.Bucket(logBucket)

	sequence, err := log.NextSequence()
	if err != nil {
		return err
	}

	if err := log.Put(sequenceKey(sequence), []byte(id)); err != nil {
		return err
	}

	meta := binary.BigEndian.AppendUint64(sequenceKey(sequence),
	                                      uint64(receivedAt.UnixNano()))

	return tx.Bucket(metaBucket).Put([]byte(id), meta)
}


// sequenceKey returns the big-endian log key of a sequence number.
func sequenceKey(sequence uint64) []byte {
	return binary.BigEndian.AppendUint64(nil, sequence)
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


// EventsSince returns up to limit events received after cursor, in order.
func (s *Store) EventsSince(cursor uint64, limit int) ([]StoredEvent, error) {
	var found []StoredEvent

	err := s.db.View(func(tx *bolt.Tx) error {
		events := tx.Bucket(eventsBucket)
		meta := tx.Bucket(metaBucket)
		log := tx.Bucket(logBucket).Cursor()

		key, id := log.Seek(sequenceKey(cursor))

		for ; key != nil && len(found) < limit; key, id = log.Next() {
			sequence := binary.BigEndian.Uint64(key)

			// Seek lands on the cursor itself when the log has it
			if sequence == cursor {
				continue
			}

			found = append(found, StoredEvent{
				Sequence:   sequence,
				ID:         string(id),
				ReceivedAt: receivedAtOf(meta.Get(id)),
				Raw:        bytes.Clone(events.Get(id)),
			})
		}

		return nil
	})

	return found, err
}


// receivedAtOf reads the time of receipt from a meta value.
func receivedAtOf(meta []byte) time.Time {
	// the sequence comes first, then the time of receipt in Unix nanoseconds
	if len(meta) < 16 {
		return time.Time{}
	}

	return time.Unix(0, int64(binary.BigEndian.Uint64(meta[8:16]))).UTC()
}


// Head returns the sequence of the last event stored, 0 for an empty log.
func (s *Store) Head() (uint64, error) {
	var head uint64

	err := s.db.View(func(tx *bolt.Tx) error {
		key, _ := tx.Bucket(logBucket).Cursor().Last()
		if key != nil {
			head = binary.BigEndian.Uint64(key)
		}

		return nil
	})

	return head, err
}
