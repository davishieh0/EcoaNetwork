package p2p

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)


func TestTopic(t *testing.T) {
	cases := []struct {
		network string
		want    string
	}{
		{"ecoa", "/ecoa/events/1"},
		{"ecoa-dev", "/ecoa-dev/events/1"},
	}

	for _, c := range cases {
		if got := Topic(c.network); got != c.want {
			t.Errorf("Topic(%q) = %q, want %q", c.network, got, c.want)
		}
	}
}


// TestTopicIsPerNetwork guards the separation the protocol relies on: two
// networks must never share a topic.
func TestTopicIsPerNetwork(t *testing.T) {
	if Topic("ecoa") == Topic("ecoa-dev") {
		t.Fatal("different networks produced the same topic")
	}
}


func TestJoinTopic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	h, err := libp2p.New(libp2p.ListenAddrStrings(loopback))
	if err != nil {
		t.Fatalf("host: %v", err)
	}
	defer h.Close()

	topic, sub, err := joinTopic(ctx, h, "ecoa-test", acceptAll)
	if err != nil {
		t.Fatalf("joinTopic: %v", err)
	}
	defer sub.Cancel()

	if topic.String() != Topic("ecoa-test") {
		t.Errorf("joined %q, want %q", topic.String(), Topic("ecoa-test"))
	}
}


func TestReceiveEventStopsWithContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	p, err := New(ctx, config())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()

	cancelled, stop := context.WithCancel(ctx)
	stop()

	_, _, err = p.ReceiveEvent(cancelled)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ReceiveEvent = %v, want a cancellation error", err)
	}
}


// TestWaitForPeersGivesUpWithNoPeers pins the first exit: alone on the
// network there is nothing to wait for, so the caller's deadline must win
// instead of the call hanging forever.
func TestWaitForPeersGivesUpWithNoPeers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	p, err := New(ctx, config())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()

	short, stop := context.WithTimeout(ctx, 200*time.Millisecond)
	defer stop()

	if err := p.WaitForPeers(short); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("WaitForPeers = %v, want a deadline error", err)
	}
}


// TestWaitForPeersGivesUpDuringHeartbeat pins the second exit: a peer is
// already there, so the wait is only for the mesh graft, and cancelling
// during it must still return rather than publish into a void.
func TestWaitForPeersGivesUpDuringHeartbeat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a, err := New(ctx, config())
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	defer a.Close()

	b, err := New(ctx, config(a.Addrs()...))
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	defer b.Close()

	deadline := time.Now().Add(10 * time.Second)
	for len(a.topic.ListPeers()) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("b never subscribed to the topic")
		}

		time.Sleep(pollInterval)
	}

	cancelled, stop := context.WithCancel(ctx)
	stop()

	if err := a.WaitForPeers(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatalf("WaitForPeers = %v, want a cancellation error", err)
	}
}


// TestPublishReachesPeer is the one that matters: an event published on one
// node has to arrive on another over Gossipsub.
func TestPublishReachesPeer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sender, err := New(ctx, config())
	if err != nil {
		t.Fatalf("sender: %v", err)
	}
	defer sender.Close()

	receiver, err := New(ctx, config(sender.Addrs()...))
	if err != nil {
		t.Fatalf("receiver: %v", err)
	}
	defer receiver.Close()

	if err := sender.WaitForPeers(ctx); err != nil {
		t.Fatalf("no peer subscribed to the topic: %v", err)
	}

	event := []byte(`{"eventType":"review.created"}`)
	if err := sender.Publish(ctx, event); err != nil {
		t.Fatalf("publish: %v", err)
	}

	got, from, err := receiver.ReceiveEvent(ctx)
	if err != nil {
		t.Fatalf("event never arrived: %v", err)
	}

	if !bytes.Equal(got, event) {
		t.Errorf("received %q, want %q", got, event)
	}

	if from != sender.host.ID() {
		t.Errorf("from = %s, want %s", from, sender.host.ID())
	}
}


