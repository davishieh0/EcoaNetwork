package p2p


import (
	"bufio"
	"errors"
	"log/slog"
	"time"

	pb "github.com/eduardokairalla1/EcoaNetwork/internal/protobuf"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/proto"
)


// --- CONSTANTS ---

const (
	// SyncProtocol is the libp2p protocol ID of the sync streams.
	SyncProtocol protocol.ID = "/ecoa/sync/1.0.0"

	// MaxPage is the most events, or ids, one sync call serves.
	MaxPage = 256

	// maxRequestSize is the largest SyncRequest a node reads.
	maxRequestSize = 32 * 1024

	// syncStreamTimeout is how long one sync call may hold its stream.
	syncStreamTimeout = time.Minute
)


// --- GLOBALS ---

// errNoCall is a SyncRequest naming neither of the two calls.
var errNoCall = errors.New("p2p: sync request carries no call")


// --- CODE ---

// LoggedEvent is one event of the local log and its receive sequence.
type LoggedEvent struct {
	// Seq is the local receive sequence number.
	Seq uint64

	// Raw is the envelope JSON exactly as stored.
	Raw []byte
}


// SyncSource is what the sync handler reads from the local log.
type SyncSource interface {
	// EventsSince returns up to limit events with a sequence above cursor,
	// in sequence order.
	EventsSince(cursor uint64, limit int) ([]LoggedEvent, error)

	// GetEvent returns the event stored under id, or nil when there is none.
	GetEvent(id string) ([]byte, error)

	// Tombstone reports whether id was pruned from the log.
	Tombstone(id string) (bool, error)
}


// ServeSync answers the sync calls peers open on SyncProtocol from src.
func (p *Host) ServeSync(src SyncSource) {
	p.host.SetStreamHandler(SyncProtocol, func(s network.Stream) {
		p.serveStream(s, src)
	})
}


// serveStream reads one SyncRequest, writes its response and closes.
func (p *Host) serveStream(s network.Stream, src SyncSource) {
	peerID := s.Conn().RemotePeer()

	if !p.limiter.acquire(peerID) {
		slog.Warn("sync over limit", "peer", peerID)
		s.Reset()
		return
	}
	defer p.limiter.release(peerID)

	resp, err := respond(s, src)
	if err != nil {
		slog.Warn("sync refused", "peer", peerID, "reason", err)
		s.Reset()
		return
	}

	_, err = protodelim.MarshalTo(s, resp)
	if err != nil {
		slog.Warn("sync write failed", "peer", peerID, "reason", err)
		s.Reset()
		return
	}

	p.limiter.record(peerID, servedEvents(resp))
	s.Close()
}


// servedEvents counts the events a response carries.
func servedEvents(resp proto.Message) int {
	switch r := resp.(type) {
	case *pb.GetEventsSinceResponse:
		return len(r.Events)
	case *pb.GetEventsResponse:
		return len(r.Results)
	}

	return 0
}


// respond reads the request off the stream and builds its response.
func respond(s network.Stream, src SyncSource) (proto.Message, error) {
	err := s.SetDeadline(time.Now().Add(syncStreamTimeout))
	if err != nil {
		return nil, err
	}

	var req pb.SyncRequest

	reader := bufio.NewReader(s)
	opts := protodelim.UnmarshalOptions{MaxSize: maxRequestSize}

	err = opts.UnmarshalFrom(reader, &req)
	if err != nil {
		return nil, err
	}

	switch call := req.Call.(type) {
	case *pb.SyncRequest_Since:
		return eventsSince(call.Since, src)
	case *pb.SyncRequest_Events:
		return events(call.Events, src)
	}

	return nil, errNoCall
}


// eventsSince serves one page of the log above the request's cursor.
func eventsSince(req *pb.GetEventsSinceRequest,
                 src SyncSource) (*pb.GetEventsSinceResponse, error) {
	limit := MaxPage
	if size := int(req.PageSize); size > 0 && size < MaxPage {
		limit = size
	}

	// one past the page, to learn whether another page follows
	logged, err := src.EventsSince(req.Cursor, limit+1)
	if err != nil {
		return nil, err
	}

	resp := &pb.GetEventsSinceResponse{
		NextCursor: req.Cursor,
		HasMore:    len(logged) > limit,
	}

	for _, entry := range logged[:min(len(logged), limit)] {
		resp.Events = append(resp.Events, entry.Raw)
		resp.NextCursor = entry.Seq
	}

	return resp, nil
}


// events answers the first MaxPage requested ids, one result each.
func events(req *pb.GetEventsRequest,
            src SyncSource) (*pb.GetEventsResponse, error) {
	resp := &pb.GetEventsResponse{}

	for _, id := range req.Ids[:min(len(req.Ids), MaxPage)] {
		result, err := lookup(id, src)
		if err != nil {
			return nil, err
		}

		resp.Results = append(resp.Results, result)
	}

	return resp, nil
}


// lookup is FOUND with the stored bytes, PRUNED with a tombstone, else UNKNOWN.
func lookup(id string, src SyncSource) (*pb.EventResult, error) {
	raw, err := src.GetEvent(id)
	if err != nil {
		return nil, err
	}

	if raw != nil {
		return &pb.EventResult{
			Id:     id,
			Status: pb.EventStatus_FOUND,
			Event:  raw,
		}, nil
	}

	pruned, err := src.Tombstone(id)
	if err != nil {
		return nil, err
	}

	if pruned {
		return &pb.EventResult{Id: id, Status: pb.EventStatus_PRUNED}, nil
	}

	return &pb.EventResult{Id: id, Status: pb.EventStatus_UNKNOWN}, nil
}
