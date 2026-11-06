package p2p


import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	pb "github.com/eduardokairalla1/EcoaNetwork/internal/protobuf"
	"github.com/libp2p/go-libp2p/core/peer"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/proto"
)


// --- CODE ---

// fakeSource is a SyncSource over events numbered 1..len(log).
type fakeSource struct {
	log        []LoggedEvent
	stored     map[string][]byte
	tombstones map[string]bool
}


func (f *fakeSource) EventsSince(cursor uint64,
                                 limit int) ([]LoggedEvent, error) {
	var page []LoggedEvent

	for _, entry := range f.log {
		if entry.Seq > cursor && len(page) < limit {
			page = append(page, entry)
		}
	}

	return page, nil
}


func (f *fakeSource) GetEvent(id string) ([]byte, error) {
	return f.stored[id], nil
}


func (f *fakeSource) Tombstone(id string) (bool, error) {
	return f.tombstones[id], nil
}


// sourceWith returns a source holding count events, evt_1 to evt_<count>.
func sourceWith(count int) *fakeSource {
	src := &fakeSource{
		stored:     map[string][]byte{},
		tombstones: map[string]bool{},
	}

	for seq := 1; seq <= count; seq++ {
		raw := []byte(fmt.Sprintf(`{"eventId":"evt_%d"}`, seq))

		src.log = append(src.log, LoggedEvent{Seq: uint64(seq), Raw: raw})
		src.stored[fmt.Sprintf("evt_%d", seq)] = raw
	}

	return src
}


// syncPair returns a client peer connected to a peer serving src.
func syncPair(t *testing.T, src SyncSource) (*Host, peer.ID) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	server, err := New(ctx, Config{
		Network:   "ecoa-test",
		Listen:    loopback,
		Validator: acceptAll,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Close() })

	client, err := New(ctx, Config{
		Network:   "ecoa-test",
		Listen:    loopback,
		Validator: acceptAll,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })

	server.ServeSync(src)

	info := peer.AddrInfo{ID: server.host.ID(), Addrs: server.host.Addrs()}
	if err := client.host.Connect(ctx, info); err != nil {
		t.Fatal(err)
	}

	return client, server.host.ID()
}


// ask opens a sync stream, sends req and reads the answer into resp.
func ask(t *testing.T,
         client *Host,
         server peer.ID,
         req *pb.SyncRequest,
         resp proto.Message) error {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s, err := client.host.NewStream(ctx, server, SyncProtocol)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if _, err := protodelim.MarshalTo(s, req); err != nil {
		t.Fatal(err)
	}

	return protodelim.UnmarshalFrom(bufio.NewReader(s), resp)
}


func sinceRequest(cursor uint64, pageSize uint32) *pb.SyncRequest {
	return &pb.SyncRequest{Call: &pb.SyncRequest_Since{
		Since: &pb.GetEventsSinceRequest{Cursor: cursor, PageSize: pageSize},
	}}
}


func eventsRequest(ids []string) *pb.SyncRequest {
	return &pb.SyncRequest{Call: &pb.SyncRequest_Events{
		Events: &pb.GetEventsRequest{Ids: ids},
	}}
}


func TestServeSyncFirstPage(t *testing.T) {
	// Arrange
	client, server := syncPair(t, sourceWith(300))

	// Act
	var resp pb.GetEventsSinceResponse
	err := ask(t, client, server, sinceRequest(0, 0), &resp)

	// Assert
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Events) != MaxPage || !resp.HasMore || resp.NextCursor != 256 {
		t.Errorf("got %d events, has_more %v, next %d",
		         len(resp.Events), resp.HasMore, resp.NextCursor)
	}
}


func TestServeSyncLastPage(t *testing.T) {
	// Arrange
	client, server := syncPair(t, sourceWith(300))

	// Act
	var resp pb.GetEventsSinceResponse
	err := ask(t, client, server, sinceRequest(256, 0), &resp)

	// Assert
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Events) != 44 || resp.HasMore || resp.NextCursor != 300 {
		t.Errorf("got %d events, has_more %v, next %d",
		         len(resp.Events), resp.HasMore, resp.NextCursor)
	}
}


func TestServeSyncPageSize(t *testing.T) {
	tests := []struct {
		name     string
		pageSize uint32
		want     int
	}{
		{"smaller than the ceiling", 10, 10},
		{"larger than the ceiling", 1000, MaxPage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			client, server := syncPair(t, sourceWith(300))

			// Act
			var resp pb.GetEventsSinceResponse
			req := sinceRequest(0, tt.pageSize)
			err := ask(t, client, server, req, &resp)

			// Assert
			if err != nil {
				t.Fatal(err)
			}

			if len(resp.Events) != tt.want {
				t.Errorf("got %d events, want %d", len(resp.Events), tt.want)
			}
		})
	}
}


func TestServeSyncEventBytes(t *testing.T) {
	// Arrange
	src := sourceWith(3)
	client, server := syncPair(t, src)

	// Act
	var resp pb.GetEventsSinceResponse
	err := ask(t, client, server, sinceRequest(0, 0), &resp)

	// Assert
	if err != nil {
		t.Fatal(err)
	}

	for i, raw := range resp.Events {
		if !bytes.Equal(raw, src.log[i].Raw) {
			t.Errorf("event %d: got %q, stored %q", i, raw, src.log[i].Raw)
		}
	}
}


func TestServeSyncGetEventsCeiling(t *testing.T) {
	// Arrange
	client, server := syncPair(t, sourceWith(0))

	ids := make([]string, 300)
	for i := range ids {
		ids[i] = fmt.Sprintf("evt_%d", i)
	}

	// Act
	var resp pb.GetEventsResponse
	err := ask(t, client, server, eventsRequest(ids), &resp)

	// Assert
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Results) != MaxPage {
		t.Errorf("got %d results, want %d", len(resp.Results), MaxPage)
	}
}


func TestServeSyncGetEventsStatus(t *testing.T) {
	// Arrange
	src := sourceWith(1)
	src.tombstones["evt_gone"] = true
	client, server := syncPair(t, src)

	ids := []string{"evt_1", "evt_gone", "evt_never"}
	want := []pb.EventStatus{
		pb.EventStatus_FOUND,
		pb.EventStatus_PRUNED,
		pb.EventStatus_UNKNOWN,
	}

	// Act
	var resp pb.GetEventsResponse
	err := ask(t, client, server, eventsRequest(ids), &resp)

	// Assert
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Results) != len(ids) {
		t.Fatalf("got %d results, want %d", len(resp.Results), len(ids))
	}

	for i, result := range resp.Results {
		if result.Id != ids[i] || result.Status != want[i] {
			t.Errorf("result %d: got %v %v", i, result.Id, result.Status)
		}
	}

	if !bytes.Equal(resp.Results[0].Event, src.stored["evt_1"]) {
		t.Errorf("FOUND bytes: got %q", resp.Results[0].Event)
	}

	if resp.Results[1].Event != nil || resp.Results[2].Event != nil {
		t.Error("only FOUND carries event bytes")
	}
}


func TestServeSyncResetsRequestWithoutCall(t *testing.T) {
	// Arrange
	client, server := syncPair(t, sourceWith(1))

	// Act
	var resp pb.GetEventsSinceResponse
	err := ask(t, client, server, &pb.SyncRequest{}, &resp)

	// Assert
	if err == nil {
		t.Error("a request naming no call should be refused")
	}
}
