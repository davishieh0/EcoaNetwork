package node


import (
	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
)


// --- CODE ---

// lookup answers references.Check from the store and the identity tracker.
type lookup struct {
	store      *store.Store
	identities *identity.Tracker
}


// Author returns who signed the stored event eventID.
func (l lookup) Author(eventID string) (string, bool, error) {
	raw, err := l.store.GetEvent(eventID)
	if err != nil || raw == nil {
		return "", false, err
	}

	env, err := envelope.Decode(raw)
	if err != nil {
		return "", false, err
	}

	return env.Author, true, nil
}


// Resolve returns the current key of the identity of key.
func (l lookup) Resolve(key string) (string, bool) {
	return l.identities.Resolve(key)
}
