package schema


import "errors"


// --- GLOBALS ---

var (
	// badPayload tells whether an error is an InvalidPayloadError
	badPayload = is[*InvalidPayloadError]
)


// --- CODE ---

// is tells whether err is or wraps an error of type T.
func is[T error](err error) bool {
	var target T
	return errors.As(err, &target)
}
