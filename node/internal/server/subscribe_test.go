package server


import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/internal/node"
	pb "github.com/eduardokairalla1/EcoaNetwork/internal/protobuf"
	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)


// --- CONSTANTS ---

// streamTimeout bounds every stream a test opens, so a hang fails it.
const streamTimeout = 5 * time.Second


// --- CODE ---

// clientStream is the client side of a SubscribeEvents stream.
type clientStream = grpc.ServerStreamingClient[pb.StoredEvent]


// follower is a node with its store, served over an in-memory connection.
type follower struct {
	node   *node.Node
	store  *store.Store
	client pb.NodeServiceClient
	grpc   *grpc.Server

	// stop ends the server's context, as the node stopping does
	stop context.CancelFunc
}


// serve starts a node and its gRPC server, capped at maxStreams.
func serve(t *testing.T, maxStreams int) follower {
	t.Helper()

	ctx, stop := context.WithCancel(context.Background())
	t.Cleanup(stop)

	n, s := startNode(t, ctx, 0)

	lis := bufconn.Listen(1 << 20)
	g := New(ctx, n, s, maxStreams)

	go g.Serve(lis)
	t.Cleanup(g.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context,
		                            _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}

	t.Cleanup(func() { conn.Close() })

	return follower{
		node:   n,
		store:  s,
		client: pb.NewNodeServiceClient(conn),
		grpc:   g,
		stop:   stop,
	}
}


// add stores count reviews on the node, each with its own body.
func (f follower) add(t *testing.T, count int) {
	t.Helper()

	for i := range count {
		body := fmt.Sprintf("review %d %d", i, time.Now().UnixNano())
		arrival := node.Arrival{Kind: node.Gateway}

		outcome, err := f.node.Receive(signed(t, 1, body), arrival)
		if outcome != node.Stored {
			t.Fatalf("Receive = %v, %v, want stored", outcome, err)
		}
	}
}


// subscribe opens a stream from cursor, ended with the test.
func (f follower) subscribe(t *testing.T, cursor *pb.Cursor) clientStream {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), streamTimeout)
	t.Cleanup(cancel)

	stream, err := f.client.SubscribeEvents(ctx, cursor)
	if err != nil {
		t.Fatalf("SubscribeEvents: %v", err)
	}

	return stream
}


// receive reads count events from stream, or stops the test.
func receive(t *testing.T,
             stream clientStream,
             count int) []*pb.StoredEvent {
	t.Helper()

	var got []*pb.StoredEvent

	for range count {
		e, err := stream.Recv()
		if err != nil {
			t.Fatalf("Recv after %d events: %v", len(got), err)
		}

		got = append(got, e)
	}

	return got
}


// sequences returns the sequence of each event, in order.
func sequences(events []*pb.StoredEvent) []uint64 {
	var got []uint64

	for _, e := range events {
		got = append(got, e.GetSequence())
	}

	return got
}


func TestSubscribeSendsTheWholeLogInOrderWithEveryField(t *testing.T) {
	// Arrange
	f := serve(t, 4)
	f.add(t, 3)

	logged, _ := f.store.EventsSince(0, 10)

	// Act
	got := receive(t, f.subscribe(t, &pb.Cursor{}), 3)

	// Assert
	for i, e := range got {
		want := logged[i]

		if e.GetSequence() != want.Sequence ||
		   e.GetEventId() != want.ID ||
		   string(e.GetEnvelope()) != string(want.Raw) ||
		   e.GetReceivedAt() != want.ReceivedAt.UnixNano() ||
		   e.GetLogId() != f.store.LogID() ||
		   e.GetEventType() != "review.created" ||
		   e.GetHead() != 3 {
			t.Fatalf("event %d = %+v, want %+v", i, e, want)
		}
	}
}


func TestSubscribeResumesAfterTheCursorInTheSameLog(t *testing.T) {
	// Arrange
	f := serve(t, 4)
	f.add(t, 3)

	cursor := &pb.Cursor{After: 2, LogId: f.store.LogID()}

	// Act
	got := receive(t, f.subscribe(t, cursor), 1)

	// Assert
	if got[0].GetSequence() != 3 {
		t.Fatalf("first sequence = %d, want 3", got[0].GetSequence())
	}
}


func TestSubscribeStartsOverForACursorInAnotherLog(t *testing.T) {
	cases := map[string]string{
		"another log": "0123456789abcdef0123456789abcdef",
		"first time":  "",
	}

	for name, logID := range cases {
		t.Run(name, func(t *testing.T) {
			// Arrange
			f := serve(t, 4)
			f.add(t, 3)

			cursor := &pb.Cursor{After: 2, LogId: logID}

			// Act
			got := receive(t, f.subscribe(t, cursor), 3)

			// Assert
			if want := []uint64{1, 2, 3}; fmt.Sprint(sequences(got)) !=
			                               fmt.Sprint(want) {
				t.Fatalf("sequences = %v, want %v", sequences(got), want)
			}
		})
	}
}


func TestSubscribeSendsAnEventStoredWhileTheStreamIsOpen(t *testing.T) {
	// Arrange
	f := serve(t, 4)
	f.add(t, 1)

	stream := f.subscribe(t, &pb.Cursor{})
	receive(t, stream, 1)

	// Act
	f.add(t, 1)
	got := receive(t, stream, 1)

	// Assert
	if got[0].GetSequence() != 2 || got[0].GetHead() < 2 {
		t.Fatalf("live event = %+v, want sequence 2", got[0])
	}
}


func TestSubscribeRefusesAStreamOverTheCap(t *testing.T) {
	// Arrange
	f := serve(t, 2)
	f.add(t, 1)

	// each one has received an event, so each holds its slot
	for range 2 {
		receive(t, f.subscribe(t, &pb.Cursor{}), 1)
	}

	// Act
	_, err := f.subscribe(t, &pb.Cursor{}).Recv()

	// Assert
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("Recv = %v, want ResourceExhausted", err)
	}
}


func TestSubscribeGivesTheSlotBackWhenAStreamEnds(t *testing.T) {
	// Arrange
	f := serve(t, 1)
	f.add(t, 1)

	ctx, leave := context.WithCancel(context.Background())

	first, err := f.client.SubscribeEvents(ctx, &pb.Cursor{})
	if err != nil {
		t.Fatalf("SubscribeEvents: %v", err)
	}

	receive(t, first, 1)

	// Act
	leave()

	// Assert: the slot comes back once the server sees the stream end
	deadline := time.Now().Add(streamTimeout)

	for {
		_, err := f.subscribe(t, &pb.Cursor{}).Recv()
		if err == nil {
			return
		}

		if status.Code(err) != codes.ResourceExhausted ||
		   time.Now().After(deadline) {
			t.Fatalf("Recv = %v, want the freed slot", err)
		}

		time.Sleep(10 * time.Millisecond)
	}
}


func TestSubscribeEndsOpenStreamsWhenTheNodeStops(t *testing.T) {
	// Arrange
	f := serve(t, 4)
	f.add(t, 1)

	stream := f.subscribe(t, &pb.Cursor{})
	receive(t, stream, 1)

	// Act
	f.stop()

	stopped := make(chan struct{})
	go func() {
		f.grpc.GracefulStop()
		close(stopped)
	}()

	_, err := stream.Recv()

	// Assert
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("Recv = %v, want Unavailable", err)
	}

	select {
	case <-stopped:
	case <-time.After(streamTimeout):
		t.Fatal("GracefulStop waited on an open stream")
	}
}
