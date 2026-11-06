package p2p


import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"slices"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
)


// --- CONSTANTS ---

const (
	// keyMode is the permission of the peer key file: owner only.
	keyMode = 0o600

	// resourceMemory is the memory, in bytes, the resource limits scale from.
	resourceMemory = 256 << 20

	// resourceFDs is the file descriptor count the resource limits scale from.
	resourceFDs = 512
)


// --- GLOBALS ---

// dialTimeout bounds each connection attempt at startup.
var dialTimeout = 5 * time.Second


// --- CODE ---

// Config is what the node needs to join the network.
type Config struct {
	// Network is the network name, such as "ecoa-dev". It names the topic.
	Network string

	// Listen is the multiaddr this node listens on.
	Listen string

	// BootstrapAddrs is the list of peer multiaddrs to connect to on startup.
	BootstrapAddrs []string

	// Validator judges each message before Gossipsub relays it.
	Validator Validator

	// Sync holds the peers that open sync streams to these ceilings.
	Sync SyncLimits

	// KnownAddrs are the peers of the last session.
	KnownAddrs []string

	// OnDialed is called with the multiaddr of each peer this node reaches.
	OnDialed func(addr string)

	// KeyPath is the file holding the peer's private key, empty for none.
	KeyPath string

	// ConnLow is the connection count the manager trims down to.
	ConnLow int

	// ConnHigh is the connection count that makes the manager start trimming.
	ConnHigh int
}


// Host is this node's presence on the network.
type Host struct {
	host  host.Host
	topic *pubsub.Topic
	sub   *pubsub.Subscription

	// limiter decides which sync streams get served
	limiter *syncLimiter
}


// New builds the peer: host, bootstrap connections, Gossipsub and the
// subscription. It does not start receiving events; call Run for that.
func New(ctx context.Context, cfg Config) (*Host, error) {
	if cfg.Validator == nil {
		return nil, errors.New("p2p: Config.Validator is required")
	}

	opts, err := hostOptions(cfg)
	if err != nil {
		return nil, err
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, err
	}

	if cfg.OnDialed != nil {
		h.Network().Notify(remember(cfg.OnDialed))
	}

	connectBootstrap(ctx, h, dialOrder(cfg))

	topic, sub, err := joinTopic(ctx, h, cfg.Network, cfg.Validator)
	if err != nil {
		h.Close()
		return nil, err
	}

	limiter := newSyncLimiter(cfg.Sync, time.Now)

	return &Host{host: h, topic: topic, sub: sub, limiter: limiter}, nil
}


// hostOptions returns the libp2p options: address, identity and limits.
func hostOptions(cfg Config) ([]libp2p.Option, error) {
	cm, err := connmgr.NewConnManager(cfg.ConnLow, cfg.ConnHigh)
	if err != nil {
		return nil, fmt.Errorf("connection manager: %w", err)
	}

	limits := rcmgr.DefaultLimits
	libp2p.SetDefaultServiceLimits(&limits)

	scaled := limits.Scale(resourceMemory, resourceFDs)

	rm, err := rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(scaled))
	if err != nil {
		return nil, fmt.Errorf("resource manager: %w", err)
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(cfg.Listen),
		libp2p.ConnectionManager(cm),
		libp2p.ResourceManager(rm),
	}

	if cfg.KeyPath == "" {
		return opts, nil
	}

	key, err := loadOrCreateKey(cfg.KeyPath)
	if err != nil {
		return nil, err
	}

	return append(opts, libp2p.Identity(key)), nil
}


// loadOrCreateKey reads the private key at path, creating it when absent.
func loadOrCreateKey(path string) (crypto.PrivKey, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return createKey(path)
	}

	if err != nil {
		return nil, fmt.Errorf("read peer key %s: %w", path, err)
	}

	key, err := crypto.UnmarshalPrivateKey(raw)
	if err != nil {
		return nil, fmt.Errorf("peer key %s is corrupt: %w", path, err)
	}

	return key, nil
}


// createKey generates an Ed25519 key and writes it to path, never over a file.
func createKey(path string) (crypto.PrivKey, error) {
	key, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}

	raw, err := crypto.MarshalPrivateKey(key)
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, keyMode)
	if err != nil {
		return nil, fmt.Errorf("create peer key %s: %w", path, err)
	}

	_, err = file.Write(raw)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}

	if err != nil {
		return nil, fmt.Errorf("write peer key %s: %w", path, err)
	}

	return key, nil
}


// dialOrder returns the peers of the last session, then the bootstrap list.
func dialOrder(cfg Config) []string {
	return append(slices.Clone(cfg.KnownAddrs), cfg.BootstrapAddrs...)
}


// remember reports each peer this node dialed.
func remember(onConnected func(addr string)) *network.NotifyBundle {
	return &network.NotifyBundle{
		ConnectedF: func(_ network.Network, c network.Conn) {
			if c.Stat().Direction != network.DirOutbound {
				return
			}

			onConnected(c.RemoteMultiaddr().String() +
			            "/p2p/" + c.RemotePeer().String())
		},
	}
}


// connectBootstrap dials every address, each under its own timeout.
func connectBootstrap(ctx context.Context, h host.Host, addrs []string) {
	for _, addr := range addrs {
		info, err := peer.AddrInfoFromString(addr)
		if err != nil {
			slog.Warn("bootstrap unusable", "addr", addr, "err", err)
			continue
		}

		dialCtx, cancel := context.WithTimeout(ctx, dialTimeout)
		err = h.Connect(dialCtx, *info)
		cancel()

		if err != nil {
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
