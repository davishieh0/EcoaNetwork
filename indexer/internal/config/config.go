// Package config reads the indexer's settings from the environment.
package config


import (
	"os"
)


// --- CONSTANTS ---

const (
	// defaultDatabaseURL is DATABASE_URL when unset: the local Postgres.
	defaultDatabaseURL = "postgres://indexer:goodpassword@127.0.0.1:5432/" +
	                     "ecoa_indexer?sslmode=disable"

	// defaultHTTPAddr is HTTP_ADDR when unset.
	defaultHTTPAddr = "127.0.0.1:8081"
)


// --- CODE ---

// Config is every setting the indexer reads at startup.
type Config struct {
	// DatabaseURL is the Postgres the projections live in.
	DatabaseURL string

	// HTTPAddr is the host:port the API answers on.
	HTTPAddr string
}


// Load reads the environment and applies the default of each unset variable.
func Load() (Config, error) {
	databaseURL := envOr("DATABASE_URL", defaultDatabaseURL)

	cfg := Config{
		DatabaseURL: databaseURL,
		HTTPAddr:    envOr("HTTP_ADDR", defaultHTTPAddr),
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
