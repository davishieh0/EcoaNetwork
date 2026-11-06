package node


import (
	"fmt"

	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
)


// --- CONSTANTS ---

// replayPage is the events read from the log at a time on rebuilding.
const replayPage = 256


// --- CODE ---

// movesIdentity reports whether eventType rotates or revokes a key.
func movesIdentity(eventType string) bool {
	return eventType == "identity.key_rotated" ||
	       eventType == "identity.key_revoked"
}


// rebuildIdentity replays the rotations and revocations of the log into t, in
// the order this node received them.
func rebuildIdentity(s *store.Store, t *identity.Tracker) error {
	var cursor uint64

	for {
		page, err := s.EventsSince(cursor, replayPage)
		if err != nil {
			return err
		}

		for _, logged := range page {
			if err := replay(t, logged); err != nil {
				return fmt.Errorf("replay sequence %d: %w",
				                  logged.Sequence, err)
			}

			cursor = logged.Sequence
		}

		if len(page) < replayPage {
			return nil
		}
	}
}


// replay observes one logged event in t when it moves an identity.
func replay(t *identity.Tracker, logged store.StoredEvent) error {
	env, err := envelope.Decode(logged.Raw)
	if err != nil {
		return err
	}

	if !movesIdentity(env.EventType) {
		return nil
	}

	return t.Observe(env)
}
