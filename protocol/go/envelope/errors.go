package envelope


import "fmt"


// --- CODE ---

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
