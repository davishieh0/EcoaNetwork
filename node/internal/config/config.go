// Package config reads the node's settings from the environment.
package config


import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
)


// --- CONSTANTS ---

// defaultAdmissionRate is the events per author per hour a node accepts.
const defaultAdmissionRate = 100


const (
	// defaultSyncMaxStreams is SYNC_MAX_STREAMS when unset.
	defaultSyncMaxStreams = 32

	// defaultSyncMaxStreamsPerPeer is SYNC_MAX_STREAMS_PER_PEER when unset.
	defaultSyncMaxStreamsPerPeer = 2

	// defaultSyncEventsPerPeer is SYNC_EVENTS_PER_PEER when unset.
	defaultSyncEventsPerPeer = 4096

	// defaultConnLow is the connection count the manager trims down to.
	defaultConnLow = 32

	// defaultConnHigh is the connection count that starts the trimming.
	defaultConnHigh = 96
)


// --- CODE ---

// Config is every setting the node reads at startup.
type Config struct {
	// Network is the network this node belongs to.
	Network string

	// ListenAddr is the multiaddr the node listens on.
	ListenAddr string

	// GRPCAddr is the host:port the gateway submits events to.
	GRPCAddr string

	// BootstrapAddrs replaces the embedded peer list when not empty.
	BootstrapAddrs []string

	// BoltDBPath is the path of the bbolt file holding the event log.
	BoltDBPath string

	// PeerKeyPath is the file holding the node's libp2p private key.
	PeerKeyPath string

	// ConnLow is the connection count the manager trims down to.
	ConnLow int

	// ConnHigh is the connection count that makes the manager start trimming.
	ConnHigh int

	// MaxEventSize is this node's limit per event, in bytes.
	MaxEventSize int

	// AdmissionRate is the events accepted per author per hour, 0 for no limit.
	AdmissionRate int

	// SyncMaxStreams is the most sync streams served at once.
	SyncMaxStreams int

	// SyncMaxStreamsPerPeer is the most sync streams open at once per peer.
	SyncMaxStreamsPerPeer int

	// SyncEventsPerPeer is the most events served to one peer per minute.
	SyncEventsPerPeer int
}


// Load reads the environment and applies the default of each unset variable.
func Load() (Config, error) {
	size, err := loadMaxEventSize()
	if err != nil {
		return Config{}, err
	}

	rate, err := loadAdmissionRate()
	if err != nil {
		return Config{}, err
	}

	limits, err := loadSyncLimits()
	if err != nil {
		return Config{}, err
	}

	low, high, err := loadConnLimits()
	if err != nil {
		return Config{}, err
	}

	network := envOr("NETWORK", "ecoa-dev")
	if !envelope.ValidProtocol(network) {
		return Config{}, fmt.Errorf("NETWORK: %q is not ecoa or ecoa-<name>",
		                            network)
	}

	cfg := Config{
		Network:        network,
		ListenAddr:     envOr("LISTEN_ADDR", "/ip4/127.0.0.1/tcp/4001"),
		GRPCAddr:       envOr("GRPC_ADDR", "127.0.0.1:50051"),
		BootstrapAddrs: splitAddrs(os.Getenv("BOOTSTRAP_ADDRS")),
		BoltDBPath:     envOr("BOLT_DB_PATH", "data/bbolt/bolt.db"),
		PeerKeyPath:    envOr("PEER_KEY_PATH", "data/peer.key"),
		ConnLow:        low,
		ConnHigh:       high,
		MaxEventSize:   size,
		AdmissionRate:  rate,

		SyncMaxStreams:        limits[0],
		SyncMaxStreamsPerPeer: limits[1],
		SyncEventsPerPeer:     limits[2],
	}

	return cfg, nil
}


// envOr returns the variable named key, or fallback when it is unset or empty.
func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}


// splitAddrs turns a comma-separated list into its items.
func splitAddrs(list string) []string {
	if list == "" {
		return nil
	}

	return strings.Split(list, ",")
}


// loadMaxEventSize reads MAX_EVENT_SIZE, or an error outside the range.
func loadMaxEventSize() (int, error) {
	value := os.Getenv("MAX_EVENT_SIZE")
	if value == "" {
		return envelope.MaxSize, nil
	}

	size, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("MAX_EVENT_SIZE: %w", err)
	}

	if size <= 0 || size > envelope.MaxSize {
		return 0, fmt.Errorf("MAX_EVENT_SIZE: %d is outside 1..%d",
		                     size,
		                     envelope.MaxSize)
	}

	return size, nil
}


// loadAdmissionRate reads ADMISSION_RATE, or an error below zero.
func loadAdmissionRate() (int, error) {
	value := os.Getenv("ADMISSION_RATE")
	if value == "" {
		return defaultAdmissionRate, nil
	}

	rate, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("ADMISSION_RATE: %w", err)
	}

	if rate < 0 {
		return 0, fmt.Errorf("ADMISSION_RATE: %d is below 0", rate)
	}

	return rate, nil
}


// loadSyncLimits reads the three sync limits, in the order of Config's fields.
func loadSyncLimits() ([3]int, error) {
	names := [3]string{
		"SYNC_MAX_STREAMS",
		"SYNC_MAX_STREAMS_PER_PEER",
		"SYNC_EVENTS_PER_PEER",
	}
	defaults := [3]int{
		defaultSyncMaxStreams,
		defaultSyncMaxStreamsPerPeer,
		defaultSyncEventsPerPeer,
	}

	var limits [3]int

	for i, name := range names {
		value, err := positiveInt(name, defaults[i])
		if err != nil {
			return limits, err
		}

		limits[i] = value
	}

	return limits, nil
}


// loadConnLimits reads CONN_LOW and CONN_HIGH, with low <= high.
func loadConnLimits() (int, int, error) {
	low, err := positiveInt("CONN_LOW", defaultConnLow)
	if err != nil {
		return 0, 0, err
	}

	high, err := positiveInt("CONN_HIGH", defaultConnHigh)
	if err != nil {
		return 0, 0, err
	}

	if low > high {
		return 0, 0, fmt.Errorf("CONN_LOW %d and CONN_HIGH %d: want "+
		                        "low <= high",
		                        low,
		                        high)
	}

	return low, high, nil
}


// positiveInt reads the variable named key as an integer above zero.
func positiveInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}

	if n <= 0 {
		return 0, fmt.Errorf("%s: %d is not above 0", key, n)
	}

	return n, nil
}
