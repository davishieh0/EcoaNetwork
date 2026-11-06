package p2p

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)


// --- CONSTANTS ---

// loopback lets the kernel pick a free port, so tests never collide.
const loopback = "/ip4/127.0.0.1/tcp/0"


// deadEnd is a well-formed multiaddr nothing listens on: parsing succeeds and
// dialing fails, which is the path a bootstrap peer being down takes.
const deadEnd = "/ip4/127.0.0.1/tcp/1/p2p/" +
	"12D3KooWL8WucnuP3YZ7yh2E8bNggmBsqhGs4PxXrxTZaEF7DH1j"


// --- CODE ---

// safeBuffer collects log output written from another goroutine.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}


func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}


func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}


// acceptAll relays every message.
func acceptAll(context.Context, peer.ID, []byte) pubsub.ValidationResult {
	return pubsub.ValidationAccept
}


// config returns the settings of a test peer that bootstraps from addrs.
func config(addrs ...string) Config {
	return Config{
		Network:        "ecoa-test",
		Listen:         loopback,
		BootstrapAddrs: addrs,
		Validator:      acceptAll,
	}
}


func TestNew(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	p, err := New(ctx, config())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()

	if p.topic.String() != Topic("ecoa-test") {
		t.Errorf("joined %q, want %q", p.topic, Topic("ecoa-test"))
	}
}


// TestNewRejectsBadListenAddr pins the first error path: a listen address the
// host cannot bind must surface, not start a half-built peer.
func TestNewRejectsBadListenAddr(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bad := config()
	bad.Listen = "nonsense"

	_, err := New(ctx, bad)
	if err == nil {
		t.Fatal("an unbindable listen address should fail")
	}
}


// TestNewSurvivesDeadBootstrap checks that a bootstrap peer being down does
// not stop this node: the network may be empty and it still has to start.
func TestNewSurvivesDeadBootstrap(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	p, err := New(ctx, config(deadEnd))
	if err != nil {
		t.Fatalf("New with a dead bootstrap peer: %v", err)
	}
	defer p.Close()
}


func TestBootstrapListFallsBackToTheEmbeddedList(t *testing.T) {
	// Arrange
	previous := DefaultBootstrapAddrs
	DefaultBootstrapAddrs = []string{"embedded"}
	t.Cleanup(func() { DefaultBootstrapAddrs = previous })

	// Act
	got := bootstrapList(Config{})

	// Assert
	if !reflect.DeepEqual(got, []string{"embedded"}) {
		t.Fatalf("bootstrapList = %v, want the embedded list", got)
	}
}


func TestBootstrapListPrefersTheConfiguredList(t *testing.T) {
	// Arrange
	previous := DefaultBootstrapAddrs
	DefaultBootstrapAddrs = []string{"embedded"}
	t.Cleanup(func() { DefaultBootstrapAddrs = previous })

	cfg := Config{BootstrapAddrs: []string{"configured"}}

	// Act
	got := bootstrapList(cfg)

	// Assert
	if !reflect.DeepEqual(got, []string{"configured"}) {
		t.Fatalf("bootstrapList = %v, want the configured list", got)
	}
}


// TestConnectBootstrapSkipsUnusable covers both ways an entry can fail: one
// that does not parse and one that parses but refuses the dial. Neither may
// stop the caller, so the function returns nothing to check and the log is
// the evidence.
func TestConnectBootstrapSkipsUnusable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	h, err := libp2p.New(libp2p.ListenAddrStrings(loopback))
	if err != nil {
		t.Fatalf("host: %v", err)
	}
	defer h.Close()

	var seen safeBuffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&seen, nil)))
	defer slog.SetDefault(previous)

	connectBootstrap(ctx, h, []string{"not-a-multiaddr", deadEnd})

	logged := seen.String()
	if !strings.Contains(logged, "bootstrap unusable") {
		t.Errorf("an unparseable addr was not reported: %q", logged)
	}

	if !strings.Contains(logged, "bootstrap unreachable") {
		t.Errorf("an undialable addr was not reported: %q", logged)
	}
}


// TestAddrs checks the round trip the comment promises: what a node reports
// has to be usable as another node's bootstrap entry, which means carrying
// the peer ID and not just the transport address.
func TestAddrs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	p, err := New(ctx, config())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()

	addrs := p.Addrs()
	if len(addrs) == 0 {
		t.Fatal("a listening node reported no addresses")
	}

	for _, addr := range addrs {
		if !strings.Contains(addr, "/p2p/") {
			t.Errorf("addr %q carries no peer ID", addr)
		}
	}
}


// TestClose checks that shutting down releases the port, which is what lets
// a node be restarted on the same address.
func TestClose(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	p, err := New(ctx, config())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if len(p.host.Addrs()) != 0 {
		t.Error("a closed node is still listening")
	}
}


func TestNewKeepsThePeerIDAcrossRestarts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	cfg := Config{
		Network:   "ecoa-test",
		Listen:    loopback,
		Validator: acceptAll,
		KeyPath:   filepath.Join(t.TempDir(), "peer.key"),
	}

	first, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	id := first.host.ID()
	first.Close()

	// Act
	second, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("New after restart: %v", err)
	}
	defer second.Close()

	// Assert
	if second.host.ID() != id {
		t.Fatalf("peer ID = %s after restart, want %s", second.host.ID(), id)
	}
}


func TestLoadOrCreateKeyCreatesTheFileForOwnerOnly(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "peer.key")

	// Act
	_, err := loadOrCreateKey(path)

	// Assert
	if err != nil {
		t.Fatalf("loadOrCreateKey: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("key file: %v", err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Fatalf("key file mode = %o, want 600", info.Mode().Perm())
	}
}


func TestNewNamesTheCorruptKeyFile(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	path := filepath.Join(t.TempDir(), "peer.key")
	if err := os.WriteFile(path, []byte("not a key"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Act
	_, err := New(ctx, Config{
		Network:   "ecoa-test",
		Listen:    loopback,
		Validator: acceptAll,
		KeyPath:   path,
	})

	// Assert
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("New = %v, want an error naming %s", err, path)
	}
}
