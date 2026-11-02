// Package config reads the node's settings from the environment.
package config


import (
	"os"
	"strings"
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
}


// Load reads the environment and applies the default of each unset variable.
func Load() (Config, error) {
	cfg := Config{
		Network:        envOr("NETWORK", "ecoa-dev"),
		ListenAddr:     envOr("LISTEN_ADDR", "/ip4/127.0.0.1/tcp/4001"),
		GRPCAddr:       envOr("GRPC_ADDR", "127.0.0.1:50051"),
		BootstrapAddrs: splitAddrs(os.Getenv("BOOTSTRAP_ADDRS")),
		BoltDBPath:     envOr("BOLT_DB_PATH", "data/bbolt/bolt.db"),
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
