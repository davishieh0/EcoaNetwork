package envelope


import "fmt"


// --- CONSTANTS ---

const (
	// RequiredFieldAbsent is a required field that is absent.
	RequiredFieldAbsent FieldFault = iota + 1

	// FieldOutsideEnvelope is a top-level field outside the closed set.
	FieldOutsideEnvelope

	// FieldBreaksItsRule is a present field that breaks its rule.
	FieldBreaksItsRule
)


// --- GLOBALS ---

// faultTexts is the text of each field fault.
var faultTexts = map[FieldFault]string{
	RequiredFieldAbsent:  "missing",
	FieldOutsideEnvelope: "unknown",
	FieldBreaksItsRule:   "malformed",
}


// --- CODE ---

// TooLargeError is an event above the size limit.
type TooLargeError struct {
	Size  int
	Limit int
}


// OtherNetworkError is an event of another network or an unknown version.
type OtherNetworkError struct {
	Protocol string
	Version  *int
}


// MalformedJSONError is input that is not one well-formed JSON object.
type MalformedJSONError struct {
	Detail string
}


// DuplicateKeyError is an object key repeated at any level.
type DuplicateKeyError struct {
	Key string
}


// TooDeepError is nesting above the limit.
type TooDeepError struct {
	Limit int
}


// UnsafeNumberError is a number that is not a safe integer.
type UnsafeNumberError struct {
	Number string
}


// FieldFault is what is wrong with an envelope field.
type FieldFault int


// FieldError is an envelope field that is missing, unknown or invalid.
type FieldError struct {
	Field string
	Fault FieldFault
}


// Error returns the size and the limit.
func (e *TooLargeError) Error() string {
	return fmt.Sprintf("envelope: %d bytes, limit %d", e.Size, e.Limit)
}


// Error returns the network or the version the event names.
func (e *OtherNetworkError) Error() string {
	if e.Version != nil {
		return fmt.Sprintf("envelope: unsupported version %d", *e.Version)
	}

	return fmt.Sprintf("envelope: event from network %q", e.Protocol)
}


// Error returns where the input stops being JSON.
func (e *MalformedJSONError) Error() string {
	return "envelope: malformed JSON: " + e.Detail
}


// Error returns the repeated key.
func (e *DuplicateKeyError) Error() string {
	return fmt.Sprintf("envelope: duplicate object key %q", e.Key)
}


// Error returns the nesting limit.
func (e *TooDeepError) Error() string {
	return fmt.Sprintf("envelope: nesting deeper than %d", e.Limit)
}


// Error returns the number as written.
func (e *UnsafeNumberError) Error() string {
	return "envelope: number is not a safe integer: " + e.Number
}


// Error returns the fault and the field.
func (e *FieldError) Error() string {
	fault := faultTexts[e.Fault]

	return fmt.Sprintf("envelope: %s field %q", fault, e.Field)
}
