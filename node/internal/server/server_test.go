package server


import (
	"bytes"
	"context"
	"crypto/ed25519"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/internal/node"
	"github.com/eduardokairalla1/EcoaNetwork/internal/p2p"
	pb "github.com/eduardokairalla1/EcoaNetwork/internal/protobuf"
	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/testevent"
)


// --- CONSTANTS ---

// network is the network every test node belongs to.
const network = "ecoa-dev"


// --- CODE ---

// startNode returns a node on a loopback port admitting rate per hour.
func startNode(t *testing.T,
               ctx context.Context,
               rate int,
               addrs ...string) (*node.Node, *store.Store) {
	t.Helper()

	s, err := store.New(filepath.Join(t.TempDir(), "bolt.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	t.Cleanup(func() { s.Close() })

	n, err := node.New(ctx, s, p2p.Config{
		Network:        network,
		Listen:         "/ip4/127.0.0.1/tcp/0",
		BootstrapAddrs: addrs,
		Sync: p2p.SyncLimits{
			MaxStreams:        100,
			MaxStreamsPerPeer: 100,
			EventsPerPeer:     100000,
		},
	}, envelope.MaxSize, rate)
	if err != nil {
		t.Fatalf("node.New: %v", err)
	}

	t.Cleanup(func() { n.Close() })

	return n, s
}


// signed returns a valid review by the key of seed with the given body.
func signed(t *testing.T, seed byte, body string) []byte {
	t.Helper()

	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, 32))

	raw, err := testevent.Sign(key, testevent.Fields{
		Protocol:  network,
		Version:   1,
		EventType: "review.created",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Payload: `{"entity":"cnpj:12345678000199","rating":4,"body":"` +
		         body + `"}`,
	})
	if err != nil {
		t.Fatalf("testevent.Sign: %v", err)
	}

	return raw
}


// endorsing returns a vote by the key of seed on the event target.
func endorsing(t *testing.T, seed byte, target string) []byte {
	t.Helper()

	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, 32))

	raw, err := testevent.Sign(key, testevent.Fields{
		Protocol:  network,
		Version:   1,
		EventType: "review.endorsed",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Payload:   `{"target":"` + target + `"}`,
	})
	if err != nil {
		t.Fatalf("testevent.Sign: %v", err)
	}

	return raw
}


// publish sends raw to a server over n, as the gateway does.
func publish(t *testing.T, n *node.Node, raw []byte) *pb.PublishResponse {
	t.Helper()

	s := &server{node: n}

	resp, err := s.Publish(context.Background(), &pb.Event{Envelope: raw})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	return resp
}


func TestPublishAcceptsAndGossipsAValidEvent(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a, _ := startNode(t, ctx, 0)
	_, b := startNode(t, ctx, 0, a.Addrs()...)

	raw := signed(t, 1, "ok")

	// Act
	resp := publish(t, a, raw)

	// Assert
	if resp.Outcome != pb.Outcome_ACCEPTED {
		t.Fatalf("Publish = %s, want ACCEPTED", resp.Outcome)
	}

	for {
		reached, err := b.EventsSince(0, 10)
		if err != nil {
			t.Fatal(err)
		}

		if len(reached) == 1 {
			return
		}

		select {
		case <-ctx.Done():
			t.Fatal("the event was never gossiped")
		case <-time.After(10 * time.Millisecond):
		}
	}
}


func TestPublishAnswersAlreadyExistsForAReplay(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a, _ := startNode(t, ctx, 0)
	raw := signed(t, 1, "ok")
	publish(t, a, raw)

	// Act
	resp := publish(t, a, raw)

	// Assert
	if resp.Outcome != pb.Outcome_ALREADY_EXISTS {
		t.Fatalf("Publish = %s, want ALREADY_EXISTS", resp.Outcome)
	}
}


func TestPublishAnswersOrphanAndStoresNothing(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a, s := startNode(t, ctx, 0)
	target := "evt_" + strings.Repeat("a", 52)

	// Act
	resp := publish(t, a, endorsing(t, 1, target))

	// Assert
	held, err := s.EventsSince(0, 10)
	if resp.Outcome != pb.Outcome_ORPHAN || err != nil || len(held) != 0 {
		t.Fatalf("Publish = %s with %d stored, want ORPHAN and none",
		         resp.Outcome, len(held))
	}
}


func TestPublishAnswersRejectedForAnInvalidEvent(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a, _ := startNode(t, ctx, 0)

	// Act
	resp := publish(t, a, []byte(`not json`))

	// Assert
	if resp.Outcome != pb.Outcome_REJECTED || resp.Reason == "" {
		t.Fatalf("Publish = %s %q, want REJECTED with a reason",
		         resp.Outcome, resp.Reason)
	}
}


func TestPublishAnswersRejectedForAnAuthorOverTheLimit(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a, _ := startNode(t, ctx, 1)
	publish(t, a, signed(t, 1, "first"))

	// Act
	resp := publish(t, a, signed(t, 1, "second"))

	// Assert
	if resp.Outcome != pb.Outcome_REJECTED ||
	   !strings.Contains(resp.Reason, "admission limit") {
		t.Fatalf("Publish = %s %q, want REJECTED by the admission limit",
		         resp.Outcome, resp.Reason)
	}
}
