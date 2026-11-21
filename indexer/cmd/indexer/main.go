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
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/config"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/database"
)


// --- GLOBALS ---

// commands maps each subcommand to what it runs.
var commands = map[string]func(context.Context, config.Config) error{
	"migrate": migrate,
	"project": notBuilt,
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


// serve answers the HTTP API.
func serve(ctx context.Context, cfg config.Config) error {
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	return api.Serve(ctx, cfg.HTTPAddr, pool)
}


// notBuilt stands in for a subcommand that does not exist yet.
func notBuilt(context.Context, config.Config) error {
	return errNotBuilt
}
