package main


import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"

	"github.com/eduardokairalla1/EcoaNetwork/internal/config"
	"github.com/eduardokairalla1/EcoaNetwork/internal/node"
	"github.com/eduardokairalla1/EcoaNetwork/internal/p2p"
	"github.com/eduardokairalla1/EcoaNetwork/internal/server"
	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
)


// --- CODE ---

func main() {
	// The composition root installs the default logger.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(os.Args[1:]); err != nil {
		slog.Error("start", "err", err)
		os.Exit(1)
	}
}


// run starts the node and holds until an interrupt ends it.
func run(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments %q: "+
		                  "the node is configured by environment variables",
		                  args)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	db, err := store.New(cfg.BoltDBPath)
	if err != nil {
		return fmt.Errorf("open store %s: %w", cfg.BoltDBPath, err)
	}
	defer db.Close()

	n, err := node.New(ctx, db, p2p.Config{
		Network:        cfg.Network,
		Listen:         cfg.ListenAddr,
		BootstrapAddrs: cfg.BootstrapAddrs,
		Sync: p2p.SyncLimits{
			MaxStreams:        cfg.SyncMaxStreams,
			MaxStreamsPerPeer: cfg.SyncMaxStreamsPerPeer,
			EventsPerPeer:     cfg.SyncEventsPerPeer,
		},
	}, cfg.MaxEventSize)
	if err != nil {
		return err
	}
	defer n.Close()

	slog.Info("joined", "topic", p2p.Topic(cfg.Network))

	for _, addr := range n.Addrs() {
		slog.Info("listening", "addr", addr)
	}

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen grpc %s: %w", cfg.GRPCAddr, err)
	}

	g := server.New(ctx, n)

	// Deferred after n.Close, so it stops before the node it calls into.
	defer g.GracefulStop()

	go func() {
		if err := g.Serve(lis); err != nil {
			slog.Error("serve grpc", "err", err)
		}
	}()

	slog.Info("grpc", "addr", lis.Addr().String())

	// Run holds until ctx ends.
	n.Run(ctx)

	return nil
}
