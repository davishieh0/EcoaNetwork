package server


import (
	pb "github.com/eduardokairalla1/EcoaNetwork/internal/protobuf"
	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)


// --- CONSTANTS ---

// page is the most events read from the log at a time, as sync serves.
const page = 256


// --- GLOBALS ---

// eventStream is the server side of a SubscribeEvents stream.
type eventStream = grpc.ServerStreamingServer[pb.StoredEvent]


// errStopping ends a stream because the node is stopping.
var errStopping = status.Error(codes.Unavailable, "node is stopping")


// --- CODE ---

// SubscribeEvents sends the log after the cursor, then each new event.
//
// History and live events are one loop: read what follows the cursor, send
// it, and when nothing follows, sleep until the store stores something.
func (s *server) SubscribeEvents(cursor *pb.Cursor, stream eventStream) error {
	logID := s.store.LogID()
	after := cursor.GetAfter()

	// a cursor in another log means nothing here: send this one from the start
	if cursor.GetLogId() != logID {
		after = 0
	}

	return s.follow(after, logID, stream)
}


// follow runs the loop of SubscribeEvents.
func (s *server) follow(after uint64,
                        logID string,
                        stream eventStream) error {
	for {
		// taken before reading, so an event stored meanwhile still wakes it
		changed := s.store.Changed()

		events, err := s.store.EventsSince(after, page)
		if err != nil {
			return status.Error(codes.Internal, err.Error())
		}

		// read after the page, so it is never below a sequence in it
		head, err := s.store.Head()
		if err != nil {
			return status.Error(codes.Internal, err.Error())
		}

		for _, e := range events {
			if err := stream.Send(storedEvent(e, logID, head)); err != nil {
				return err
			}

			after = e.Sequence
		}

		if len(events) > 0 {
			continue
		}

		select {
		case <-changed:
		case <-stream.Context().Done():
			return nil
		case <-s.ctx.Done():
			return errStopping
		}
	}
}


// storedEvent is e as SubscribeEvents sends it.
func storedEvent(e store.StoredEvent,
                 logID string,
                 head uint64) *pb.StoredEvent {
	return &pb.StoredEvent{
		Envelope:   e.Raw,
		Sequence:   e.Sequence,
		ReceivedAt: e.ReceivedAt.UnixNano(),
		LogId:      logID,
		EventId:    e.ID,
		EventType:  eventTypeOf(e.Raw),
		Head:       head,
	}
}


// eventTypeOf reads the eventType of a stored envelope, "" if it cannot.
func eventTypeOf(raw []byte) string {
	// every stored event was validated on arrival, so this does not fail
	env, err := envelope.Decode(raw)
	if err != nil {
		return ""
	}

	return env.EventType
}
