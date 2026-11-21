// Package config reads the indexer's settings from the environment.
package config


import (
	"fmt"
	"net/url"
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
	if err := checkDatabaseURL(databaseURL); err != nil {
		return Config{}, err
	}

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
