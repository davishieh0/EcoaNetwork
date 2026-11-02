// Package envelope decodes, canonicalizes and verifies a signed event.
package envelope


import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"
)


// --- CONSTANTS ---

// createdAtLayout is RFC 3339 UTC with a literal Z and second precision.
const createdAtLayout = "2006-01-02T15:04:05Z"


// MaxSize is the 16 KB protocol ceiling, in bytes.
const MaxSize = 16 << 10


// supportedVersion is the only envelope version v1 implements.
const supportedVersion = 1


// maxClockSkew is how far ahead of the local clock createdAt may be.
const maxClockSkew = 5 * time.Minute


// maxDepth is the nesting cap of the protocol.
const maxDepth = 32


// maxInteger is the largest number magnitude the protocol accepts, 2^53-1.
const maxInteger = 1<<53 - 1


// --- GLOBALS ---

// names is the closed set of top-level envelope fields.
var names = []string{
	"protocol",
	"version",
	"eventId",
	"author",
	"deviceKey",
	"eventType",
	"createdAt",
	"payload",
	"signature",
}


// the patterns of schemas/envelope.schema.json and schemas/common.schema.json
var (
	protocolPattern = regexp.MustCompile(
		`^ecoa(-[a-z0-9]([a-z0-9-]*[a-z0-9])?)?$`,
	)
	eventIDPattern = regexp.MustCompile(`^evt_[a-z2-7]{52}$`)
	didPattern     = regexp.MustCompile(
		`^did:key:z6Mk[1-9A-HJ-NP-Za-km-z]{44}$`,
	)
	eventTypePattern = regexp.MustCompile(`^[a-z]+\.[a-z0-9_]+$`)
	createdAtPattern = regexp.MustCompile(
		`^([1-9][0-9]{3}|0[1-9][0-9]{2}|00[1-9][0-9]|000[1-9])` +
			`-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])` +
			`T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]Z$`,
	)
	signaturePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{86}$`)
)


// --- CODE ---

// Envelope is one event as received, before any trust in its contents.
type Envelope struct {
	Protocol  string
	Version   int
	EventID   string
	Author    string
	DeviceKey *string
	EventType string
	CreatedAt string
	Payload   json.RawMessage
	Signature string
}


// header is the protocol and version of raw, nil where Decode reports.
type header struct {
	// Protocol is nil when absent or not a string
	Protocol *string

	// Version is nil when absent or not an integer
	Version *int
}


// frame is one open object or array.
type frame struct {
	// isObject tells an object from an array
	isObject bool

	// expectKey is true when the next string of an object is a key
	expectKey bool

	// keys holds the keys already seen in an object
	keys map[string]bool
}


// walker follows the tokens of a document without recursion.
type walker struct {
	// frames is the stack of containers still open
	frames []frame

	// started is true once the top-level value has begun
	started bool
}


// Decode turns raw bytes into an envelope, or a rejection.
func Decode(raw []byte) (*Envelope, error) {
	if err := checkStructure(raw); err != nil {
		return nil, err
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, &MalformedJSONError{Detail: err.Error()}
	}

	if err := checkClosedSet(fields); err != nil {
		return nil, err
	}

	var env Envelope
	if err := readFields(fields, &env); err != nil {
		return nil, err
	}

	return &env, nil
}


// ValidProtocol tells whether name is ecoa or ecoa-<name>.
func ValidProtocol(name string) bool {
	return protocolPattern.MatchString(name)
}


// checkClosedSet rejects a missing or an unknown top-level field.
func checkClosedSet(fields map[string]json.RawMessage) error {
	for _, name := range names {
		if _, ok := fields[name]; !ok {
			return &FieldError{Field: name, Fault: RequiredFieldAbsent}
		}
	}

	if len(fields) == len(names) {
		return nil
	}

	for name := range fields {
		if !isKnown(name) {
			return &FieldError{Field: name, Fault: FieldOutsideEnvelope}
		}
	}

	return nil
}


// isKnown tells whether name belongs to the closed set, byte for byte.
func isKnown(name string) bool {
	for _, known := range names {
		if name == known {
			return true
		}
	}

	return false
}


// readFields fills env and checks each field against its rule.
func readFields(fields map[string]json.RawMessage, env *Envelope) error {
	texts := []struct {
		name    string
		target  *string
		pattern *regexp.Regexp
	}{
		{"protocol", &env.Protocol, protocolPattern},
		{"eventId", &env.EventID, eventIDPattern},
		{"author", &env.Author, didPattern},
		{"eventType", &env.EventType, eventTypePattern},
		{"createdAt", &env.CreatedAt, createdAtPattern},
		{"signature", &env.Signature, signaturePattern},
	}

	for _, text := range texts {
		raw := fields[text.name]
		if !readText(raw, text.target, text.pattern) {
			return &FieldError{Field: text.name, Fault: FieldBreaksItsRule}
		}
	}

	if _, err := time.Parse(createdAtLayout, env.CreatedAt); err != nil {
		return &FieldError{Field: "createdAt", Fault: FieldBreaksItsRule}
	}

	if err := readVersion(fields["version"], env); err != nil {
		return err
	}

	if err := readDeviceKey(fields["deviceKey"], env); err != nil {
		return err
	}

	return readPayload(fields["payload"], env)
}


// readText reads a required string and tells whether it matches pattern.
func readText(raw json.RawMessage,
              target *string,
              pattern *regexp.Regexp) bool {
	if err := json.Unmarshal(raw, target); err != nil {
		return false
	}

	return pattern.MatchString(*target)
}


// readVersion reads the envelope version, an integer from 1 up.
func readVersion(raw json.RawMessage, env *Envelope) error {
	if err := json.Unmarshal(raw, &env.Version); err != nil {
		return &FieldError{Field: "version", Fault: FieldBreaksItsRule}
	}

	if env.Version < 1 {
		return &FieldError{Field: "version", Fault: FieldBreaksItsRule}
	}

	return nil
}


// readDeviceKey reads the field that is required but nullable.
func readDeviceKey(raw json.RawMessage, env *Envelope) error {
	if err := json.Unmarshal(raw, &env.DeviceKey); err != nil {
		return &FieldError{Field: "deviceKey", Fault: FieldBreaksItsRule}
	}

	if env.DeviceKey != nil && !didPattern.MatchString(*env.DeviceKey) {
		return &FieldError{Field: "deviceKey", Fault: FieldBreaksItsRule}
	}

	return nil
}


// readPayload keeps the payload bytes as received, once known to be an object.
func readPayload(raw json.RawMessage, env *Envelope) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return &FieldError{Field: "payload", Fault: FieldBreaksItsRule}
	}

	env.Payload = raw
	return nil
}


// checkSize rejects raw above limit, which never exceeds the protocol ceiling.
func checkSize(raw []byte, limit int) error {
	if limit > MaxSize {
		limit = MaxSize
	}

	if len(raw) > limit {
		return &TooLargeError{Size: len(raw), Limit: limit}
	}

	return nil
}


// readHeader reads protocol and version, under their exact keys.
func readHeader(raw []byte) (header, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return header{}, &MalformedJSONError{Detail: err.Error()}
	}

	return header{
		Protocol: headerProtocol(fields["protocol"]),
		Version:  headerVersion(fields["version"]),
	}, nil
}


// headerProtocol returns raw when it is a string, or else nil.
func headerProtocol(raw json.RawMessage) *string {
	var protocol *string
	if err := json.Unmarshal(raw, &protocol); err != nil {
		return nil
	}

	return protocol
}


// headerVersion returns raw when it is an integer of the protocol, or else nil.
func headerVersion(raw json.RawMessage) *int {
	version, err := strconv.Atoi(string(raw))
	if err != nil || checkNumber(json.Number(raw)) != nil {
		return nil
	}

	return &version
}


// checkNetwork rejects a protocol that is not this network.
func checkNetwork(protocol *string, network string) error {
	if protocol != nil && *protocol != network {
		return &OtherNetworkError{Protocol: *protocol}
	}

	return nil
}


// checkVersion rejects a version this implementation does not support.
func checkVersion(version *int) error {
	if version != nil && *version != supportedVersion {
		return &OtherNetworkError{Version: version}
	}

	return nil
}


// checkCreatedAt rejects a createdAt more than 5 minutes ahead of now.
func checkCreatedAt(createdAt string, now time.Time) error {
	at, err := time.Parse(createdAtLayout, createdAt)
	if err != nil {
		return &FieldError{Field: "createdAt", Fault: FieldBreaksItsRule}
	}

	if at.After(now.Add(maxClockSkew)) {
		return &FutureCreatedAtError{CreatedAt: createdAt}
	}

	return nil
}


// checkStructure rejects input that would leave a choice to the parser.
func checkStructure(raw []byte) error {
	if !utf8.Valid(raw) {
		return &MalformedJSONError{Detail: "invalid UTF-8"}
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var w walker

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return w.finish()
		}

		if err != nil {
			return &MalformedJSONError{Detail: err.Error()}
		}

		if err := w.visit(token); err != nil {
			return err
		}
	}
}


// finish rejects a document that ended with nothing or with something open.
func (w *walker) finish() error {
	if !w.started || len(w.frames) > 0 {
		return &MalformedJSONError{Detail: "unexpected end"}
	}

	return nil
}


// visit checks one token and updates the stack.
func (w *walker) visit(token json.Token) error {
	// a value after the top-level one has closed
	if w.started && len(w.frames) == 0 {
		return &MalformedJSONError{Detail: "trailing data"}
	}

	w.started = true

	switch t := token.(type) {
	case json.Delim:
		return w.delimiter(t)

	case string:
		return w.text(t)

	case json.Number:
		if err := checkNumber(t); err != nil {
			return err
		}
	}

	w.valueDone()
	return nil
}


// delimiter opens or closes a container.
func (w *walker) delimiter(d json.Delim) error {
	if d == '}' || d == ']' {
		w.frames = w.frames[:len(w.frames)-1]
		w.valueDone()
		return nil
	}

	if len(w.frames) == maxDepth {
		return &TooDeepError{Limit: maxDepth}
	}

	opened := frame{}
	if d == '{' {
		opened = frame{
			isObject:  true,
			expectKey: true,
			keys:      map[string]bool{},
		}
	}

	w.frames = append(w.frames, opened)
	return nil
}


// text handles a string, which is either a key or a value.
func (w *walker) text(s string) error {
	if len(w.frames) == 0 || !w.frames[len(w.frames)-1].expectKey {
		w.valueDone()
		return nil
	}

	top := &w.frames[len(w.frames)-1]

	if top.keys[s] {
		return &DuplicateKeyError{Key: s}
	}

	top.keys[s] = true
	top.expectKey = false

	return nil
}


// valueDone records that the value of the enclosing object has ended.
func (w *walker) valueDone() {
	if len(w.frames) > 0 && w.frames[len(w.frames)-1].isObject {
		w.frames[len(w.frames)-1].expectKey = true
	}
}


// checkNumber accepts only an integer spelling within the safe range.
func checkNumber(n json.Number) error {
	value, err := strconv.ParseInt(n.String(), 10, 64)
	if err != nil || value > maxInteger || value < -maxInteger || n == "-0" {
		return &UnsafeNumberError{Number: n.String()}
	}

	return nil
}
