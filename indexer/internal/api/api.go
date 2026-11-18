// Package api is the indexer's HTTP API: read only, called by the gateway.
package api


import (
	"net/http"
)


// --- CODE ---

// Handler returns the router of every endpoint the API answers.
//
// Each route lives in a file of its own; this is the one list of them all.
func Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)

	return mux
}
