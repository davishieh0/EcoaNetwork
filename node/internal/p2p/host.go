package p2p


import (
	"context"
	"log/slog"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)


// --- GLOBALS ---

// DefaultBootstrapAddrs is the bootstrap list shipped with the node. Empty
// until a public node exists; the operator overrides it from configuration.
var DefaultBootstrapAddrs []string


// --- CODE ---

// Config is what the node needs to join the network.
type Config struct {
	// Network is the network name, such as "ecoa-dev". It names the topic.
	Network string

	// Listen is the multiaddr this node listens on.
	Listen string

	// BootstrapAddrs is the list of peer multiaddrs to connect to on startup.
	BootstrapAddrs []string
}


// Host is this node's presence on the network.
type Host struct {
	host  host.Host
	topic *pubsub.Topic
	sub   *pubsub.Subscription
}


// New builds the peer: host, bootstrap connections, Gossipsub and the
// subscription. It does not start receiving events; call Run for that.
func New(ctx context.Context, cfg Config) (*Host, error) {
	h, err := libp2p.New(libp2p.ListenAddrStrings(cfg.Listen))
	if err != nil {
		return nil, err
	}

	connectBootstrap(ctx, h, bootstrapList(cfg))

	topic, sub, err := joinTopic(ctx, h, cfg.Network)
	if err != nil {
		h.Close()
		return nil, err
	}

	return &Host{host: h, topic: topic, sub: sub}, nil
}


// bootstrapList returns the configured peers, or else the embedded list.
func bootstrapList(cfg Config) []string {
	if len(cfg.BootstrapAddrs) > 0 {
		return cfg.BootstrapAddrs
	}

	return DefaultBootstrapAddrs
}


// connectBootstrap dials every bootstrap address. A peer being down must not
// stop this node from starting, so a failure is logged and skipped.
func connectBootstrap(ctx context.Context, h host.Host, addrs []string) {
	for _, addr := range addrs {
		info, err := peer.AddrInfoFromString(addr)
		if err != nil {
			slog.Warn("bootstrap unusable", "addr", addr, "err", err)
			continue
		}

		if err := h.Connect(ctx, *info); err != nil {
			slog.Warn("bootstrap unreachable", "addr", addr, "err", err)
		}
	}
}


// Addrs returns this node's full multiaddrs, ready to be used as another
// node's bootstrap entries.
func (p *Host) Addrs() []string {
	var addrs []string
	for _, a := range p.host.Addrs() {
		addrs = append(addrs, a.String()+"/p2p/"+p.host.ID().String())
	}
	return addrs
}


// Close shuts down the host.
func (p *Host) Close() error {
	return p.host.Close()
}
