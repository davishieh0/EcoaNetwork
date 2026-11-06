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


// OverLimitError is an event whose author is over the admission limit.
type OverLimitError struct {
	Author string
}


// Error returns the author over the limit.
func (e *OverLimitError) Error() string {
	return "node: author over the admission limit: " + e.Author
}
