// Package fixturetest runs the behavioural cases of fixtures/ against a rule.
package fixturetest


import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)


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


// Load reads and checks fixtures/<name>.json from the repository root.
func Load(t *testing.T, name string) File {
	t.Helper()

	file, err := load("../../../fixtures", name)
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


// parse validates raw against the fixture schema and decodes it.
func parse(dir string, raw []byte) (File, error) {
	if err := checkSchema(dir, raw); err != nil {
		return File{}, err
	}

	var file File
	if err := json.Unmarshal(raw, &file); err != nil {
		return File{}, err
	}

	return file, nil
}


// checkSchema validates raw against fixtures/fixture.schema.json.
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