// TestPublishCarriesAnEventOfTheSizeCeiling guards MaxMessage.
func TestPublishCarriesAnEventOfTheSizeCeiling(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sender, err := New(ctx, config())
	if err != nil {
		t.Fatalf("sender: %v", err)
	}
	defer sender.Close()

	receiver, err := New(ctx, config(sender.Addrs()...))
	if err != nil {
		t.Fatalf("receiver: %v", err)
	}
	defer receiver.Close()

	if err := sender.WaitForPeers(ctx); err != nil {
		t.Fatalf("no peer subscribed to the topic: %v", err)
	}

	event := bytes.Repeat([]byte("x"), envelope.MaxSize)

	// Act
	if err := sender.Publish(ctx, event); err != nil {
		t.Fatalf("publish: %v", err)
	}

	got, _, err := receiver.ReceiveEvent(ctx)

	// Assert
	if err != nil || !bytes.Equal(got, event) {
		t.Fatalf("received %d bytes, %v, want %d", len(got), err, len(event))
	}
}


func TestNewRequiresAValidator(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := config()
	cfg.Validator = nil

	// Act
	_, err := New(ctx, cfg)

	// Assert
	if err == nil {
		t.Fatal("New started a peer that would relay before validating")
	}
}


// line starts three peers in a row, the middle one judging with validate.
func line(t *testing.T,
          ctx context.Context,
          validate Validator) (first, last *Host) {
	t.Helper()

	first, err := New(ctx, config())
	if err != nil {
		t.Fatalf("first: %v", err)
	}

	t.Cleanup(func() { first.Close() })

	middleConfig := config(first.Addrs()...)
	middleConfig.Validator = validate

	middle, err := New(ctx, middleConfig)
	if err != nil {
		t.Fatalf("middle: %v", err)
	}

	t.Cleanup(func() { middle.Close() })

	last, err = New(ctx, config(middle.Addrs()...))
	if err != nil {
		t.Fatalf("last: %v", err)
	}

	t.Cleanup(func() { last.Close() })

	// the middle peer relays only once both ends are in its mesh
	for len(middle.topic.ListPeers()) < 2 {
		select {
		case <-ctx.Done():
			t.Fatal("the ends never subscribed to the middle peer")
		case <-time.After(pollInterval):
		}
	}

	if err := middle.WaitForPeers(ctx); err != nil {
		t.Fatalf("middle mesh: %v", err)
	}

	return first, last
}


func TestARelayedEventIsValidatedBeforeItReachesTheNextPeer(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	refuseBad := func(_ context.Context,
	                  _ peer.ID,
	                  raw []byte) pubsub.ValidationResult {
		if string(raw) == "bad" {
			return pubsub.ValidationIgnore
		}

		return pubsub.ValidationAccept
	}

	first, last := line(t, ctx, refuseBad)

	// Act
	badErr := first.Publish(ctx, []byte("bad"))
	goodErr := first.Publish(ctx, []byte("good"))
	got, _, receiveErr := last.ReceiveEvent(ctx)

	// Assert
	if badErr != nil || goodErr != nil {
		t.Fatalf("publish: %v, %v", badErr, goodErr)
	}

	// the first message to arrive is "good": "bad" was never relayed
	if receiveErr != nil || string(got) != "good" {
		t.Fatalf("last peer received %q, %v, want only good", got, receiveErr)
	}
}


func TestAPeerSkipsItsOwnValidatorForWhatItPublishes(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	called := false
	cfg := config()
	cfg.Validator = func(context.Context,
	                     peer.ID,
	                     []byte) pubsub.ValidationResult {
		called = true
		return pubsub.ValidationIgnore
	}

	p, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()

	// Act
	err = p.Publish(ctx, []byte("mine"))

	// Assert
	if err != nil || called {
		t.Fatalf("Publish = %v, validator called = %t, want nil, false",
		         err, called)
	}
}
