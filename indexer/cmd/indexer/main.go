// Command indexer follows Ecoa nodes and serves what it projects over HTTP.
package main


import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/api"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/claim"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/config"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/database"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/projector"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/review"
)


// --- GLOBALS ---

// commands maps each subcommand to what it runs.
var commands = map[string]func(context.Context, config.Config) error{
	"migrate": migrate,
	"project": project,
	"serve":   serve,
	"rebuild": notBuilt,
}


// errNotBuilt is what a subcommand that does not exist yet returns.
var errNotBuilt = errors.New("not built yet")


// --- CODE ---

func main() {
	// The composition root installs the default logger.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(os.Args[1:]); err != nil {
		slog.Error("indexer", "err", err)
		os.Exit(1)
	}
}


// run starts the subcommand named in args and holds until it ends.
func run(args []string) error {

	// subcommand is not built yet: return an error
	if len(args) != 1 || commands[args[0]] == nil {
		return fmt.Errorf("want one subcommand: migrate, project, serve " +
			"or rebuild")
	}

	// load the config file
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Ctrl+C and the SIGTERM a container receives both end the subcommand.
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGTERM)
	defer stop()

	return commands[args[0]](ctx, cfg)
}


// migrate brings the database up to the latest migration.
func migrate(ctx context.Context, cfg config.Config) error {
	return database.Migrate(ctx, cfg.DatabaseURL)
}


// project follows the nodes and writes what they send.
func project(ctx context.Context, cfg config.Config) error {
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	// domain claims are looked up beside the projection, writing what this
	// indexer observed of each
	go claim.NewChecker(pool, lookupOf(cfg)).Run(ctx)

	return projector.Run(ctx, pool, cfg.NodeAddrs)
}


// lookupOf is how domain claims are resolved: the DNS server when one is
// set, DNS-over-HTTPS otherwise.
func lookupOf(cfg config.Config) claim.Lookup {
	if cfg.DNSServer != "" {
		slog.Info("claims", "dns", cfg.DNSServer)
		return claim.Plain{Server: cfg.DNSServer}
	}

	slog.Info("claims", "doh", cfg.DoHURL)

	return claim.DoH{URL: cfg.DoHURL}
}


// serve answers the HTTP API.
func serve(ctx context.Context, cfg config.Config) error {
	// a bad moderators file stops the API rather than follow the wrong list
	policy, err := label.LoadPolicy(cfg.ModeratorsFile)
	if err != nil {
		return err
	}

	slog.Info("moderation", "following", len(policy.Moderators()))

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	slog.Info("ranking", "order", cfg.Ranking)

	return api.Serve(ctx, cfg.HTTPAddr, pool, policy,
	                 review.Ranking(cfg.Ranking))
}


// notBuilt stands in for a subcommand that does not exist yet.
func notBuilt(context.Context, config.Config) error {
	return errNotBuilt
}
