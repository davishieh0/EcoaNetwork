// Package stream follows one node's log over SubscribeEvents, reconnecting
// on its own, and hands every event it receives on.
package stream


import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	pb "github.com/eduardokairalla1/EcoaNetwork/indexer/internal/protobuf"
	"google.golang.org/grpc/status"
)


// --- CONSTANTS ---

const (
	// firstWait is the wait before the first retry after a stream ends.
	firstWait = time.Second

	// maxWait is the longest wait between retries.
	maxWait = 30 * time.Second
)


// --- CODE ---

// Received is one event of a node's log, with the node it came from.
type Received struct {
	// Node is the node's address, as configured.
	Node string

	// Event is the event and what the node observed of it.
	Event *pb.StoredEvent
}


// Position is where to start reading a node's log.
type Position struct {
	// LogID is the log After counts in, "" the first time.
	LogID string

	// After is the last sequence applied, 0 for the whole log.
	After uint64
}


// PositionOf returns where to resume a node's log: the last cursor applied,
// never the last event received, so a reconnect loses nothing.
type PositionOf func(ctx context.Context, node string) (Position, error)


// Follower follows one node.
type Follower struct {
	// Node is the node's address, as configured.
	Node string

	// Client reaches the node.
	Client pb.NodeServiceClient

	// PositionOf says where each stream starts.
	PositionOf PositionOf

	// FirstWait and MaxWait bound the wait between retries; 0 means the
	// defaults, and tests make them short.
	FirstWait, MaxWait time.Duration
}


// Follow reads the node's log into out until ctx ends, opening a new stream
// whenever one ends, after a wait that grows while they keep failing.
func (f Follower) Follow(ctx context.Context, out chan<- Received) {
	first, longest := f.FirstWait, f.MaxWait
	if first == 0 {
		first = firstWait
	}

	if longest == 0 {
		longest = maxWait
	}

	wait := first

	for {
		received, err := f.once(ctx, out)

		if ctx.Err() != nil {
			return
		}

		// a stream that delivered something was working: start over short
		if received > 0 {
			wait = first
		}

		slog.Warn("stream ended",
		          "node", f.Node,
		          "received", received,
		          "code", status.Code(err).String(),
		          "err", err,
		          "retryIn", wait.String())

		if !sleep(ctx, jitter(wait)) {
			return
		}

		wait = min(wait*2, longest)
	}
}


// once opens one stream from the applied cursor and reads it until it ends,
// returning how many events it received.
func (f Follower) once(ctx context.Context, out chan<- Received) (int, error) {
	position, err := f.PositionOf(ctx, f.Node)
	if err != nil {
		return 0, err
	}

	stream, err := f.Client.SubscribeEvents(ctx, &pb.Cursor{
		After: position.After,
		LogId: position.LogID,
	})
	if err != nil {
		return 0, err
	}

	slog.Info("stream open",
	          "node", f.Node,
	          "after", position.After,
	          "logId", position.LogID)

	received := 0

	for {
		event, err := stream.Recv()
		if err != nil {
			return received, err
		}

		select {
		case out <- Received{Node: f.Node, Event: event}:
			received++
		case <-ctx.Done():
			return received, ctx.Err()
		}
	}
}


// jitter returns a wait between half of d and d, so that readers of a node
// that went down do not all come back in the same instant.
func jitter(d time.Duration) time.Duration {
	half := d / 2

	return half + rand.N(half+1)
}


// sleep waits d, or reports false when ctx ends first.
func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
