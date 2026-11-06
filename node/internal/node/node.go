// Package node decides what happens to each event the node receives.
package node


import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/internal/p2p"
	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/event"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/schema"
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
)


// --- GLOBALS ---

// outcomeNames is the log name of each outcome.
var outcomeNames = map[Outcome]string{
	Stored:    "stored",
	Duplicate: "duplicate",
	Ignored:   "ignored",
	Rejected:  "rejected",
	Failed:    "failed",
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

	n, err := build(s, rules)
	if err != nil {
		return nil, err
	}

	n.host, err = p2p.New(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return n, nil
}


// build returns a node over s with no network, judging events by rules.
func build(s *store.Store, rules envelope.Rules) (*Node, error) {
	schemas, err := schema.Load()
	if err != nil {
		return nil, err
	}

	n := &Node{
		store:   s,
		rules:   rules,
		schemas: schemas,
	}

	return n, nil
}


// Run pulls events from the topic until ctx is cancelled.
func (n *Node) Run(ctx context.Context) {
	for {
		raw, from, err := n.host.ReceiveEvent(ctx)
		if err != nil {
			return
		}

		n.Receive(raw, Arrival{Kind: Gossip, Peer: from})
	}
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

	err = n.store.AddEvent(verified.ID(), verified.Raw(), n.rules.Now())
	if errors.Is(err, store.ErrDuplicate) {
		return Duplicate, nil
	}

	if err != nil {
		return report(arrival, Failed, err)
	}

	slog.Info("stored", "arrival", arrival.Kind, "bytes", len(raw))
	return Stored, nil
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
