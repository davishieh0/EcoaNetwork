package server


import (
	"errors"
	"log/slog"

	pb "github.com/eduardokairalla1/EcoaNetwork/internal/protobuf"
	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
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


// errFull refuses a stream over the cap on open streams.
var errFull = status.Error(codes.ResourceExhausted,
                           "too many open streams; try again later")


// --- CODE ---

// SubscribeEvents sends the log after the cursor, then each new event.
//
// History and live events are one loop: read what follows the cursor, send
// it, and when nothing follows, sleep until the store stores something.
func (s *server) SubscribeEvents(cursor *pb.Cursor, stream eventStream) error {
	if !s.acquire() {
		slog.Warn("subscribe refused",
		          "client", clientOf(stream),
		          "reason", "every stream slot is taken",
		          "maxStreams", cap(s.slots))

		return errFull
	}
	defer s.release()

	logID := s.store.LogID()
	after := cursor.GetAfter()

	// a cursor in another log means nothing here: send this one from the start
	sameLog := cursor.GetLogId() == logID
	if !sameLog {
		after = 0
	}

	slog.Info("subscribed",
	          "client", clientOf(stream),
	          "after", after,
	          "sameLog", sameLog)

	sent, err := s.follow(after, logID, stream)

	slog.Info("unsubscribed",
	          "client", clientOf(stream),
	          "sent", sent,
	          "reason", reasonOf(err))

	return err
}


// follow runs the loop of SubscribeEvents, returning how many events it sent.
func (s *server) follow(after uint64,
                        logID string,
                        stream eventStream) (int, error) {
	sent := 0

	for {
		// taken before reading, so an event stored meanwhile still wakes it
		changed := s.store.Changed()

		events, err := s.store.EventsSince(after, page)
		if err != nil {
			return sent, status.Error(codes.Internal, err.Error())
		}

		// read after the page, so it is never below a sequence in it
		head, err := s.store.Head()
		if err != nil {
			return sent, status.Error(codes.Internal, err.Error())
		}

		for _, e := range events {
			if err := stream.Send(storedEvent(e, logID, head)); err != nil {
				return sent, err
			}

			after = e.Sequence
			sent++
		}

		if len(events) > 0 {
			continue
		}

		select {
		case <-changed:
		case <-stream.Context().Done():
			return sent, nil
		case <-s.ctx.Done():
			return sent, errStopping
		}
	}
}


// acquire takes a slot for a stream, or reports that every slot is taken.
func (s *server) acquire() bool {
	select {
	case s.slots <- struct{}{}:
		return true
	default:
		return false
	}
}


// release gives back the slot of a stream that ended.
func (s *server) release() {
	<-s.slots
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


// clientOf is the address a stream comes from, for the log.
func clientOf(stream eventStream) string {
	if p, ok := peer.FromContext(stream.Context()); ok {
		return p.Addr.String()
	}

	return "unknown"
}


// reasonOf is why a stream ended, for the log.
func reasonOf(err error) string {
	switch {
	case err == nil:
		return "client left"
	case errors.Is(err, errStopping):
		return "node stopping"
	}

	return err.Error()
}
