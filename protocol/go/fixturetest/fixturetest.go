// Package fixturetest runs the cases of protocol/fixtures/ against a rule.
package fixturetest


import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/testevent"
	"github.com/santhosh-tekuri/jsonschema/v6"
)


// --- CONSTANTS ---

// network is the network of every event built from a case.
const network = "ecoa-dev"


// cnpjDigits is the length of the digits of a cnpj entity identifier.
const cnpjDigits = 14


// defaultAt is the createdAt of an arrival that declares none.
const defaultAt = "2026-01-01T00:00:00Z"


// --- GLOBALS ---

// eventFields are the payload fields that name another event.
var eventFields = []string{"target", "replaces"}


// authorFields are the payload fields that name an author.
var authorFields = []string{"successor", "subject"}


// idEncoding is base32 lowercase without padding, as in an eventId.
var idEncoding = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567").
	WithPadding(base32.NoPadding)


// referenceKeys are the expect keys whose values name events or authors.
var referenceKeys = []string{
	"head", "chain", "losingBranch", "withdrawnChain", "responsesOnHead",
	"respondedTo", "countedVote", "stored", "counted", "labelledProposals",
	"retained",
}


// --- CODE ---

// File is one fixture file.
type File struct {
	Topic string `json:"topic"`
	Rule  string `json:"rule"`
	Cases []Case `json:"cases"`
}


// Case is one behavioural case of a file.
type Case struct {
	Name       string                     `json:"name"`
	Why        string                     `json:"why"`
	Since      string                     `json:"since"`
	Identities map[string]json.RawMessage `json:"identities"`
	Arrival    []Arrival                  `json:"arrival"`
	Expect     map[string]any             `json:"expect"`
}


// Arrival is one event, in the order the node observed it.
type Arrival struct {
	ID       string          `json:"id"`
	Author   string          `json:"author"`
	SignedBy string          `json:"signedBy"`
	Type     string          `json:"type"`
	At       string          `json:"at"`
	Payload  json.RawMessage `json:"payload"`
}


// Built is one arrival of a case as a signed event.
type Built struct {
	// ID is the symbolic id the fixture gave the arrival.
	ID string

	// Raw is the signed envelope.
	Raw []byte

	// Env is Raw decoded.
	Env *envelope.Envelope
}


// builder signs the arrivals of one case, each after the events it names.
type builder struct {
	t        *testing.T
	arrivals map[string]Arrival
	built    map[string]Built
}


// Author returns the did:key of the symbolic author name.
func Author(name string) string {
	return identity.FromKey(key(name).Public().(ed25519.PublicKey)).String()
}


// entityID returns the entity identifier a fixture's short cnpj stands for.
func entityID(name string) string {
	digits, ok := strings.CutPrefix(name, "cnpj:")
	if !ok {
		return name
	}

	return "cnpj:" + strings.Repeat("0", cnpjDigits-len(digits)) + digits
}


// EventID returns the eventId a fixture gives to a name no arrival carries.
func EventID(name string) string {
	digest := sha256.Sum256([]byte(name))
	return "evt_" + idEncoding.EncodeToString(digest[:])
}


// Events signs the arrivals of c, in arrival order, naming real ids and keys.
func (c Case) Events(t *testing.T) []Built {
	t.Helper()

	b := &builder{
		t:        t,
		arrivals: map[string]Arrival{},
		built:    map[string]Built{},
	}

	for _, a := range c.Arrival {
		b.arrivals[a.ID] = a
	}

	events := make([]Built, 0, len(c.Arrival))
	for _, a := range c.Arrival {
		events = append(events, b.build(a.ID))
	}

	return events
}


// List returns the strings of the expect field name, none when it is absent.
func (c Case) List(name string) []string {
	var ids []string

	items, _ := c.Expect[name].([]any)
	for _, item := range items {
		id, _ := item.(string)
		ids = append(ids, id)
	}

	return ids
}


// Rejected returns the ids named by the rejected entries of expect.
func (c Case) Rejected() []string {
	var ids []string

	items, _ := c.Expect["rejected"].([]any)
	for _, item := range items {
		entry, _ := item.(map[string]any)
		id, _ := entry["id"].(string)
		ids = append(ids, id)
	}

	return ids
}


// key returns the signing key every event of the symbolic name uses.
func key(name string) ed25519.PrivateKey {
	seed := sha256.Sum256([]byte(name))
	return ed25519.NewKeyFromSeed(seed[:])
}


// build signs the arrival id once, after the arrivals its payload names.
func (b *builder) build(id string) Built {
	if done, ok := b.built[id]; ok {
		return done
	}

	a := b.arrivals[id]

	at := a.At
	if at == "" {
		at = defaultAt
	}

	raw, err := testevent.Sign(key(a.Author), testevent.Fields{
		Protocol:  network,
		Version:   1,
		EventType: a.Type,
		CreatedAt: at,
		Payload:   string(b.payload(a)),
	})
	if err != nil {
		b.t.Fatalf("sign %s: %v", id, err)
	}

	env, err := envelope.Decode(raw)
	if err != nil {
		b.t.Fatalf("decode %s: %v", id, err)
	}

	b.built[id] = Built{ID: id, Raw: raw, Env: env}
	return b.built[id]
}


// payload returns the payload of a with its symbolic names made real.
func (b *builder) payload(a Arrival) []byte {
	fields := map[string]json.RawMessage{}
	if len(a.Payload) > 0 {
		if err := json.Unmarshal(a.Payload, &fields); err != nil {
			b.t.Fatalf("payload of %s: %v", a.ID, err)
		}
	}

	for _, name := range eventFields {
		b.rename(fields, name, b.eventID)
	}

	for _, name := range authorFields {
		b.rename(fields, name, Author)
	}

	b.rename(fields, "entity", entityID)

	raw, err := json.Marshal(fields)
	if err != nil {
		b.t.Fatalf("payload of %s: %v", a.ID, err)
	}

	return raw
}


