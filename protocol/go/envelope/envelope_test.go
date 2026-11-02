package envelope


import (
	"errors"
	"strings"
	"testing"
)


// --- GLOBALS ---

var (
	// malformed tells whether an error is a MalformedJSONError
	malformed = is[*MalformedJSONError]

	// duplicateKey tells whether an error is a DuplicateKeyError
	duplicateKey = is[*DuplicateKeyError]

	// tooDeep tells whether an error is a TooDeepError
	tooDeep = is[*TooDeepError]
)


// --- CODE ---

// is tells whether err is or wraps an error of type T.
func is[T error](err error) bool {
	var target T
	return errors.As(err, &target)
}


// matches tells whether err passes want, or is nil when want is nil.
func matches(err error, want func(error) bool) bool {
	if want == nil {
		return err == nil
	}

	return want(err)
}


// nested returns levels of array nesting around nothing.
func nested(levels int) string {
	return strings.Repeat("[", levels) + strings.Repeat("]", levels)
}


func TestCheckStructure(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want func(error) bool
	}{
		{"flat object", `{"a":1}`, nil},
		{"same key in sibling objects", `{"a":{"k":1},"b":{"k":2}}`, nil},
		{"string value equal to a key", `{"a":"a","b":"a"}`, nil},
		{"strings in an array", `{"a":["x","x"]}`, nil},
		{"depth 32", nested(32), nil},
		{"depth 33", nested(33), tooDeep},
		{"8000 levels in under 16 KB", nested(8000), tooDeep},
		{"objects past depth 32",
			strings.Repeat(`{"a":`, 33) + "1" + strings.Repeat("}", 33),
			tooDeep},
		{"duplicate at the top", `{"a":1,"a":2}`, duplicateKey},
		{"duplicate inside", `{"p":[{"a":1,"a":1}]}`, duplicateKey},
		{"duplicate after an object", `{"a":{},"a":1}`, duplicateKey},
		{"duplicate after an array", `{"a":[],"a":1}`, duplicateKey},
		{"duplicate after true", `{"a":true,"a":1}`, duplicateKey},
		{"duplicate after null", `{"a":null,"a":1}`, duplicateKey},
		{"strings at the top of an array", `["x","x","x"]`, nil},
		{"duplicate through an escape", `{"a":1,"\u0061":2}`, duplicateKey},
		{"trailing data", `{"a":1} x`, malformed},
		{"second value", `{"a":1} {"b":2}`, malformed},
		{"truncated", `{"a":1`, malformed},
		{"empty", ``, malformed},
		{"trailing comma", `{"a":1,}`, malformed},
		{"invalid UTF-8", "{\"a\":\"\xff\"}", malformed},
		{"byte order mark", "\xef\xbb\xbf{\"a\":1}", malformed},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			raw := []byte(c.raw)

			// Act
			err := checkStructure(raw)

			// Assert
			if !matches(err, c.want) {
				t.Fatalf("checkStructure = %v, want another error", err)
			}
		})
	}
}
