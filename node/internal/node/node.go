// Package node decides what happens to each event the node receives.
package node


import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/eduardokairalla1/EcoaNetwork/internal/p2p"
	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
)


// --- CODE ---

type Node struct {
	store *store.Store
	host  *p2p.Host
}


func New(ctx context.Context, s *store.Store, cfg p2p.Config) (*Node, error) {
	p, err := p2p.New(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &Node{store: s, host: p}, nil
}


// Handle stores events from every door under their claimed, unverified id.
func (n *Node) Handle(raw []byte) error {
	var env struct {
		EventID string `json:"eventId"`
	}

	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}

	return n.store.AddEvent(env.EventID, raw)
}


// Run pulls events from the topic until ctx is cancelled.
func (n *Node) Run(ctx context.Context) {
	for {
		raw, _, err := n.host.ReceiveEvent(ctx)
		if err != nil {
			return
		}

		if err := n.Handle(raw); err != nil {
			// A duplicate is dropped silently.
			if !errors.Is(err, store.ErrDuplicate) {
				slog.Warn("handle", "err", err)
			}
			continue
		}

		slog.Info("stored", "bytes", len(raw))
	}
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
