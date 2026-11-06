// Package node decides what happens to each event the node receives.
package node


import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/internal/admission"
	"github.com/eduardokairalla1/EcoaNetwork/internal/orphan"
	"github.com/eduardokairalla1/EcoaNetwork/internal/p2p"
	pb "github.com/eduardokairalla1/EcoaNetwork/internal/protobuf"
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

	// OverLimit is an event whose author is over the admission limit.
	OverLimit
)


// maxOrphans is the entries the orphan buffer holds.
const maxOrphans = 10000


// orphanShare is the entries of the orphan buffer one origin may hold.
const orphanShare = maxOrphans / 10


// orphanAge is how long an orphan waits for its target.
const orphanAge = time.Hour


// admissionWindow is the period an author's events are counted over.
const admissionWindow = time.Hour


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
	OverLimit: "over limit",
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

	// admission limits the events one author may have accepted
	admission *admission.Limiter

	// orphans hold the events whose references have not resolved
	orphans *orphan.Buffer

	// ctx is the lifetime of the node, which the target fetches run under
	ctx context.Context

	// fetching holds the targets already being asked for
	fetching map[string]bool

	// fetchMu guards fetching
	fetchMu sync.Mutex

	// mu serializes events from the revocation check to the store
	mu sync.Mutex
}


// Outcome is what the node did with one received event.
type Outcome int


// New returns a node on the network of cfg, admitting admissionRate per hour.
func New(ctx context.Context,
         s *store.Store,
         cfg p2p.Config,
         maxEventSize int,
         admissionRate int) (*Node, error) {
	rules := envelope.Rules{
		Network: cfg.Network,
		MaxSize: maxEventSize,
		Now:     time.Now,
	}

	limiter := admission.New(admissionRate, admissionWindow, time.Now)
	orphans := orphan.New(maxOrphans, orphanShare, orphanAge, time.Now)

	n, err := build(ctx, s, rules, limiter, orphans)
	if err != nil {
		return nil, err
	}

	cfg.Validator = n.validate

	n.host, err = p2p.New(ctx, cfg)
	if err != nil {
		return nil, err
	}

	n.host.ServeSync(source{s})
	n.host.OnConnect(func(id peer.ID) { n.syncWith(ctx, id) })

	return n, nil
}


