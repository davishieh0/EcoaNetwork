package node


// --- CODE ---

// RevokedKeyError is an event signed by a key this node saw revoked.
type RevokedKeyError struct {
	Key string
}


// Error returns the revoked key.
func (e *RevokedKeyError) Error() string {
	return "node: signing key revoked: " + e.Key
}
