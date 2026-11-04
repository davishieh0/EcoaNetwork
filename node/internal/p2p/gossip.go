package p2p


import (
	"context"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)


// --- CONSTANTS ---

// pollInterval is how often WaitForPeers rechecks the topic.
const pollInterval = 50 * time.Millisecond


// --- CODE ---

// Topic returns the Gossipsub topic for a network: one topic per network,
// carrying every event type.
func Topic(network string) string {
	return "/" + network + "/events/1"
}


// joinTopic starts Gossipsub and subscribes to the network's topic.
func joinTopic(
	ctx context.Context,
	h host.Host,
	network string,
) (*pubsub.Topic, *pubsub.Subscription, error) {
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, nil, err
	}

	name := Topic(network)

	topic, err := ps.Join(name)
	if err != nil {
		return nil, nil, err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, nil, err
	}

	return topic, sub, nil
}


// ReceiveEvent returns the next event on the topic and the peer it came from.
func (p *Host) ReceiveEvent(ctx context.Context) ([]byte, peer.ID, error) {
	msg, err := p.sub.Next(ctx)
	if err != nil {
		return nil, "", err
	}

	return msg.Data, msg.ReceivedFrom, nil
}


// WaitForPeers blocks until the topic is ready to carry a publish, or ctx is
// cancelled. Gossipsub drops a message with no route instead of queueing it.
//
// A peer appears in ListPeers as soon as its subscription arrives, but it only
// starts relaying once the mesh is grafted, which happens on the next
// heartbeat. Publishing in between is silently lost, so waiting for a peer is
// necessary and not sufficient: one heartbeat has to pass as well.
func (p *Host) WaitForPeers(ctx context.Context) error {
	for len(p.topic.ListPeers()) == 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(pubsub.GossipSubHeartbeatInterval):
		return nil
	}
}


// Publish sends an event to the topic.
func (p *Host) Publish(ctx context.Context, raw []byte) error {
	return p.topic.Publish(ctx, raw)
}
