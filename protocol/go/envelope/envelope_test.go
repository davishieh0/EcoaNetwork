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
	"time"
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
	// tooLarge tells whether an error is a TooLargeError
	tooLarge = is[*TooLargeError]

	// otherNetwork tells whether an error is an OtherNetworkError
	otherNetwork = is[*OtherNetworkError]

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

	// futureCreatedAt tells whether an error is a FutureCreatedAtError
	futureCreatedAt = is[*FutureCreatedAtError]
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
		{"timestamp", createdAtPattern},
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


func TestDecodeKeepsTheReceivedBytes(t *testing.T) {
	// Arrange
	raw := []byte(strings.Replace(validEnvelope, ":", " : ", 1))

	// Act
	env, err := Decode(raw)

	// Assert
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if !bytes.Equal(env.Raw(), raw) {
		t.Fatalf("Raw = %q, want %q", env.Raw(), raw)
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
		{"createdAt not a date", `2026-08-05`, `2026-13-45`, badField},
		{"createdAt february 30", `2026-08-05`, `2026-02-30`, badField},
		{"createdAt year zero", `2026-08-05`, `0000-08-05`, badField},
		{"createdAt leap second", `17:00:00Z`, `16:59:60Z`, badField},
		{"createdAt hour 24", `17:00:00Z`, `24:00:00Z`, badField},
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


func TestMaxSizeIsSixteenKibibytes(t *testing.T) {
	// Arrange
	want := 16384

	// Act
	got := MaxSize

	// Assert
	if got != want {
		t.Fatalf("MaxSize = %d, want %d", got, want)
	}
}


func TestCheckSize(t *testing.T) {
	cases := []struct {
		name  string
		size  int
		limit int
		want  func(error) bool
	}{
		{"exactly 16 KB", MaxSize, MaxSize, nil},
		{"16 KB plus one", MaxSize + 1, MaxSize, tooLarge},
		{"stricter node limit", 4097, 4096, tooLarge},
		{"limit above the ceiling", MaxSize + 1, MaxSize * 2, tooLarge},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			raw := make([]byte, c.size)

			// Act
			err := checkSize(raw, c.limit)

			// Assert
			if !matches(err, c.want) {
				t.Fatalf("checkSize = %v, want another error", err)
			}
		})
	}
}


func TestReadHeader(t *testing.T) {
	// Arrange
	raw := []byte(`{"version":1,"protocol":"ecoa-dev","payload":{"x":[[1]]}}`)

	// Act
	header, err := readHeader(raw)

	// Assert
	if err != nil {
		t.Fatalf("readHeader: %v", err)
	}

	if *header.Protocol != "ecoa-dev" || *header.Version != 1 {
		t.Fatalf("readHeader = %q, %d", *header.Protocol, *header.Version)
	}
}


func TestReadHeaderLeavesToDecodeWhatItCannotCompare(t *testing.T) {
	cases := []struct {
		name         string
		raw          string
		wantProtocol bool
		wantVersion  bool
	}{
		{"protocol absent", `{"version":1}`, false, true},
		{"protocol not text", `{"protocol":1,"version":1}`, false, true},
		{"protocol null", `{"protocol":null,"version":1}`, false, true},
		{"protocol in capitals", `{"PROTOCOL":"e","version":1}`, false, true},
		{"version absent", `{"protocol":"e"}`, true, false},
		{"version in capitals", `{"protocol":"e","VERSION":1}`, true, false},
		{"version text", `{"protocol":"e","version":"1"}`, true, false},
		{"version null", `{"protocol":"e","version":null}`, true, false},
		{"version fraction", `{"protocol":"e","version":1.5}`, true, false},
		{"version 1.0", `{"protocol":"e","version":1.0}`, true, false},
		{"version -0", `{"protocol":"e","version":-0}`, true, false},
		{
			"version past the safe range",
			`{"protocol":"e","version":9007199254740992}`, true, false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			raw := []byte(c.raw)

			// Act
			header, err := readHeader(raw)

			// Assert
			if err != nil {
				t.Fatalf("readHeader: %v", err)
			}

			if (header.Protocol != nil) != c.wantProtocol {
				t.Fatalf("Protocol = %v", header.Protocol)
			}

			if (header.Version != nil) != c.wantVersion {
				t.Fatalf("Version = %v", header.Version)
			}
		})
	}
}


func TestReadHeaderRejects(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want func(error) bool
	}{
		{"not json", `not json`, malformed},
		{"not an object", `[]`, malformed},
		{"byte order mark", "\xef\xbb\xbf{}", malformed},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			raw := []byte(c.raw)

			// Act
			_, err := readHeader(raw)

			// Assert
			if !matches(err, c.want) {
				t.Fatalf("readHeader = %v, want another error", err)
			}
		})
	}
}


