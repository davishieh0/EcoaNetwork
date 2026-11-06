package node


import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/internal/p2p"
	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
	"github.com/libp2p/go-libp2p/core/peer"
)


// --- CONSTANTS ---

// loopback lets the kernel pick a free port, so tests never collide.
const loopback = "/ip4/127.0.0.1/tcp/0"


// --- GLOBALS ---

// roomy are limits no test here comes near.
var roomy = p2p.SyncLimits{
	MaxStreams:        100,
	MaxStreamsPerPeer: 100,
	EventsPerPeer:     100000,
}


// --- CODE ---

// logServer is a SyncSource over a fixed log that notes each cursor asked.
type logServer struct {
	log   []p2p.LoggedEvent
	asked []uint64

	// onAsk runs when a page is requested, before it is served
	onAsk func(cursor uint64)
}


func (l *logServer) EventsSince(cursor uint64,
                                limit int) ([]p2p.LoggedEvent, error) {
	l.asked = append(l.asked, cursor)

	if l.onAsk != nil {
		l.onAsk(cursor)
	}

	var page []p2p.LoggedEvent

	for _, entry := range l.log {
		if entry.Seq > cursor && len(page) < limit {
			page = append(page, entry)
		}
	}

	return page, nil
}


func (l *logServer) GetEvent(id string) ([]byte, error) {
	return nil, nil
}


func (l *logServer) Tombstone(id string) (bool, error) {
	return false, nil
}


// logOf returns count distinct valid events, numbered from 1.
func logOf(t *testing.T, count int) *logServer {
	t.Helper()

	server := &logServer{}

	for seq := 1; seq <= count; seq++ {
		fields := review()
		fields.Payload = fmt.Sprintf(`{"entity":"cnpj:12345678000199",`+
		                             `"rating":4,"body":"review %d"}`, seq)

		server.log = append(server.log, p2p.LoggedEvent{
			Seq: uint64(seq),
			Raw: sign(t, 1, fields),
		})
	}

	return server
}


// syncedNode returns a node connected to a peer serving src.
func syncedNode(t *testing.T, src p2p.SyncSource) (*Node, peer.ID, string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	n, path := newNode(t)

	server, err := p2p.New(ctx, p2p.Config{
		Network:   network,
		Listen:    loopback,
		Validator: n.validate,
		Sync:      roomy,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Close() })

	server.ServeSync(src)

	info, err := peer.AddrInfoFromString(server.Addrs()[0])
	if err != nil {
		t.Fatal(err)
	}

	n.host, err = p2p.New(ctx, p2p.Config{
		Network:        network,
		Listen:         loopback,
		BootstrapAddrs: server.Addrs(),
		Validator:      n.validate,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { n.host.Close() })

	return n, info.ID, path
}


func TestCatchUpPullsEveryPage(t *testing.T) {
	// Arrange
	src := logOf(t, 600)
	n, server, path := syncedNode(t, src)

	// Act
	err := n.catchUp(context.Background(), server)

	// Assert
	if err != nil {
		t.Fatal(err)
	}

	if got := len(stored(t, n, path)); got != 600 {
		t.Errorf("stored %d events, want 600", got)
	}

	want := []uint64{0, 256, 512}
	if fmt.Sprint(src.asked) != fmt.Sprint(want) {
		t.Errorf("asked cursors %v, want %v", src.asked, want)
	}
}


func TestCatchUpResumesFromTheStoredCursor(t *testing.T) {
	// Arrange
	src := logOf(t, 10)
	n, server, _ := syncedNode(t, src)

	if err := n.store.SetCursor(server.String(), 7); err != nil {
		t.Fatal(err)
	}

	// Act
	err := n.catchUp(context.Background(), server)

	// Assert
	if err != nil {
		t.Fatal(err)
	}

	if len(src.asked) == 0 || src.asked[0] != 7 {
		t.Errorf("first cursor asked %v, want 7", src.asked)
	}
}


func TestCatchUpKeepsThePreviousCursorWhenAPageFails(t *testing.T) {
	// Arrange
	src := logOf(t, 600)
	n, server, path := syncedNode(t, src)

	src.onAsk = func(cursor uint64) {
		if cursor == 256 {
			n.store.Close()
		}
	}

	// Act
	err := n.catchUp(context.Background(), server)

	// Assert
	if err == nil {
		t.Fatal("a page the store could not take should stop the catch-up")
	}

	reopened, err := store.New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()

	if got, _ := reopened.Cursor(server.String()); got != 256 {
		t.Errorf("cursor = %d, want 256", got)
	}
}


func TestCatchUpMovesPastARejectedEvent(t *testing.T) {
	// Arrange
	src := logOf(t, 3)
	src.log[1].Raw = []byte(`{"not":"an envelope"}`)
	n, server, path := syncedNode(t, src)

	// Act
	err := n.catchUp(context.Background(), server)

	// Assert
	if err != nil {
		t.Fatal(err)
	}

	cursor, _ := n.store.Cursor(server.String())
	if cursor != 3 {
		t.Errorf("cursor = %d, want 3", cursor)
	}

	if got := len(stored(t, n, path)); got != 2 {
		t.Errorf("stored %d events, want 2", got)
	}
}


func TestCatchUpRefusesAPeerWhoseCursorDoesNotAdvance(t *testing.T) {
	// Arrange
	src := logOf(t, 600)
	n, server, _ := syncedNode(t, &stuck{src})

	// Act
	err := n.catchUp(context.Background(), server)

	// Assert
	if err == nil {
		t.Error("has_more with a cursor that stays put should stop the loop")
	}
}


// stuck serves pages whose events never move the cursor.
type stuck struct {
	*logServer
}


func (s *stuck) EventsSince(cursor uint64,
                            limit int) ([]p2p.LoggedEvent, error) {
	page, err := s.logServer.EventsSince(cursor, limit)

	for i := range page {
		page[i].Seq = cursor
	}

	return page, err
}


// waitStored waits until n holds every id, or stops the test.
func waitStored(t *testing.T, ctx context.Context, n *Node, ids []string) {
	t.Helper()

	for _, id := range ids {
		for {
			raw, err := n.store.GetEvent(id)
			if err != nil {
				t.Fatalf("GetEvent: %v", err)
			}

			if raw != nil {
				break
			}

			select {
			case <-ctx.Done():
				t.Fatalf("event %s never reached the node", id)
			case <-time.After(10 * time.Millisecond):
			}
		}
	}
}


func TestNewServesItsLogToAPeerThatConnectsLater(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	served := startNode(t, ctx)
	src := logOf(t, 3)

	var ids []string

	for _, entry := range src.log {
		outcome, err := served.Receive(entry.Raw, gateway)
		if outcome != Stored {
			t.Fatalf("Receive = %s, %v, want stored", outcome, err)
		}

		ids = append(ids, field(t, entry.Raw, "eventId"))
	}

	// Act
	late := startNode(t, ctx, served.Addrs()...)

	// Assert
	waitStored(t, ctx, late, ids)
}
