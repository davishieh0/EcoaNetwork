package node


import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/testevent"
	bolt "go.etcd.io/bbolt"
)


// --- CONSTANTS ---

// network is the network every test node belongs to.
const network = "ecoa-dev"


// --- GLOBALS ---

// now is the local clock of every test node.
var now = time.Date(2026, 8, 5, 17, 0, 0, 0, time.UTC)


// gateway is the arrival the tests hand events through.
var gateway = Arrival{Kind: Gateway}


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

	rules := envelope.Rules{
		Network: network,
		MaxSize: envelope.MaxSize,
		Now:     func() time.Time { return now },
	}

	n, err := build(s, rules)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	return n, path
}


// review returns the fields of a valid review on network.
func review() testevent.Fields {
	return testevent.Fields{
		Protocol:  network,
		Version:   1,
		EventType: "review.created",
		CreatedAt: "2026-08-05T17:00:00Z",
		Payload: `{"entity":"cnpj:12345678000199",` +
		         `"rating":4,"body":"ok"}`,
	}
}


// sign returns fields signed by the key of seed, or stops the test.
func sign(t *testing.T, seed byte, fields testevent.Fields) []byte {
	t.Helper()

	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, 32))

	raw, err := testevent.Sign(key, fields)
	if err != nil {
		t.Fatalf("testevent.Sign: %v", err)
	}

	return raw
}


// withField returns raw with the top-level field name set to value.
func withField(t *testing.T, raw []byte, name string, value any) []byte {
	t.Helper()

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	fields[name] = encoded

	edited, err := json.Marshal(fields)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	return edited
}


// field returns the string value of the top-level field name of raw.
func field(t *testing.T, raw []byte, name string) string {
	t.Helper()

	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	value, _ := fields[name].(string)

	return value
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


func TestReceiveStoresAValidEventByteForByte(t *testing.T) {
	// Arrange
	n, path := newNode(t)
	raw := sign(t, 1, review())

	// Act
	outcome, err := n.Receive(raw, gateway)

	// Assert
	if outcome != Stored || err != nil {
		t.Fatalf("Receive = %s, %v, want stored", outcome, err)
	}

	entries := stored(t, n, path)
	id := field(t, raw, "eventId")

	if len(entries) != 1 || !bytes.Equal(entries[id], raw) {
		t.Fatalf("stored %q, want only %s = %q", entries, id, raw)
	}
}


func TestReceiveStoresAnUnknownEventType(t *testing.T) {
	// Arrange
	n, _ := newNode(t)
	opaque := review()
	opaque.EventType = "thing.happened"
	opaque.Payload = `{"anything":[1,2]}`

	// Act
	outcome, err := n.Receive(sign(t, 1, opaque), gateway)

	// Assert
	if outcome != Stored || err != nil {
		t.Fatalf("Receive = %s, %v, want stored", outcome, err)
	}
}


func TestReceiveDropsAReplay(t *testing.T) {
	// Arrange
	n, path := newNode(t)
	first := sign(t, 1, review())

	if outcome, err := n.Receive(first, gateway); outcome != Stored {
		t.Fatalf("first Receive = %s, %v, want stored", outcome, err)
	}

	// same eventId with a signature that does not verify
	other := field(t, sign(t, 2, review()), "signature")
	replay := withField(t, first, "signature", other)

	// Act
	outcome, err := n.Receive(replay, gateway)

	// Assert
	if outcome != Duplicate || err != nil {
		t.Fatalf("Receive = %s, %v, want duplicate", outcome, err)
	}

	id := field(t, first, "eventId")
	if entries := stored(t, n, path); !bytes.Equal(entries[id], first) {
		t.Fatalf("stored %q, want the first %s", entries[id], id)
	}
}


func TestReceiveIgnoresAnotherNetwork(t *testing.T) {
	// Arrange
	n, path := newNode(t)
	other := review()
	other.Protocol = "ecoa-other"

	// Act
	outcome, _ := n.Receive(sign(t, 1, other), gateway)

	// Assert
	if outcome != Ignored {
		t.Fatalf("Receive = %s, want ignored", outcome)
	}

	if entries := stored(t, n, path); len(entries) != 0 {
		t.Fatalf("stored %q, want nothing", entries)
	}
}


func TestReceiveRejects(t *testing.T) {
	badRating := review()
	badRating.Payload = `{"entity":"cnpj:12345678000199",` +
	                    `"rating":9,"body":"ok"}`

	future := review()
	future.CreatedAt = "2026-08-05T17:06:00Z"

	device := review()
	deviceKey := "did:key:z6MkiTBz1ymuepAQ4HEHYSF1H8quG5GLVVQR3djdX3mDooWp"
	device.DeviceKey = &deviceKey

	valid := sign(t, 1, review())
	other := sign(t, 2, review())

	cases := []struct {
		name string
		raw  []byte
	}{
		{"too large", bytes.Repeat([]byte(" "), envelope.MaxSize+1)},
		{"not json", []byte(`not json`)},
		{"future createdAt", sign(t, 1, future)},
		{"bad payload", sign(t, 1, badRating)},
		{"tampered body", bytes.Replace(valid, []byte(`"ok"`),
		                                []byte(`"ko"`), 1)},
		{"forged eventId", withField(t, valid, "eventId",
		                             field(t, other, "eventId"))},
		{"other signature", withField(t, valid, "signature",
		                              field(t, other, "signature"))},
		{"device key", sign(t, 1, device)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			n, path := newNode(t)

			// Act
			outcome, err := n.Receive(c.raw, gateway)

			// Assert
			if outcome != Rejected || err == nil {
				t.Fatalf("Receive = %s, %v, want rejected", outcome, err)
			}

			if entries := stored(t, n, path); len(entries) != 0 {
				t.Fatalf("stored %q, want nothing", entries)
			}
		})
	}
}


func TestReceiveFailsWhenTheStoreIsClosed(t *testing.T) {
	// Arrange
	n, _ := newNode(t)
	if err := n.store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	// Act
	outcome, err := n.Receive(sign(t, 1, review()), gateway)

	// Assert
	if outcome != Failed || err == nil {
		t.Fatalf("Receive = %s, %v, want failed", outcome, err)
	}
}
