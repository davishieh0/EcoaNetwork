package envelope


import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)


// --- CONSTANTS ---

// validEnvelope is well-formed in every field; nothing checks its signature.
const validEnvelope = `{
	"protocol": "ecoa-dev",
	"version": 1,
	"eventId": "evt_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	"author": "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
	"deviceKey": null,
	"eventType": "review.created",
	"createdAt": "2026-08-05T17:00:00Z",
	"payload": { "z" : 1,  "a": "keeps spacing" },
	"signature": "` + signature86 + `"
}`


// signature86 is 86 characters of base64url.
const signature86 = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"


// --- GLOBALS ---

var (
	// malformed tells whether an error is a MalformedJSONError
	malformed = is[*MalformedJSONError]

	// duplicateKey tells whether an error is a DuplicateKeyError
	duplicateKey = is[*DuplicateKeyError]

	// tooDeep tells whether an error is a TooDeepError
	tooDeep = is[*TooDeepError]

	// badNumber tells whether an error is an UnsafeNumberError
	badNumber = is[*UnsafeNumberError]

	// missingField tells whether an error is a FieldError of that fault
	missingField = isField(RequiredFieldAbsent)

	// unknownField tells whether an error is a FieldError of that fault
	unknownField = isField(FieldOutsideEnvelope)

	// badField tells whether an error is a FieldError of that fault
	badField = isField(FieldBreaksItsRule)
)


// --- CODE ---

// schemaPatterns returns the pattern of each named field or definition.
func schemaPatterns(t *testing.T) map[string]string {
	t.Helper()

	var file struct {
		Properties map[string]struct {
			Pattern string `json:"pattern"`
		} `json:"properties"`
		Defs map[string]struct {
			Pattern string `json:"pattern"`
		} `json:"$defs"`
	}

	patterns := map[string]string{}

	for _, name := range []string{"envelope", "common"} {
		path := filepath.Join("../../../schemas", name+".schema.json")

		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}

		if err := json.Unmarshal(raw, &file); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
	}

	for name, field := range file.Properties {
		patterns[name] = field.Pattern
	}

	for name, def := range file.Defs {
		patterns[name] = def.Pattern
	}

	return patterns
}


func TestPatternsAreTheOnesOfTheSchemas(t *testing.T) {
	patterns := schemaPatterns(t)

	cases := []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"protocol", protocolPattern},
		{"eventType", eventTypePattern},
		{"eventId", eventIDPattern},
		{"did", didPattern},
		{"signature", signaturePattern},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			want := patterns[c.name]

			// Act
			got := c.pattern.String()

			// Assert
			if got != want {
				t.Fatalf("pattern = %s, schema has %s", got, want)
			}
		})
	}
}


func TestValidProtocol(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"ecoa", true},
		{"ecoa-dev", true},
		{"ecoa-a1-b", true},
		{"foo", false},
		{"", false},
		{"ecoa-", false},
		{"ECOA", false},
		{"ecoa-Dev", false},
		{"ecoa_dev", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			name := c.name

			// Act
			got := ValidProtocol(name)

			// Assert
			if got != c.want {
				t.Fatalf("ValidProtocol(%q) = %v", name, got)
			}
		})
	}
}


func TestDecodeAcceptsTheEdgesOfEachField(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
	}{
		{"digits after the dot", `review.created`, `review.created_v2`},
		{"year one", `2026-08-05`, `0001-01-01`},
		{"last instant of a day", `17:00:00Z`, `23:59:59Z`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			raw := strings.Replace(validEnvelope, c.from, c.to, 1)

			// Act
			_, err := Decode([]byte(raw))

			// Assert
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
		})
	}
}


func TestDecodeReadsAValidEnvelope(t *testing.T) {
	// Arrange
	raw := []byte(validEnvelope)
	payload := `{ "z" : 1,  "a": "keeps spacing" }`

	// Act
	env, err := Decode(raw)

	// Assert
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if !bytes.Equal(env.Payload, []byte(payload)) {
		t.Fatalf("Payload = %q, want %q", env.Payload, payload)
	}

	if env.DeviceKey != nil || env.Version != 1 ||
		env.Protocol != "ecoa-dev" || env.EventType != "review.created" {
		t.Fatalf("Decode = %+v", env)
	}
}


func TestDecodeKeepsTheFieldsARuleReads(t *testing.T) {
	// Arrange
	device := "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK"
	raw := strings.Replace(validEnvelope,
	                       `"deviceKey": null`,
	                       `"deviceKey": "`+device+`"`,
	                       1)

	// Act
	env, err := Decode([]byte(raw))

	// Assert
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if env.DeviceKey == nil || *env.DeviceKey != device {
		t.Fatalf("DeviceKey = %v, want %s", env.DeviceKey, device)
	}

	if env.CreatedAt != "2026-08-05T17:00:00Z" {
		t.Fatalf("CreatedAt = %q", env.CreatedAt)
	}
}


