// Package schema validates payloads against the embedded JSON Schemas.
package schema


import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)


//go:generate sh copy.sh


// --- CONSTANTS ---

// suffix ends the name of every schema file.
const suffix = ".schema.json"


// payloadsDir holds one schema per eventType inside the embedded tree.
const payloadsDir = "files/payloads"


// --- GLOBALS ---

// embedded is the copy of schemas/ made by go:generate.
//
//go:embed files
var embedded embed.FS


// --- CODE ---

// Set is every embedded schema, compiled.
type Set struct {
	// payloads maps an eventType to its compiled schema
	payloads map[string]*jsonschema.Schema
}


// Load compiles every embedded schema, each registered under its $id.
func Load() (*Set, error) {
	compiler := jsonschema.NewCompiler()
	ids := map[string]string{}

	err := fs.WalkDir(embedded, "files", func(name string,
	                                          entry fs.DirEntry,
	                                          err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		id, err := register(compiler, name)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}

		ids[name] = id
		return nil
	})

	if err != nil {
		return nil, err
	}

	return compile(compiler, ids)
}


// register adds the file at name to compiler and returns its $id.
func register(compiler *jsonschema.Compiler, name string) (string, error) {
	raw, err := embedded.ReadFile(name)
	if err != nil {
		return "", err
	}

	var header struct {
		ID string `json:"$id"`
	}

	if err := json.Unmarshal(raw, &header); err != nil {
		return "", err
	}

	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return "", err
	}

	return header.ID, compiler.AddResource(header.ID, doc)
}


// compile compiles every registered schema and indexes those of a payload.
func compile(compiler *jsonschema.Compiler,
             ids map[string]string) (*Set, error) {
	set := &Set{payloads: map[string]*jsonschema.Schema{}}

	for name, id := range ids {
		compiled, err := compiler.Compile(id)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}

		if path.Dir(name) == payloadsDir {
			eventType := strings.TrimSuffix(path.Base(name), suffix)
			set.payloads[eventType] = compiled
		}
	}

	return set, nil
}
