package projector


import (
	"bytes"
	"context"
	"crypto/ed25519"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/database"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/database/dbtest"
	pb "github.com/eduardokairalla1/EcoaNetwork/indexer/internal/protobuf"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/stream"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/testevent"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)


// --- CONSTANTS ---

// waitFor bounds how long a test waits for the projector to catch up.
const waitFor = 5 * time.Second


// --- CODE ---

// fakeNode serves a fixed log over SubscribeEvents, as a node does: from the
// cursor in the same log, from the start in another.
type fakeNode struct {
	pb.UnimplementedNodeServiceServer

	logID  string
	events []*pb.StoredEvent

	// dropFirst ends the first stream with Unavailable once it sent its log
	dropFirst bool

	mu      sync.Mutex
	cursors []*pb.Cursor
}


func (f *fakeNode) SubscribeEvents(c *pb.Cursor,
                                   s grpc.ServerStreamingServer[pb.StoredEvent]) error {
	f.mu.Lock()
	f.cursors = append(f.cursors, c)
	first := len(f.cursors) == 1
	f.mu.Unlock()

	after := c.GetAfter()
	if c.GetLogId() != f.logID {
		after = 0
	}

	for _, e := range f.events {
		if e.GetSequence() <= after {
			continue
		}

		if err := s.Send(e); err != nil {
			return err
		}
	}

	if first && f.dropFirst {
		return status.Error(codes.Unavailable, "node restarting")
	}

	<-s.Context().Done()

	return nil
}


// subscriptions returns the cursors the node was asked from, in order.
func (f *fakeNode) subscriptions() []*pb.Cursor {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]*pb.Cursor(nil), f.cursors...)
}


// serveFake starts node on an in-memory connection and returns a client.
func serveFake(t *testing.T, node *fakeNode) pb.NodeServiceClient {
	t.Helper()

	lis := bufconn.Listen(1 << 20)
	g := grpc.NewServer()
	pb.RegisterNodeServiceServer(g, node)

	go g.Serve(lis)
	t.Cleanup(g.Stop)

	conn, err := grpc.NewClient("passthrough:///fake",
		grpc.WithContextDialer(func(ctx context.Context,
		                            _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}

	t.Cleanup(func() { conn.Close() })

	return pb.NewNodeServiceClient(conn)
}


// migrated returns a pool over the emptied, migrated test database.
func migrated(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := dbtest.URL(t)

	if err := database.Migrate(context.Background(), url); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	return dbtest.Pool(t, url)
}


// signed returns a valid review with the given body, and its eventId.
func signed(t *testing.T, body string) ([]byte, string) {
	t.Helper()

	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{7}, 32))

	raw, err := testevent.Sign(key, testevent.Fields{
		Protocol:  "ecoa-dev",
		Version:   1,
		EventType: "review.created",
		CreatedAt: "2026-11-21T12:00:00Z",
		Payload: `{"entity":"cnpj:12345678000199","rating":4,"body":"` +
		         body + `"}`,
	})
	if err != nil {
		t.Fatalf("testevent.Sign: %v", err)
	}

	env, err := envelope.Decode(raw)
	if err != nil {
		t.Fatalf("envelope.Decode: %v", err)
	}

	return raw, env.EventID
}


// logOf returns count signed events as a node's log, numbered from 1.
func logOf(t *testing.T, logID string, count int) []*pb.StoredEvent {
	t.Helper()

	var events []*pb.StoredEvent

	for i := 1; i <= count; i++ {
		raw, id := signed(t, fmt.Sprintf("review %d", i))

		events = append(events, &pb.StoredEvent{
			Envelope:   raw,
			Sequence:   uint64(i),
			ReceivedAt: time.Date(2026, 11, 21, 12, 0, i, 0, time.UTC).UnixNano(),
			LogId:      logID,
			EventId:    id,
			EventType:  "review.created",
			Head:       uint64(count),
		})
	}

	return events
}


