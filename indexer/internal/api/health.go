package api


import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/respond"
)


// --- CONSTANTS ---

// pingTimeout bounds how long /health waits for the database.
const pingTimeout = 2 * time.Second


// --- CODE ---

// pinger is the part of the database /health asks.
type pinger interface {
	Ping(ctx context.Context) error
}


// health answers that the API is up and the database answers.
func health(db pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), pingTimeout)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			slog.Warn("health: database unavailable", "err", err)
			respond.Error(w, http.StatusServiceUnavailable,
			              "database_unavailable_error",
			              "The database does not answer.")
			return
		}

		respond.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
