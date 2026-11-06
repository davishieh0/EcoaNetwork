package identity


import "fmt"


// --- CODE ---

// SuccessorObservedError is a rotation of a key that already has a successor.
type SuccessorObservedError struct {
	Author    string
	Successor string
}


// RotationCycleError is a rotation whose successor leads back to its author.
type RotationCycleError struct {
	Author    string
	Successor string
}


// UnsupportedEventError is an event the tracker does not observe.
type UnsupportedEventError struct {
	EventType string
}


// Error returns the key and the successor that came too late.
func (e *SuccessorObservedError) Error() string {
	return fmt.Sprintf("identity: successor already observed for %s, not %s",
	                   e.Author,
	                   e.Successor)
}


// Error returns the key and the successor that close the cycle.
func (e *RotationCycleError) Error() string {
	return fmt.Sprintf("identity: cycle in succession chain: %s to %s",
	                   e.Author,
	                   e.Successor)
}


// Error returns the event type.
func (e *UnsupportedEventError) Error() string {
	return "identity: not a rotation or revocation: " + e.EventType
}
