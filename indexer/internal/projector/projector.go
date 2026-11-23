// Package projector follows the nodes and writes what they send into the
// database: every stream feeds one applier, which writes one event at a time.
package projector


import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	pb "github.com/eduardokairalla1/EcoaNetwork/indexer/internal/protobuf"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/stream"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)


// --- CONSTANTS ---

const (
	// buffered is how many received events wait for the applier; past it,
	// the streams wait, and gRPC holds the nodes back.
	buffered = 64

	// progressEvery is how often the progress of each node is logged.
	progressEvery = 10 * time.Second

	// retryFirst and retryMax bound the wait before applying an event again
	// after the database refused it.
	retryFirst = time.Second
	retryMax   = 30 * time.Second
)


// --- CODE ---

// progress is how far the projector is in one node's log.
type progress struct {
	sequence uint64
	head     uint64
	applied  int
}


// Run follows every node in addrs and writes what they send through pool,
// until ctx ends.
func Run(ctx context.Context, pool *pgxpool.Pool, addrs []string) error {
	followers := make([]stream.Follower, 0, len(addrs))
	positions := positionsFrom(pool)

	for _, addr := range addrs {
		conn, err := grpc.NewClient(addr,
		                            grpc.WithTransportCredentials(
		                                insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("node %s: %w", addr, err)
		}
		defer conn.Close()

		followers = append(followers, stream.Follower{
			Node:       addr,
			Client:     pb.NewNodeServiceClient(conn),
			PositionOf: positions,
		})
	}

	return run(ctx, pool, followers)
}


// run feeds every follower into one applier until ctx ends.
func run(ctx context.Context,
         pool *pgxpool.Pool,
         followers []stream.Follower) error {
	received := make(chan stream.Received, buffered)

	var wg sync.WaitGroup

	for _, f := range followers {
		wg.Go(func() { f.Follow(ctx, received) })
	}

	slog.Info("projecting", "nodes", len(followers))

	a := applier{pool: pool}
	seen := map[string]*progress{}

	ticker := time.NewTicker(progressEvery)
	defer ticker.Stop()

	for {
		select {
		case r := <-received:
			if !applyUntilDone(ctx, a, r) {
				wg.Wait()
				return nil
			}

			record(seen, r)

		case <-ticker.C:
			report(seen)

		case <-ctx.Done():
			wg.Wait()
			report(seen)

			return nil
		}
	}
}


// applyUntilDone applies r, trying again while the database refuses, and
// reports false when ctx ends first. An event is never skipped for a
// database error: the ones after it would apply before it.
func applyUntilDone(ctx context.Context, a applier, r stream.Received) bool {
	wait := retryFirst

	for {
		err := a.apply(r)
		if err == nil {
			return true
		}

		slog.Warn("apply failed",
		          "node", r.Node,
		          "sequence", r.Event.GetSequence(),
		          "err", err,
		          "retryIn", wait.String())

		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return false
		}

		wait = min(wait*2, retryMax)
	}
}


// positionsFrom reads where each stream resumes from the saved cursors.
func positionsFrom(pool *pgxpool.Pool) stream.PositionOf {
	cursors := NewRepository(pool)

	return func(ctx context.Context, node string) (stream.Position, error) {
		c, err := cursors.CursorOf(ctx, node)

		return stream.Position{LogID: c.LogID, After: c.After}, err
	}
}


// record counts r in the progress of its node.
func record(seen map[string]*progress, r stream.Received) {
	p := seen[r.Node]
	if p == nil {
		p = &progress{}
		seen[r.Node] = p
	}

	p.sequence = r.Event.GetSequence()
	p.head = r.Event.GetHead()
	p.applied++
}


// report logs how far behind each node the projector is, and resets the
// count of what it applied since the last report.
func report(seen map[string]*progress) {
	for node, p := range seen {
		if p.applied == 0 {
			continue
		}

		slog.Info("progress",
		          "node", node,
		          "sequence", p.sequence,
		          "head", p.head,
		          "behind", p.head-min(p.sequence, p.head),
		          "applied", p.applied)

		p.applied = 0
	}
}