// rename replaces the symbolic name held by fields[field] with resolve(name).
func (b *builder) rename(fields map[string]json.RawMessage,
                         field string,
                         resolve func(name string) string) {
	var name string
	if json.Unmarshal(fields[field], &name) != nil {
		return
	}

	fields[field], _ = json.Marshal(resolve(name))
}


// eventID returns the eventId of the arrival name, or a made-up one.
func (b *builder) eventID(name string) string {
	if _, ok := b.arrivals[name]; !ok {
		return EventID(name)
	}

	return b.build(name).Env.EventID
}


// Load reads and checks protocol/fixtures/<name>.json from a library package.
func Load(t *testing.T, name string) File {
	t.Helper()

	return LoadFrom(t, "../../fixtures", name)
}


// LoadFrom reads and checks <dir>/<name>.json.
func LoadFrom(t *testing.T, dir string, name string) File {
	t.Helper()

	file, err := load(dir, name)
	if err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}

	return file
}


// load reads and checks <dir>/<name>.json.
func load(dir, name string) (File, error) {
	raw, err := os.ReadFile(filepath.Join(dir, name+".json"))
	if err != nil {
		return File{}, err
	}

	return parse(dir, raw)
}


// Run runs rule once per case, each as a subtest named after the case.
func Run(t *testing.T, file File, rule func(t *testing.T, c Case)) {
	t.Helper()

	if rule == nil {
		t.Skipf("%s: no rule implements %s yet", file.Topic, file.Rule)
	}

	for _, c := range file.Cases {
		t.Run(c.Name, func(t *testing.T) {
			if c.Since != "" {
				t.Skipf("runs since %s, not in this version", c.Since)
			}

			rule(t, c)
		})
	}
}


// parse validates raw against the fixture schema and decodes it.
func parse(dir string, raw []byte) (File, error) {
	if err := checkSchema(dir, raw); err != nil {
		return File{}, err
	}

	var file File
	if err := json.Unmarshal(raw, &file); err != nil {
		return File{}, err
	}

	if err := checkCases(file); err != nil {
		return File{}, err
	}

	return file, nil
}


// checkCases applies the checks beyond the schema to every case of file.
func checkCases(file File) error {
	for _, c := range file.Cases {
		known, err := checkArrival(c)
		if err != nil {
			return err
		}

		if err := checkReferences(c, known); err != nil {
			return err
		}

		if err := checkWeights(c); err != nil {
			return err
		}
	}

	return nil
}


// checkArrival refuses a repeated id and returns the ids and authors of c.
func checkArrival(c Case) (map[string]bool, error) {
	known := map[string]bool{}

	for _, a := range c.Arrival {
		if known[a.ID] {
			return nil, fmt.Errorf("case %q: id %q repeats", c.Name, a.ID)
		}

		known[a.ID] = true
	}

	// a rotation chain in expect names authors and their successors
	for _, a := range c.Arrival {
		known[a.Author] = true

		if next := successor(a); next != "" {
			known[next] = true
		}
	}

	return known, nil
}


// successor returns the successor key a rotation names, or "" for no one.
func successor(a Arrival) string {
	var payload struct {
		Successor string `json:"successor"`
	}

	// a payload that is not an object names no successor
	_ = json.Unmarshal(a.Payload, &payload)

	return payload.Successor
}


// checkReferences refuses a name in expect that arrival never introduces.
func checkReferences(c Case, known map[string]bool) error {
	for _, ref := range references(c.Expect) {
		if !known[ref] {
			return fmt.Errorf("case %q: expect names %q, absent from arrival",
			                  c.Name, ref)
		}
	}

	return nil
}


// references lists the names expect gives to events and authors.
func references(expect map[string]any) []string {
	var refs []string

	for _, key := range referenceKeys {
		refs = append(refs, names(expect[key])...)
	}

	for name := range asMap(expect["historical"]) {
		refs = append(refs, name)
	}

	for _, label := range asList(expect["labelsOnHead"]) {
		refs = append(refs, names(asMap(label)["id"])...)
	}

	return refs
}


// names returns the strings of v, which is one string or a list of them.
func names(v any) []string {
	if s, ok := v.(string); ok {
		return []string{s}
	}

	var out []string

	for _, item := range asList(v) {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}

	return out
}


// asList returns v as a list, or nil when it is not one.
func asList(v any) []any {
	list, _ := v.([]any)

	return list
}


// asMap returns v as an object, or nil when it is not one.
func asMap(v any) map[string]any {
	object, _ := v.(map[string]any)

	return object
}


// checkWeights refuses a proposal whose author has no weight declared.
func checkWeights(c Case) error {
	if len(c.Identities) == 0 {
		return nil
	}

	for _, a := range c.Arrival {
		_, declared := c.Identities[a.Author]
		if a.Type == "entity.updated" && !declared {
			return fmt.Errorf("case %q: %q proposes with no weight",
			                  c.Name, a.Author)
		}
	}

	return nil
}


// checkSchema validates raw against protocol/fixtures/fixture.schema.json.
func checkSchema(dir string, raw []byte) error {
	compiler := jsonschema.NewCompiler()

	schema, err := compiler.Compile(filepath.Join(dir, "fixture.schema.json"))
	if err != nil {
		return err
	}

	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return err
	}

	return schema.Validate(doc)
}
