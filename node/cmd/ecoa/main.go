package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"

	"github.com/eduardokairalla1/EcoaNetwork/internal/node"
	"github.com/eduardokairalla1/EcoaNetwork/internal/p2p"
	"github.com/eduardokairalla1/EcoaNetwork/internal/server"
	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
)


// --- GLOBALS ---



// --- CODE ---

func main() {
	network := flag.String("network",
						   "ecoa-dev",
						   "network this node belongs to")

	listen := flag.String("listen",
						  "/ip4/127.0.0.1/tcp/4001",
						  "multiaddr to listen on")

	grpcAddr := flag.String("grpc",
							"127.0.0.1:50051",
							"host:port the gateway submits events to")

	bootstrapAddrs := flag.String(
		"bootstrap",
		"",
		"comma-separated peer multiaddrs; replaces the default list",
	)

	publish := flag.String("publish",
						   "",
						   "event file to publish after connecting")
	flag.Parse()

	addrs := p2p.DefaultBootstrapAddrs

	// The flag replaces the default list, it does not extend it.
	if *bootstrapAddrs != "" {
		addrs = strings.Split(*bootstrapAddrs, ",")
	}

	// The composition root installs the default logger.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	boltDbPath := os.Getenv("BOLT_DB_PATH")
	if boltDbPath == "" {
		boltDbPath = "data/bbolt/bolt.db"
	}

	db, err := store.New(boltDbPath)
	if err != nil {
		slog.Error("open store", "path", boltDbPath, "err", err)
		os.Exit(1)
	}

	defer db.Close()

	n, err := node.New(
		ctx,
		db,
		p2p.Config{
			Network:        *network,
			Listen:         *listen,
			BootstrapAddrs: addrs,
		},
	)

	if err != nil {
		slog.Error("start", "err", err)
		os.Exit(1)
	}
	defer n.Close()

	slog.Info("joined", "topic", p2p.Topic(*network))

	for _, addr := range n.Addrs() {
		slog.Info("listening", "addr", addr)
	}

	lis, err := net.Listen("tcp", *grpcAddr)
	if err != nil {
		slog.Error("listen grpc", "addr", *grpcAddr, "err", err)
		os.Exit(1)
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

	if *publish != "" {
		raw, err := os.ReadFile(*publish)
		if err != nil {
			slog.Error("read event", "err", err)
			os.Exit(1)
		}

		go func() {
			if err := n.Publish(ctx, raw); err != nil {
				slog.Error("publish", "err", err)
			}
		}()
	}

	// Run holds main until ctx ends, so the deferred closes wait for it.
	n.Run(ctx)
}