// build returns a node over s with no network, judging events by rules.
func build(ctx context.Context,
           s *store.Store,
           rules envelope.Rules,
           limiter *admission.Limiter,
           orphans *orphan.Buffer) (*Node, error) {
	schemas, err := schema.Load()
	if err != nil {
		return nil, err
	}

	identities := identity.NewTracker()

	if err := rebuildIdentity(s, identities); err != nil {
		return nil, fmt.Errorf("rebuild identities: %w", err)
	}

	n := &Node{
		store:      s,
		rules:      rules,
		schemas:    schemas,
		identities: identities,
		admission:  limiter,
		orphans:    orphans,
		ctx:        ctx,
		fetching:   map[string]bool{},
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


// Receive runs one event through the whole checklist, however it arrived,
// and hands back the orphans it frees.
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


// settle runs the checklist on a verified event from the revoked key on.
func (n *Node) settle(verified event.Event, arrival Arrival) (Outcome, error) {
	env := verified.Envelope()

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.identities.IsRevoked(env.Author) {
		return report(arrival, Rejected, &RevokedKeyError{Key: env.Author})
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

	if !n.admitted(env.Author, arrival) {
		return report(arrival, OverLimit, &OverLimitError{Author: env.Author})
	}

	return n.commit(verified, arrival)
}


// admitted applies the admission limit, which sync is exempt from.
func (n *Node) admitted(author string, arrival Arrival) bool {
	return arrival.Kind == Sync || n.admission.Allow(author)
}


// commit records a rotation or a revocation, stores the event and tells the
// indexer.
func (n *Node) commit(verified event.Event, arrival Arrival) (Outcome, error) {
	env := verified.Envelope()

	if identity.Moves(env.EventType) {
		if err := n.identities.Observe(env); err != nil {
			return report(arrival, Rejected, err)
		}
	}

	err := n.store.AddEvent(verified.ID(), verified.Raw(), n.rules.Now())
	if errors.Is(err, store.ErrDuplicate) {
		return Duplicate, nil
	}

	if err != nil {
		return report(arrival, Failed, err)
	}

	slog.Info("stored", "arrival", arrival.Kind, "bytes", len(verified.Raw()))

	n.notifyIndexer(verified.Raw())

	return Stored, nil
}


// notifyIndexer is where a stored event will be handed to the indexer. There
// is no indexer in this phase.
func (n *Node) notifyIndexer(_ []byte) {}


// offline reports whether n was built without a network.
func (n *Node) offline() bool {
	return n.host == nil
}


// hold parks an event until the events it names are stored.
func (n *Node) hold(verified event.Event,
                    missing []string,
                    arrival Arrival) Outcome {
	fetch := n.wantsTargets(arrival)

	n.orphans.Add(orphan.Entry{
		Raw:     verified.Raw(),
		Missing: missing,
		Origin:  originOf(arrival),
		Kind:    int(arrival.Kind),
	})

	slog.Info("orphaned",
	          "arrival", arrival.Kind,
	          "peer", arrival.Peer,
	          "missing", missing[0])

	if fetch {
		go n.fetchTargets(arrival.Peer, missing)
	}

	return Orphaned
}


// release hands every orphan waiting for id back to Receive, as the arrival it
// first came in by.
func (n *Node) release(id string) {
	for _, entry := range n.orphans.Release(id) {
		n.Receive(entry.Raw, arrivalOf(entry))
	}
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
		page, err := n.pullPage(ctx, p, cursor)
		if err != nil {
			return err
		}

		if !page.HasMore {
			return nil
		}

		cursor = page.NextCursor
	}
}


// pullPage receives the page of p after cursor and keeps its next cursor.
func (n *Node) pullPage(ctx context.Context,
                        p peer.ID,
                        cursor uint64) (*pb.GetEventsSinceResponse, error) {
	page, err := n.host.GetEventsSince(ctx, p, cursor)
	if err != nil {
		return nil, err
	}

	if page.HasMore && page.NextCursor <= cursor {
		return nil, fmt.Errorf("cursor stuck at %d with more to send", cursor)
	}

	if err := n.receiveFrom(p, page.Events); err != nil {
		return nil, err
	}

	if err := n.store.SetCursor(p.String(), page.NextCursor); err != nil {
		return nil, err
	}

	return page, nil
}


// receiveFrom hands each event to Receive, as an arrival from p.
func (n *Node) receiveFrom(p peer.ID, events [][]byte) error {
	arrival := Arrival{Kind: Sync, Peer: p}

	for _, raw := range events {
		outcome, err := n.Receive(raw, arrival)

		// A rejected event is the peer's fault; a failed one is ours, and
		// going on past it would lose it.
		if outcome == Failed {
			return fmt.Errorf("receive: %w", err)
		}
	}

	return nil
}


// wantsTargets reports whether the targets of an orphan are asked for.
func (n *Node) wantsTargets(arrival Arrival) bool {
	if n.offline() || arrival.Peer == "" {
		return false
	}

	return n.orphans.HasRoom(originOf(arrival))
}


// fetchTargets asks p for the ids not already being asked for.
func (n *Node) fetchTargets(p peer.ID, ids []string) {
	var wanted []string

	n.fetchMu.Lock()
	for _, id := range ids {
		if !n.fetching[id] {
			n.fetching[id] = true
			wanted = append(wanted, id)
		}
	}
	n.fetchMu.Unlock()

	defer func() {
		n.fetchMu.Lock()
		for _, id := range wanted {
			delete(n.fetching, id)
		}
		n.fetchMu.Unlock()
	}()

	if len(wanted) == 0 {
		return
	}

	if err := n.fetchMissing(n.ctx, p, wanted); err != nil {
		slog.Warn("fetch targets stopped", "peer", p, "reason", err)
	}
}


// fetchMissing asks p for the events under ids and receives those found.
func (n *Node) fetchMissing(ctx context.Context,
                            p peer.ID,
                            ids []string) error {
	resp, err := n.host.GetEvents(ctx, p, ids)
	if err != nil {
		return err
	}

	var found [][]byte

	for _, result := range resp.Results {
		if result.Status == pb.EventStatus_FOUND {
			found = append(found, result.Event)
		}
	}

	return n.receiveFrom(p, found)
}
