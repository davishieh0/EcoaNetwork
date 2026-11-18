package server


import (
	pb "github.com/eduardokairalla1/EcoaNetwork/internal/protobuf"
	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
)


// --- CODE ---

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
