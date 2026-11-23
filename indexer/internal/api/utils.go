package api


import (
	"encoding/json"
	"log/slog"
	"net/http"
)


// --- CODE ---

// writeJSON sends body as JSON with the given status.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Warn("write response", "err", err)
	}
}
