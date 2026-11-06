// Package node decides what happens to each event the node receives.
package node


import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/internal/orphan"
	"github.com/eduardokairalla1/EcoaNetwork/internal/p2p"
	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/event"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/references"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/schema"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)


// --- CONSTANTS ---

const (
	// Stored is an event that passed every check and is now in the log.
	Stored Outcome = iota + 1

	// Duplicate is an eventId already in the log.
	Duplicate

	// Ignored is an event of another network or version.
	Ignored

	// Rejected is an event that failed a check.
	Rejected

	// Failed is an event the node could not judge, for a fault of its own.
	Failed

	// Orphaned is an event held until the events it names arrive.
	Orphaned
)


// maxOrphans is the entries the orphan buffer holds.
const maxOrphans = 10000


// orphanShare is the entries of the orphan buffer one origin may hold.
const orphanShare = maxOrphans / 10


// orphanAge is how long an orphan waits for its target.
const orphanAge = time.Hour


// gatewayOrigin is the origin the orphan buffer charges the gateway to.
const gatewayOrigin = "gateway"


// --- GLOBALS ---

// outcomeNames is the log name of each outcome.
var outcomeNames = map[Outcome]string{
	Stored:    "stored",
	Duplicate: "duplicate",
	Ignored:   "ignored",
	Rejected:  "rejected",
	Failed:    "failed",
	Orphaned:  "orphaned",
}


// --- CODE ---

// Node is the one path every arrival is handed to.
type Node struct {
	store *store.Store
	host  *p2p.Host

	// rules are the size, network and clock this node opens envelopes by
	rules envelope.Rules

	// schemas validate payloads by eventType
	schemas *schema.Set

	// identities are the rotations and revocations this node observed
	identities *identity.Tracker

	// orphans hold the events whose references have not resolved
	orphans *orphan.Buffer

	// mu serializes events from the revocation check to the store
	mu sync.Mutex
}


// Outcome is what the node did with one received event.
type Outcome int


// New returns a node that judges events by the rules of cfg.Network.
func New(ctx context.Context,
         s *store.Store,
         cfg p2p.Config,
         maxEventSize int) (*Node, error) {
	rules := envelope.Rules{
		Network: cfg.Network,
		MaxSize: maxEventSize,
		Now:     time.Now,
	}

	orphans := orphan.New(maxOrphans, orphanShare, orphanAge, time.Now)

	n, err := build(s, rules, orphans)
	if err != nil {
		return nil, err
	}

	cfg.Validator = n.validate

	n.host, err = p2p.New(ctx, cfg)
	if err != nil {
		return nil, err
	}

	n.host.OnConnect(func(id peer.ID) { n.syncWith(ctx, id) })

	return n, nil
}


// build returns a node over s with no network, judging events by rules.
func build(s *store.Store,
           rules envelope.Rules,
           orphans *orphan.Buffer) (*Node, error) {
	schemas, err := schema.Load()
	if err != nil {
		return nil, err
	}

	n := &Node{
		store:      s,
		rules:      rules,
		schemas:    schemas,
		identities: identity.NewTracker(),
		orphans:    orphans,
	}

	return n, nil
}


// Run keeps the subscription, and so the mesh, alive until ctx is cancelled.
func (n *Node) Run(ctx context.Context) {
	for {
		if _, _, err := n.host.ReceiveEvent(ctx); err != nil {
			return
		}
	}
}


// validate translates the outcome of Receive for Gossipsub.
func (n *Node) validate(_ context.Context,
                        from peer.ID,
                        raw []byte) pubsub.ValidationResult {
	outcome, _ := n.Receive(raw, Arrival{Kind: Gossip, Peer: from})
	if outcome == Stored {
		return pubsub.ValidationAccept
	}

	return pubsub.ValidationIgnore
}


// Receive validates one event and stores it, however it arrived.
func (n *Node) Receive(raw []byte, arrival Arrival) (Outcome, error) {
	env, err := envelope.Open(raw, n.rules)
	if _, ok := errors.AsType[*envelope.OtherNetworkError](err); ok {
		return Ignored, err
	}

	if err != nil {
		return report(arrival, Rejected, err)
	}

	stored, err := n.store.GetEvent(env.EventID)
	if err != nil {
		return report(arrival, Failed, err)
	}

	if stored != nil {
		return Duplicate, nil
	}

	verified, err := event.Verify(env, n.schemas)
	if err != nil {
		return report(arrival, Rejected, err)
	}

	outcome, err := n.settle(verified, arrival)
	if outcome == Stored {
		n.release(env.EventID)
	}

	return outcome, err
}


