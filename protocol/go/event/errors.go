package event


import "fmt"


// --- CODE ---

// EventIDMismatchError is an eventId that differs from the recomputed digest.
type EventIDMismatchError struct {
	Received string
	Computed string
}


// Error returns both eventIds.
func (e *EventIDMismatchError) Error() string {
	return fmt.Sprintf("event: eventId %s, computed %s",
	                   e.Received,
	                   e.Computed)
}
