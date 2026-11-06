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

	// MaxEventSize is this node's limit per event, in bytes.
	MaxEventSize int

	// AdmissionRate is the events accepted per author per hour, 0 for no limit.
	AdmissionRate int
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
		MaxEventSize:   size,
		AdmissionRate:  rate,
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
