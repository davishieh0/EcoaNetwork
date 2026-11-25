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
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
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


// redialEvery is how often a node with few peers dials the ones it knows, so
// one that came back while its peers were away, or the other way round,
// finds them again on its own.
var redialEvery = 30 * time.Second


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

	// OnSeen is called with each multiaddr a peer listens on, whichever side
	// opened the connection, as the peer reports it through identify.
	OnSeen func(addr string)

	// Redial returns the peers to try while this node has fewer than ConnLow
	// connected; nil tries none after startup.
	Redial func() []string

	// KeyPath is the file holding the peer's private key, empty for none.
	KeyPath string

	// ConnLow is the connection count the manager trims down to.
	ConnLow int

	// ConnHigh is the connection count that makes the manager start trimming.
	ConnHigh int

	// DeferConnect skips initial bootstrap dialing in New so the caller can
	// register ServeSync and OnConnect before calling Connect.
	DeferConnect bool
}


// Host is this node's presence on the network.
type Host struct {
	host  host.Host
	topic *pubsub.Topic
	sub   *pubsub.Subscription

	// limiter decides which sync streams get served
	limiter *syncLimiter

	// dial stores the startup dial configuration for Connect
	cfg Config

	// meshReady tracks whether Gossipsub has completed at least one heartbeat
	// with peers on the topic since the last empty state.
	meshReady atomic.Bool
}


// New builds the peer: host and Gossipsub topic. Unless cfg.DeferConnect is
// true, it also connects to bootstrap peers and starts the redial loop.
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

	if cfg.OnSeen != nil {
		if err := rememberIdentified(ctx, h, cfg.OnSeen); err != nil {
			h.Close()
			return nil, err
		}
	}

	topic, sub, err := joinTopic(ctx, h, cfg.Network, cfg.Validator)
	if err != nil {
		h.Close()
		return nil, err
	}

	limiter := newSyncLimiter(cfg.Sync, time.Now)
	p := &Host{host: h, topic: topic, sub: sub, limiter: limiter, cfg: cfg}

	if !cfg.DeferConnect {
		p.Connect(ctx)
	}

	return p, nil
}


// Connect dials known and bootstrap peers and starts the background redial loop.
func (p *Host) Connect(ctx context.Context) {
	connectBootstrap(ctx, p.host, dialOrder(p.cfg))

	if p.cfg.Redial != nil {
		go keepConnected(ctx, p.host, max(p.cfg.ConnLow, 1), p.cfg.Redial)
	}
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


// rememberIdentified reports the addresses each peer listens on, once
// identify has told them, until ctx ends.
//
// Not the address of the connection: for a peer that dialed this node, that
// is the port its system picked for the call, which nothing listens on once
// the connection closes.
func rememberIdentified(ctx context.Context,
                        h host.Host,
                        onSeen func(addr string)) error {
	sub, err := h.EventBus().Subscribe(new(event.EvtPeerIdentificationCompleted))
	if err != nil {
		return fmt.Errorf("subscribe to identify: %w", err)
	}

	go func() {
		defer sub.Close()

		for {
			select {
			case e, ok := <-sub.Out():
				if !ok {
					return
				}

				identified := e.(event.EvtPeerIdentificationCompleted)

				hasNonLoopback := false
				for _, a := range identified.ListenAddrs {
					if dialable(a) && !manet.IsIPLoopback(a) {
						hasNonLoopback = true
						break
					}
				}

				for _, a := range identified.ListenAddrs {
					if !dialable(a) {
						continue
					}
					if hasNonLoopback && manet.IsIPLoopback(a) {
						continue
					}
					onSeen(a.String() + "/p2p/" + identified.Peer.String())
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}


// dialable reports whether another node can dial a: not an unspecified
// address such as 0.0.0.0, and not a relay.
func dialable(a multiaddr.Multiaddr) bool {
	if manet.IsIPUnspecified(a) {
		return false
	}

	_, err := a.ValueForProtocol(multiaddr.P_CIRCUIT)

	return err != nil
}


// keepConnected dials the peers addrs returns that are not connected, every
// redialEvery while this node has fewer than below, until ctx ends.
func keepConnected(ctx context.Context,
                   h host.Host,
                   below int,
                   addrs func() []string) {
	ticker := time.NewTicker(redialEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}

		if len(h.Network().Peers()) >= below {
			continue
		}

		redial(ctx, h, addrs())
	}
}


// redial dials each of addrs whose peer is not connected, each under its
// own timeout, and logs the peers it reached.
func redial(ctx context.Context, h host.Host, addrs []string) {
	for _, addr := range addrs {
		info, err := peer.AddrInfoFromString(addr)
		if err != nil || h.Network().Connectedness(info.ID) == network.Connected {
			continue
		}

		dialCtx, cancel := context.WithTimeout(ctx, dialTimeout)
		err = h.Connect(dialCtx, *info)
		cancel()

		if err == nil {
			slog.Info("redialed", "peer", info.ID.String())
		}
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

		if info.ID == h.ID() || h.Network().Connectedness(info.ID) == network.Connected {
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
