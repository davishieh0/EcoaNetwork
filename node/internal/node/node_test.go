package node


import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
	bolt "go.etcd.io/bbolt"
)


// --- CODE ---

// newNode returns a node over a fresh store, and the store's file path.
func newNode(t *testing.T) (*Node, string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "bolt.db")

	s, err := store.New(path)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	t.Cleanup(func() { s.Close() })

	return &Node{store: s}, path
}


// stored reads every entry in the events bucket of the file at path.
func stored(t *testing.T, n *Node, path string) map[string][]byte {
	t.Helper()

	// the store holds the file lock until closed
	if err := n.store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	db, err := bolt.Open(path, 0o600, &bolt.Options{ReadOnly: true})
	if err != nil {
		t.Fatalf("bolt.Open: %v", err)
	}

	defer db.Close()

	entries := map[string][]byte{}
	err = db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("events")).ForEach(func(k, v []byte) error {
			entries[string(k)] = bytes.Clone(v)
			return nil
		})
	})

	if err != nil {
		t.Fatalf("read events: %v", err)
	}

	return entries
}


func TestHandleStoresTheRawBytesUnderTheEventID(t *testing.T) {
	// Arrange
	n, path := newNode(t)
	raw := []byte(`{ "eventType":"review.created",  "eventId":"evt_1" }`)

	// Act
	err := n.Handle(raw)

	// Assert
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	entries := stored(t, n, path)
	if len(entries) != 1 || !bytes.Equal(entries["evt_1"], raw) {
		t.Fatalf("stored %q, want only evt_1 = %q", entries, raw)
	}
}


func TestHandleStoresEachEventOnce(t *testing.T) {
	// Arrange
	n, path := newNode(t)
	first := []byte(`{"eventId":"evt_1","n":1}`)
	second := []byte(`{"eventId":"evt_1","n":2}`)

	if err := n.Handle(first); err != nil {
		t.Fatalf("first Handle: %v", err)
	}

	// Act
	err := n.Handle(second)

	// Assert
	if !errors.Is(err, store.ErrDuplicate) {
		t.Fatalf("second Handle = %v, want store.ErrDuplicate", err)
	}

	entries := stored(t, n, path)
	if len(entries) != 1 || !bytes.Equal(entries["evt_1"], first) {
		t.Fatalf("stored %q, want only the first evt_1", entries)
	}
}


func TestHandleRejectsWhatIsNotAnEnvelope(t *testing.T) {
	cases := []struct {
		name string
		raw  []byte
	}{
		{"not json", []byte(`not json`)},
		{"empty", []byte(``)},
		{"no event id", []byte(`{"eventType":"review.created"}`)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			n, path := newNode(t)

			// Act
			err := n.Handle(c.raw)

			// Assert
			if err == nil {
				t.Fatal("Handle = nil, want an error")
			}

			if entries := stored(t, n, path); len(entries) != 0 {
				t.Fatalf("stored %q, want nothing", entries)
			}
		})
	}
}
