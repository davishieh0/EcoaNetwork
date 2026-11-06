package references


import "fmt"


// --- CODE ---

// AuthorMismatchError is an event acting on a target of another identity.
type AuthorMismatchError struct {
	EventType string
	Target    string
	Author    string
	Owner     string
}


// Error returns both authors and the target.
func (e *AuthorMismatchError) Error() string {
	return fmt.Sprintf("references: %s by %s acts on %s of %s",
	                   e.EventType,
	                   e.Author,
	                   e.Target,
	                   e.Owner)
}
