// Package config reads the indexer's settings from the environment.
package config


import (
	"fmt"
	"net/url"
	"os"
	"strings"
)


// --- CONSTANTS ---

const (
	// defaultDatabaseURL is DATABASE_URL when unset: the local Postgres.
	defaultDatabaseURL = "postgres://indexer:goodpassword@127.0.0.1:5432/" +
	                     "ecoa_indexer?sslmode=disable"

	// defaultNodeAddrs is NODE_ADDRS when unset: one local node.
	defaultNodeAddrs = "127.0.0.1:50051"

	// defaultHTTPAddr is HTTP_ADDR when unset.
	defaultHTTPAddr = "127.0.0.1:8081"
)


// --- CODE ---

// Config is every setting the indexer reads at startup.
type Config struct {
	// DatabaseURL is the Postgres the projections live in.
	DatabaseURL string

	// NodeAddrs is the gRPC address of every node the projector follows.
	NodeAddrs []string

	// HTTPAddr is the host:port the API answers on.
	HTTPAddr string

	// ModeratorsFile is the JSON file listing the moderators whose labels
	// count, "" for none.
	ModeratorsFile string

	// Ranking is how the API orders an entity's reviews: newest or
	// endorsed.
	Ranking string
}


// Load reads the environment and applies the default of each unset variable.
func Load() (Config, error) {
	databaseURL := envOr("DATABASE_URL", defaultDatabaseURL)
	if err := checkDatabaseURL(databaseURL); err != nil {
		return Config{}, err
	}

	nodes := splitAddrs(envOr("NODE_ADDRS", defaultNodeAddrs))
	if len(nodes) == 0 {
		return Config{}, fmt.Errorf("NODE_ADDRS: no address in the list")
	}

	cfg := Config{
		DatabaseURL: databaseURL,
		NodeAddrs:   nodes,
		HTTPAddr:    envOr("HTTP_ADDR", defaultHTTPAddr),

		ModeratorsFile: os.Getenv("MODERATORS_FILE"),
		Ranking: envOr("RANKING", "newest"),
	}

	if cfg.Ranking != "newest" && cfg.Ranking != "endorsed" {
		return Config{}, fmt.Errorf("RANKING: %q is neither newest nor "+
		                            "endorsed", cfg.Ranking)
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


// splitAddrs splits a comma-separated list, dropping blank entries.
func splitAddrs(list string) []string {
	var addrs []string

	for _, addr := range strings.Split(list, ",") {
		if addr = strings.TrimSpace(addr); addr != "" {
			addrs = append(addrs, addr)
		}
	}

	return addrs
}


// checkDatabaseURL refuses a DATABASE_URL that is not a Postgres URL.
func checkDatabaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		// the error quotes the URL, password included, so it is not wrapped
		return fmt.Errorf("DATABASE_URL: not a valid URL")
	}

	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return fmt.Errorf("DATABASE_URL: scheme %q, want postgres", u.Scheme)
	}

	return nil
}