// project runs the projector over nodes until the test ends.
func project(t *testing.T, pool *pgxpool.Pool, nodes map[string]*fakeNode) {
	t.Helper()

	var followers []stream.Follower

	for name, node := range nodes {
		followers = append(followers, stream.Follower{
			Node:       name,
			Client:     serveFake(t, node),
			PositionOf: positionsFrom(pool),
			FirstWait:  10 * time.Millisecond,
			MaxWait:    50 * time.Millisecond,
		})
	}

	ctx, stop := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		run(ctx, pool, followers)
		close(done)
	}()

	t.Cleanup(func() {
		stop()
		<-done
	})
}


// eventually waits until ok holds, or stops the test.
func eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()

	deadline := time.Now().Add(waitFor)

	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}

		time.Sleep(10 * time.Millisecond)
	}
}


// count returns the rows of table.
func count(t *testing.T, pool *pgxpool.Pool, table string) int {
	t.Helper()

	var n int

	err := pool.QueryRow(context.Background(),
		"SELECT count(*) FROM "+table).Scan(&n)
	if err != nil {
		t.Fatalf("count %s: %v", table, err)
	}

	return n
}


// cursorOf returns the saved cursor of node.
func cursorOf(t *testing.T, pool *pgxpool.Pool, node string) Cursor {
	t.Helper()

	c, err := NewRepository(pool).CursorOf(context.Background(), node)
	if err != nil {
		t.Fatalf("CursorOf: %v", err)
	}

	return c
}


func TestProjectorStoresEveryEventWithItsObservationAndCursor(t *testing.T) {
	// Arrange
	pool := migrated(t)
	node := &fakeNode{logID: "log-a", events: logOf(t, "log-a", 3)}

	// Act
	project(t, pool, map[string]*fakeNode{"node-a": node})

	// Assert
	eventually(t, "the cursor at 3", func() bool {
		return cursorOf(t, pool, "node-a") == Cursor{LogID: "log-a", After: 3}
	})

	if count(t, pool, "events") != 3 || count(t, pool, "event_observations") != 3 {
		t.Fatalf("events = %d, observations = %d, want 3 and 3",
		         count(t, pool, "events"), count(t, pool, "event_observations"))
	}

	var stored []byte

	err := pool.QueryRow(context.Background(),
		"SELECT envelope FROM events WHERE event_id = $1",
		node.events[0].GetEventId()).Scan(&stored)
	if err != nil || !bytes.Equal(stored, node.events[0].GetEnvelope()) {
		t.Fatalf("envelope = %q, %v, want the bytes the node sent", stored, err)
	}
}


func TestProjectorResumesFromTheSavedCursor(t *testing.T) {
	// Arrange: the first two were applied before
	pool := migrated(t)
	node := &fakeNode{logID: "log-a", events: logOf(t, "log-a", 3)}

	err := NewRepository(pool).Save(context.Background(), "node-a",
	                                Cursor{LogID: "log-a", After: 2})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Act
	project(t, pool, map[string]*fakeNode{"node-a": node})

	// Assert
	eventually(t, "the cursor at 3", func() bool {
		return cursorOf(t, pool, "node-a").After == 3
	})

	first := node.subscriptions()[0]

	if first.GetAfter() != 2 || first.GetLogId() != "log-a" ||
	   count(t, pool, "events") != 1 {
		t.Fatalf("asked from %+v, stored %d, want after 2 in log-a, 1 stored",
		         first, count(t, pool, "events"))
	}
}


func TestProjectorReconnectsWhenAStreamDrops(t *testing.T) {
	// Arrange: the node ends the first stream after sending its log
	pool := migrated(t)
	node := &fakeNode{
		logID:     "log-a",
		events:    logOf(t, "log-a", 3),
		dropFirst: true,
	}

	// Act
	project(t, pool, map[string]*fakeNode{"node-a": node})

	// Assert: it came back, and nothing was stored twice
	eventually(t, "a second stream", func() bool {
		return len(node.subscriptions()) >= 2
	})

	eventually(t, "the cursor at 3", func() bool {
		return cursorOf(t, pool, "node-a").After == 3
	})

	if count(t, pool, "events") != 3 {
		t.Fatalf("events = %d, want 3", count(t, pool, "events"))
	}
}


