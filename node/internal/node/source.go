package node


import (
	"github.com/eduardokairalla1/EcoaNetwork/internal/p2p"
	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
)


// --- CODE ---

// source serves the store to peers as the log the sync handler reads.
type source struct {
	store *store.Store
}


// EventsSince returns up to limit stored events above cursor, in order.
func (s source) EventsSince(cursor uint64,
                            limit int) ([]p2p.LoggedEvent, error) {
	found, err := s.store.EventsSince(cursor, limit)
	if err != nil {
		return nil, err
	}

	logged := make([]p2p.LoggedEvent, 0, len(found))

	for _, event := range found {
		logged = append(logged, p2p.LoggedEvent{
			Seq: event.Sequence,
			Raw: event.Raw,
		})
	}

	return logged, nil
}


// GetEvent returns the event stored under id, or nil when there is none.
func (s source) GetEvent(id string) ([]byte, error) {
	return s.store.GetEvent(id)
}


// Tombstone reports whether id was pruned from the store.
func (s source) Tombstone(id string) (bool, error) {
	_, found, err := s.store.Tombstone(id)

	return found, err
}
