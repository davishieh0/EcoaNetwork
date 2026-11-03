package event


import "fmt"


// --- CODE ---

// EventIDMismatchError is an eventId that differs from the recomputed digest.
type EventIDMismatchError struct {
	Received string
	Computed string
}


// InvalidSignatureError is a signature that does not verify against the author.
type InvalidSignatureError struct {
	Detail string
}


// DeviceKeyNotSupportedError is a filled deviceKey, which v1 lacks.
type DeviceKeyNotSupportedError struct {
	DeviceKey string
}


// Error returns both eventIds.
func (e *EventIDMismatchError) Error() string {
	return fmt.Sprintf("event: eventId %s, computed %s",
	                   e.Received,
	                   e.Computed)
}


// Error returns why the signature fails.
func (e *InvalidSignatureError) Error() string {
	return "event: invalid signature: " + e.Detail
}


// Error returns the device key.
func (e *DeviceKeyNotSupportedError) Error() string {
	return "event: device keys are not supported in v1: " + e.DeviceKey
}