func TestProjectorFollowsANodeThatStartedANewLog(t *testing.T) {
	// Arrange: the cursor is in a log the node no longer has
	pool := migrated(t)
	node := &fakeNode{logID: "log-new", events: logOf(t, "log-new", 2)}

	err := NewRepository(pool).Save(context.Background(), "node-a",
	                                Cursor{LogID: "log-old", After: 500})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Act
	project(t, pool, map[string]*fakeNode{"node-a": node})

	// Assert
	eventually(t, "the cursor in the new log", func() bool {
		return cursorOf(t, pool, "node-a") == Cursor{LogID: "log-new", After: 2}
	})

	if count(t, pool, "events") != 2 {
		t.Fatalf("events = %d, want both of the new log", count(t, pool, "events"))
	}
}


func TestProjectorStoresAnEventTwoNodesSendOnceWithTheEarliestTime(t *testing.T) {
	// Arrange: the same event, at another sequence and time on each node
	pool := migrated(t)

	a := logOf(t, "log-a", 1)
	b := []*pb.StoredEvent{{
		Envelope:   a[0].GetEnvelope(),
		Sequence:   7,
		ReceivedAt: a[0].GetReceivedAt() - int64(time.Hour),
		LogId:      "log-b",
		EventId:    a[0].GetEventId(),
		EventType:  a[0].GetEventType(),
		Head:       7,
	}}

	// Act
	project(t, pool, map[string]*fakeNode{
		"node-a": {logID: "log-a", events: a},
		"node-b": {logID: "log-b", events: b},
	})

	// Assert
	eventually(t, "both cursors", func() bool {
		return cursorOf(t, pool, "node-a").After == 1 &&
		       cursorOf(t, pool, "node-b").After == 7
	})

	var receivedAt time.Time

	err := pool.QueryRow(context.Background(),
		"SELECT received_at FROM event_observations").Scan(&receivedAt)
	if err != nil {
		t.Fatalf("read observation: %v", err)
	}

	earliest := time.Unix(0, b[0].GetReceivedAt())

	if count(t, pool, "events") != 1 || !receivedAt.Equal(earliest) {
		t.Fatalf("events = %d, receivedAt = %v, want 1 at %v",
		         count(t, pool, "events"), receivedAt, earliest)
	}
}


func TestApplyingTheSameEventTwiceChangesNothing(t *testing.T) {
	// Arrange
	pool := migrated(t)
	r := stream.Received{Node: "node-a", Event: logOf(t, "log-a", 1)[0]}
	a := applier{pool: pool}

	// Act
	first, second := a.apply(r), a.apply(r)

	// Assert
	if first != nil || second != nil || count(t, pool, "events") != 1 ||
	   count(t, pool, "event_observations") != 1 {
		t.Fatalf("apply = %v, %v; events = %d", first, second,
		         count(t, pool, "events"))
	}
}


func TestProjectorSkipsAnUnreadableEventAndMovesOn(t *testing.T) {
	// Arrange: the second of three is not an envelope
	pool := migrated(t)
	events := logOf(t, "log-a", 3)
	events[1].Envelope = []byte("not an envelope")

	// Act
	project(t, pool, map[string]*fakeNode{
		"node-a": {logID: "log-a", events: events},
	})

	// Assert
	eventually(t, "the cursor at 3", func() bool {
		return cursorOf(t, pool, "node-a").After == 3
	})

	if count(t, pool, "events") != 2 {
		t.Fatalf("events = %d, want the two readable ones",
		         count(t, pool, "events"))
	}
}


func TestRunStopsWithItsContext(t *testing.T) {
	// Arrange
	pool := migrated(t)
	node := &fakeNode{logID: "log-a", events: logOf(t, "log-a", 1)}

	followers := []stream.Follower{{
		Node:       "node-a",
		Client:     serveFake(t, node),
		PositionOf: positionsFrom(pool),
	}}

	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() { done <- run(ctx, pool, followers) }()

	eventually(t, "the event", func() bool {
		return cursorOf(t, pool, "node-a").After == 1
	})

	// Act
	stop()

	// Assert
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run = %v, want a clean stop", err)
		}
	case <-time.After(waitFor):
		t.Fatal("run did not stop with its context")
	}
}
