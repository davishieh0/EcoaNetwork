// Package config reads the indexer's settings from the environment.
package config


import (
	"os"
)


// --- CONSTANTS ---

// defaultHTTPAddr is HTTP_ADDR when unset.
const defaultHTTPAddr = "127.0.0.1:8081"


// --- CODE ---

// Config is every setting the indexer reads at startup.
type Config struct {
	// HTTPAddr is the host:port the API answers on.
	HTTPAddr string
}


// Load reads the environment and applies the default of each unset variable.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr: envOr("HTTP_ADDR", defaultHTTPAddr),
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