func TestDecodeRejects(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
		want func(error) bool
	}{
		{
			"unknown field",
			`"version": 1`, `"version": 1, "x": 1`,
			unknownField,
		},
		{
			"known name in another case",
			`"version": 1`, `"Version": 1`,
			missingField,
		},
		{"deviceKey absent", `"deviceKey": null,`, ``, missingField},
		{"payload absent", `"payload": { "z" : 1,  "a": "keeps spacing" },`, ``,
			missingField},
		{
			"known name twice in two cases",
			`"version": 1`, `"version": 1, "Version": 1`,
			unknownField,
		},
		{
			"structure before the fields",
			`"version": 1`, `"version": 0, "version": 0`,
			duplicateKey,
		},
		{"protocol shape", `"ecoa-dev"`, `"foo"`, badField},
		{"eventId shape", `"evt_a`, `"evt_A`, badField},
		{"eventId null", `"eventId": "evt_` + strings.Repeat("a", 52) + `"`,
			`"eventId": null`, badField},
		{"author shape", `did:key:z6Mk`, `did:key:zQ3s`, badField},
		{
			"deviceKey shape",
			`"deviceKey": null`, `"deviceKey": ""`,
			badField,
		},
		{"eventType shape", `review.created`, `Review.Created`, badField},
		{"createdAt offset", `17:00:00Z`, `17:00:00+00:00`, badField},
		{"createdAt millis", `17:00:00Z`, `17:00:00.000Z`, badField},
		{
			"eventType with a trailing newline",
			`review.created`, `review.created\n`,
			badField,
		},
		{"eventType digit before the dot", `review.`, `review1.`, badField},
		{"signature shape", signature86, "AAAA", badField},
		{"signature padded", signature86, signature86 + "==", badField},
		{"version zero", `"version": 1`, `"version": 0`, badField},
		{"version negative", `"version": 1`, `"version": -1`, badField},
		{
			"deviceKey not a did",
			`"deviceKey": null`, `"deviceKey": "abc"`,
			badField,
		},
		{"version text", `"version": 1`, `"version": "1"`, badField},
		{"payload not object", `{ "z" : 1,  "a": "keeps spacing" }`, `[]`,
			badField},
		{"payload null", `{ "z" : 1,  "a": "keeps spacing" }`, `null`,
			badField},
		{"duplicate key", `"version": 1`, `"version": 1, "version": 1`,
			duplicateKey},
		{"fractional number", `"z" : 1`, `"z" : 1.0`, badNumber},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			raw := strings.Replace(validEnvelope, c.from, c.to, 1)

			// Act
			_, err := Decode([]byte(raw))

			// Assert
			if !matches(err, c.want) {
				t.Fatalf("Decode = %v, want another error", err)
			}
		})
	}
}


func TestDecodeRejectsWhatIsNotAnObject(t *testing.T) {
	for _, raw := range []string{"", "[]", "null", "1", `"x"`, "not json"} {
		t.Run(raw, func(t *testing.T) {
			// Arrange
			input := []byte(raw)

			// Act
			_, err := Decode(input)

			// Assert
			if err == nil {
				t.Fatal("Decode accepted it")
			}
		})
	}
}


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


// isField returns a check for a field error with fault.
func isField(fault FieldFault) func(error) bool {
	return func(err error) bool {
		var field *FieldError
		return errors.As(err, &field) && field.Fault == fault
	}
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
		{"largest integer", `{"n":9007199254740991}`, nil},
		{"smallest integer", `{"n":-9007199254740991}`, nil},
		{"zero", `{"n":0}`, nil},
		{"negative zero", `{"n":-0}`, badNumber},
		{"integer past the top", `{"n":9007199254740992}`,
			badNumber},
		{"integer past the bottom", `{"n":-9007199254740992}`,
			badNumber},
		{"integer rounded by the canonicalizer", `{"n":9007199254740993}`,
			badNumber},
		{"integer past 64 bits", `{"n":99999999999999999999}`,
			badNumber},
		{"fraction", `{"n":1.5}`, badNumber},
		{"integral fraction", `{"n":1.0}`, badNumber},
		{"exponent", `{"n":1e2}`, badNumber},
		{"deep in a payload", `{"p":{"q":[{"n":1.0}]}}`, badNumber},
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