// settle stores a verified event, or holds it when a reference is missing.
func (n *Node) settle(verified event.Event, arrival Arrival) (Outcome, error) {
	env := verified.Envelope()

	n.mu.Lock()
	defer n.mu.Unlock()

	if err := n.observe(env); err != nil {
		return report(arrival, Rejected, err)
	}

	result, err := references.Check(env, lookup{n.store, n.identities})
	if err != nil {
		return report(arrival, Failed, err)
	}

	if result.Kind == references.Rejected {
		return report(arrival, Rejected, result.Err)
	}

	if result.Kind == references.Unresolved {
		return n.hold(verified, result.Missing, arrival), nil
	}

	err = n.store.AddEvent(verified.ID(), verified.Raw(), n.rules.Now())
	if errors.Is(err, store.ErrDuplicate) {
		return Duplicate, nil
	}

	if err != nil {
		return report(arrival, Failed, err)
	}

	slog.Info("stored", "arrival", arrival.Kind, "bytes", len(verified.Raw()))
	return Stored, nil
}


// hold parks an event until the events it names are stored.
func (n *Node) hold(verified event.Event,
                    missing []string,
                    arrival Arrival) Outcome {
	n.orphans.Add(orphan.Entry{
		Raw:     verified.Raw(),
		Missing: missing,
		Origin:  originOf(arrival),
	})

	slog.Info("orphaned",
	          "arrival", arrival.Kind,
	          "peer", arrival.Peer,
	          "missing", missing[0])

	return Orphaned
}


// release hands every orphan waiting for id back to Receive.
func (n *Node) release(id string) {
	for _, entry := range n.orphans.Release(id) {
		n.Receive(entry.Raw, arrivalOf(entry.Origin))
	}
}


// observe refuses a revoked key and records a rotation or a revocation.
func (n *Node) observe(env *envelope.Envelope) error {
	if n.identities.IsRevoked(env.Author) {
		return &RevokedKeyError{Key: env.Author}
	}

	switch env.EventType {
	case "identity.key_rotated", "identity.key_revoked":
		return n.identities.Observe(env)
	}

	return nil
}


// report logs an event the node rejected or failed to store.
func report(arrival Arrival, outcome Outcome, err error) (Outcome, error) {
	level := slog.LevelInfo
	if outcome == Failed {
		level = slog.LevelWarn
	}

	slog.Log(context.Background(), level, outcome.String(),
	         "arrival", arrival.Kind,
	         "peer", arrival.Peer,
	         "reason", err)

	return outcome, err
}


// Publish waits for the mesh first, since Gossipsub drops unroutable messages.
func (n *Node) Publish(ctx context.Context, raw []byte) error {
	if err := n.host.WaitForPeers(ctx); err != nil {
		return err
	}

	return n.host.Publish(ctx, raw)
}


func (n *Node) Addrs() []string {
	return n.host.Addrs()
}


func (n *Node) Close() error {
	return n.host.Close()
}


// String returns the log name of the outcome.
func (o Outcome) String() string {
	return outcomeNames[o]
}


// syncWith catches up with p and logs why it could not.
func (n *Node) syncWith(ctx context.Context, p peer.ID) {
	if err := n.catchUp(ctx, p); err != nil {
		slog.Warn("catch up stopped", "peer", p, "reason", err)
	}
}


// catchUp pulls every page p accepted since the cursor kept for it.
func (n *Node) catchUp(ctx context.Context, p peer.ID) error {
	cursor, err := n.store.Cursor(p.String())
	if err != nil {
		return err
	}

	for {
		page, err := n.host.GetEventsSince(ctx, p, cursor)
		if err != nil {
			return err
		}

		if page.HasMore && page.NextCursor <= cursor {
			return fmt.Errorf("cursor stuck at %d with more to send", cursor)
		}

		if err := n.receivePage(page.Events, p); err != nil {
			return err
		}

		cursor = page.NextCursor

		if err := n.store.SetCursor(p.String(), cursor); err != nil {
			return err
		}

		if !page.HasMore {
			return nil
		}
	}
}


// receivePage hands each event of a page to Receive, as an arrival from p.
func (n *Node) receivePage(events [][]byte, p peer.ID) error {
	arrival := Arrival{Kind: Sync, Peer: p}

	for _, raw := range events {
		outcome, err := n.Receive(raw, arrival)

		// A rejected event is the peer's fault; a failed one is ours, and
		// moving the cursor past it would lose it.
		if outcome == Failed {
			return fmt.Errorf("receive: %w", err)
		}
	}

	return nil
}
