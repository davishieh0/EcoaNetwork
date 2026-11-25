// Package api is the indexer's HTTP API: read only, called by the gateway.
package api


import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/claim"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/entity"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/event"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/identities"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/profile"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/response"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/review"
	"github.com/jackc/pgx/v5/pgxpool"
)


// --- CONSTANTS ---

// shutdownTimeout bounds the wait for open requests when the API stops.
const shutdownTimeout = 5 * time.Second


// --- CODE ---

// Serve answers the API on addr, reading from pool and acting on the labels
// of the moderators policy follows, until ctx ends.
func Serve(ctx context.Context,
           addr string,
           pool *pgxpool.Pool,
           policy label.Policy,
           verifiers claim.Policy,
           ranking review.Ranking) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           Handler(pool, policy, verifiers, ranking),
		ReadHeaderTimeout: 5 * time.Second,
	}

	failed := make(chan error, 1)

	go func() {
		failed <- srv.ListenAndServe()
	}()

	slog.Info("http", "addr", addr)

	select {
	case err := <-failed:
		return fmt.Errorf("serve http %s: %w", addr, err)
	case <-ctx.Done():
	}

	shutdown, cancel := context.WithTimeout(context.Background(),
		shutdownTimeout)
	defer cancel()

	return srv.Shutdown(shutdown)
}


// Handler returns the router of every endpoint the API answers.
//
// Each route lives in a file of its own; this is the one list of them all.
func Handler(pool *pgxpool.Pool,
             policy label.Policy,
             verifiers claim.Policy,
             ranking review.Ranking) http.Handler {
	moderation := label.NewModeration(policy, pool)
	recognition := claim.NewRecognition(verifiers, pool)

	eventRepo := event.NewRepository(pool)
	events := event.NewHandler(eventRepo, moderation)
	entities := entity.NewHandler(entity.NewRepository(pool), moderation,
	                              recognition)
	profiles := profile.NewHandler(profile.NewRepository(pool), moderation)
	keys := identities.NewHandler(identities.NewRepository(pool), recognition)
	reviews := review.NewHandler(review.NewRepository(pool), moderation,
	                             recognition, ranking).WithEvents(eventRepo)
	responses := response.NewHandler(response.NewRepository(pool),
	                                 review.NewRepository(pool),
	                                 moderation,
	                                 recognition)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health(pool))
	mux.HandleFunc("GET /nodes", nodes(pool))
	mux.HandleFunc("GET /events/{eventId}", events.Get)
	mux.HandleFunc("GET /entities", entities.Search)
	mux.HandleFunc("GET /entities/{entity}", entities.Get)
	mux.HandleFunc("GET /entities/{entity}/reviews", reviews.ListByEntity)
	mux.HandleFunc("GET /reviews/{eventId}", reviews.Get)
	mux.HandleFunc("GET /reviews/{eventId}/responses", responses.ListByReview)
	mux.HandleFunc("GET /profiles", profiles.List)
	mux.HandleFunc("GET /profiles/{did}", profiles.Get)
	mux.HandleFunc("GET /identities/{did}", keys.Get)

	return mux
}
