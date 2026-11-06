package p2p


import (
	"bufio"
	"context"
	"time"

	pb "github.com/eduardokairalla1/EcoaNetwork/internal/protobuf"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/proto"
)


// --- CONSTANTS ---

// maxResponseSize is the largest sync response a node reads.
const maxResponseSize = 8 * 1024 * 1024


// --- CODE ---

// GetEventsSince asks p for one page of the events it accepted after cursor.
func (p *Host) GetEventsSince(ctx context.Context,
                              id peer.ID,
                              cursor uint64) (*pb.GetEventsSinceResponse,
                                              error) {
	req := &pb.SyncRequest{Call: &pb.SyncRequest_Since{
		Since: &pb.GetEventsSinceRequest{Cursor: cursor, PageSize: MaxPage},
	}}

	var resp pb.GetEventsSinceResponse
	if err := p.call(ctx, id, req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}


// GetEvents asks p for the events stored under ids.
func (p *Host) GetEvents(ctx context.Context,
                         id peer.ID,
                         ids []string) (*pb.GetEventsResponse, error) {
	req := &pb.SyncRequest{Call: &pb.SyncRequest_Events{
		Events: &pb.GetEventsRequest{Ids: ids},
	}}

	var resp pb.GetEventsResponse
	if err := p.call(ctx, id, req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}


// OnConnect calls fn in its own goroutine for each connected peer.
func (p *Host) OnConnect(fn func(peer.ID)) {
	p.host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(_ network.Network, c network.Conn) {
			go fn(c.RemotePeer())
		},
	})

	for _, id := range p.host.Network().Peers() {
		go fn(id)
	}
}


// call opens a sync stream to id, sends req and reads the answer into resp.
func (p *Host) call(ctx context.Context,
                    id peer.ID,
                    req *pb.SyncRequest,
                    resp proto.Message) error {
	s, err := p.host.NewStream(ctx, id, SyncProtocol)
	if err != nil {
		return err
	}

	err = s.SetDeadline(time.Now().Add(syncStreamTimeout))
	if err == nil {
		_, err = protodelim.MarshalTo(s, req)
	}

	if err == nil {
		opts := protodelim.UnmarshalOptions{MaxSize: maxResponseSize}
		err = opts.UnmarshalFrom(bufio.NewReader(s), resp)
	}

	if err != nil {
		s.Reset()
		return err
	}

	return s.Close()
}
