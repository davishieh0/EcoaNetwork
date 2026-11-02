package fixturetest


import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)


// --- CONSTANTS ---

// validFile is the smallest fixture the schema accepts.
const validFile = `{
	"topic": "t", "rule": "r",
	"cases": [{
		"name": "n", "why": "w",
		"arrival": [{"id": "e1", "author": "alice", "type": "review.created"}],
		"expect": {"accepted": ["e1"]}
	}]
}`


// --- CODE ---

func TestParseAcceptsAValidFile(t *testing.T) {
	// Arrange
	raw := []byte(validFile)

	// Act
	file, err := parse("../../../fixtures", raw)

	// Assert
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(file.Cases) != 1 || file.Cases[0].Name != "n" {
		t.Fatalf("parse = %+v", file)
	}
}


func TestParseRefusesABrokenFile(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
		want string
	}{
		{"unknown field", `"why": "w"`, `"why": "w", "foo": 1`, "foo"},
		{
			"since other than v2",
			`"why": "w"`, `"why": "w", "since": "banana"`,
			"since",
		},
		{
			"since of the current version",
			`"why": "w"`, `"why": "w", "since": "v1"`,
			"since",
		},
		{
			"file without a rule",
			`"rule": "r",`, ``,
			"rule",
		},
		{
			"case without a why",
			`"why": "w",`, ``,
			"why",
		},
		{
			"unknown field at the top",
			`"topic": "t",`, `"topic": "t", "foo": 1,`,
			"foo",
		},
		{
			"unknown field in an arrival",
			`"type": "review.created"`, `"type": "x", "foo": 1`,
			"foo",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			raw := []byte(strings.Replace(validFile, c.from, c.to, 1))

			// Act
			_, err := parse("../../../fixtures", raw)

			// Assert
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("parse = %v, want an error citing %q", err, c.want)
			}
		})
	}
}


func TestParseAcceptsSinceV2(t *testing.T) {
	// Arrange
	raw := strings.Replace(validFile,
	                       `"why": "w"`,
	                       `"why": "w", "since": "v2"`,
	                       1)

	// Act
	_, err := parse("../../../fixtures", []byte(raw))

	// Assert
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
}


func TestLoadChecksEveryFixtureFile(t *testing.T) {
	// Arrange
	pattern := filepath.Join("../../../fixtures", "*.json")

	paths, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), ".json")
		if name == "fixture.schema" {
			continue
		}

		t.Run(name, func(t *testing.T) {
			// Act
			file := Load(t, name)

			// Assert
			if len(file.Cases) == 0 {
				t.Fatal("Load returned no cases")
			}
		})
	}
}


func TestLoadReadsTheNamedFileAndChecksIt(t *testing.T) {
	// Arrange
	dir := t.TempDir()

	schema, err := os.ReadFile(
		filepath.Join("../../../fixtures", "fixture.schema.json"),
	)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	files := map[string]string{
		"fixture.schema.json": string(schema),
		"good.json":           validFile,
		"bad.json":            `{"topic": "t"}`,
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	// Act
	good, goodErr := load(dir, "good")
	_, badErr := load(dir, "bad")
	_, absentErr := load(dir, "absent")

	// Assert
	if goodErr != nil || good.Topic != "t" {
		t.Fatalf("load good = %+v, %v", good, goodErr)
	}

	if badErr == nil {
		t.Fatal("load accepted a file the schema refuses")
	}

	if !errors.Is(absentErr, os.ErrNotExist) {
		t.Fatalf("load absent = %v, want os.ErrNotExist", absentErr)
	}
}