func TestCheckNetwork(t *testing.T) {
	// Arrange
	network := "ecoa-dev"
	mine, theirs, sameLength := "ecoa-dev", "ecoa-testnet", "ecoa-foo"

	// Act
	same := checkNetwork(&mine, network)
	other := checkNetwork(&theirs, network)
	close := checkNetwork(&sameLength, network)
	unread := checkNetwork(nil, network)

	// Assert
	if same != nil || unread != nil {
		t.Fatalf("same network: %v, unread: %v", same, unread)
	}

	if !otherNetwork(other) {
		t.Fatalf("other network = %v, want otherNetwork", other)
	}

	if !otherNetwork(close) {
		t.Fatalf("same length = %v, want otherNetwork", close)
	}
}


func TestCheckVersion(t *testing.T) {
	// Arrange
	supported, later, earlier := 1, 2, 0

	// Act
	current := checkVersion(&supported)
	next := checkVersion(&later)
	before := checkVersion(&earlier)
	unread := checkVersion(nil)

	// Assert
	if current != nil || unread != nil {
		t.Fatalf("version 1: %v, unread: %v", current, unread)
	}

	if !otherNetwork(next) {
		t.Fatalf("version 2 = %v, want otherNetwork", next)
	}

	if !otherNetwork(before) {
		t.Fatalf("version 0 = %v, want otherNetwork", before)
	}
}


func TestCheckCreatedAt(t *testing.T) {
	now := time.Date(2026, 8, 5, 17, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		createdAt string
		want      func(error) bool
	}{
		{"now", "2026-08-05T17:00:00Z", nil},
		{"five minutes ahead", "2026-08-05T17:05:00Z", nil},
		{"five minutes and a second", "2026-08-05T17:05:01Z",
			futureCreatedAt},
		{"years ago", "2020-01-01T00:00:00Z", nil},
		{"not a date", "yesterday", badField},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			createdAt := c.createdAt

			// Act
			err := checkCreatedAt(createdAt, now)

			// Assert
			if !matches(err, c.want) {
				t.Fatalf("checkCreatedAt = %v, want another error", err)
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


// rules returns the rules of a node that accepts validEnvelope.
func rules(t *testing.T) Rules {
	t.Helper()

	now, err := time.Parse(time.RFC3339, "2026-08-05T17:00:00Z")
	if err != nil {
		t.Fatalf("parse clock: %v", err)
	}

	return Rules{
		Network: "ecoa-dev",
		MaxSize: MaxSize,
		Now:     func() time.Time { return now },
	}
}


// with replaces the first old of validEnvelope with new.
func with(old, new string) []byte {
	return []byte(strings.Replace(validEnvelope, old, new, 1))
}


func TestOpenReturnsTheEnvelopeAndItsBytes(t *testing.T) {
	// Arrange
	raw := []byte(validEnvelope)

	// Act
	env, err := Open(raw, rules(t))

	// Assert
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if env.EventType != "review.created" || string(env.Raw()) != validEnvelope {
		t.Fatalf("Open = %+v", env)
	}
}


func TestOpenRejects(t *testing.T) {
	network := `"protocol": "ecoa-dev"`
	version := `"version": 1`
	other := `"protocol": "ecoa-other"`

	cases := []struct {
		name string
		raw  []byte
		want func(error) bool
	}{
		{
			"too large",
			[]byte(strings.Repeat(" ", MaxSize+1)),
			tooLarge,
		},
		{"not json", []byte("not json"), malformed},
		{"other network", with(network, other), otherNetwork},
		{
			"other version",
			with(version, `"version": 2`),
			otherNetwork,
		},
		{
			"other network, whatever the version",
			[]byte(strings.Replace(string(with(network, other)),
			                       version, `"version": "x"`, 1)),
			otherNetwork,
		},
		{
			"version as text",
			with(version, `"version": "1"`),
			badField,
		},
		{
			"version as a fraction",
			with(version, `"version": 1.5`),
			badNumber,
		},
		{
			"protocol as a number",
			with(network, `"protocol": 1`),
			badField,
		},
		{
			"protocol twice in two cases",
			with(network, network+`, "PROTOCOL": "ecoa-other"`),
			unknownField,
		},
		{
			"unknown field",
			with(version, version+`, "extra": 1`),
			unknownField,
		},
		{
			"future createdAt",
			with("2026-08-05T17:00:00Z", "2026-08-05T17:05:01Z"),
			futureCreatedAt,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			cfg := rules(t)

			// Act
			_, err := Open(c.raw, cfg)

			// Assert
			if !matches(err, c.want) {
				t.Fatalf("Open = %v, want another error", err)
			}
		})
	}
}


func TestOpenHoldsTheConfiguredLimit(t *testing.T) {
	// Arrange
	cfg := rules(t)
	cfg.MaxSize = 10

	// Act
	_, err := Open([]byte(validEnvelope), cfg)

	// Assert
	if !tooLarge(err) {
		t.Fatalf("Open = %v, want tooLarge", err)
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
