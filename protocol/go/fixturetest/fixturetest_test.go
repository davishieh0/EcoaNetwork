package fixturetest


import (
	"encoding/json"
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
	file, err := parse("../../fixtures", raw)

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
			_, err := parse("../../fixtures", raw)

			// Assert
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("parse = %v, want an error citing %q", err, c.want)
			}
		})
	}
}


func TestParseChecksWhatTheSchemaCannot(t *testing.T) {
	arrival := `"arrival": [{"id": "e1", "author": "alice", ` +
		`"type": "review.created"}]`

	cases := []struct {
		name string
		from string
		to   string
		want string
	}{
		{
			"repeated id",
			arrival,
			`"arrival": [{"id": "e1", "author": "a", "type": "t"},
			             {"id": "e1", "author": "b", "type": "t"}]`,
			"repeats",
		},
		{
			"expect names an id outside arrival",
			`"expect": {"accepted": ["e1"]}`,
			`"expect": {"head": "e9"}`,
			"e9",
		},
		{
			"expect names an id outside a list",
			`"expect": {"accepted": ["e1"]}`,
			`"expect": {"chain": ["e1", "e9"]}`,
			"e9",
		},
		{
			"expect names a label outside arrival",
			`"expect": {"accepted": ["e1"]}`,
			`"expect": {"labelsOnHead": [{"id": "l9"}]}`,
			"l9",
		},
		{
			"expect names a version outside arrival",
			`"expect": {"accepted": ["e1"]}`,
			`"expect": {"historical": {"e9": {}}}`,
			"e9",
		},
		{
			"proposal from an author with no weight",
			arrival,
			`"identities": {"bob": {"weight": 0.5}},
			 "arrival": [{"id": "e1", "author": "alice",
			              "type": "entity.updated"}]`,
			"alice",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			raw := []byte(strings.Replace(validFile, c.from, c.to, 1))

			// Act
			_, err := parse("../../fixtures", raw)

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
	_, err := parse("../../fixtures", []byte(raw))

	// Assert
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
}


func TestLoadChecksEveryFixtureFile(t *testing.T) {
	// Arrange
	pattern := filepath.Join("../../fixtures", "*.json")

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

	path := "../../fixtures/fixture.schema.json"

	schema, err := os.ReadFile(path)
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


func TestRunGivesEachCaseASubtestNamedAfterIt(t *testing.T) {
	// Arrange
	file := File{Cases: []Case{{Name: "first"}, {Name: "second"}}}
	var seen []string

	// Act
	Run(t, file, func(t *testing.T, c Case) {
		seen = append(seen, t.Name())
	})

	// Assert
	want := []string{
		"TestRunGivesEachCaseASubtestNamedAfterIt/first",
		"TestRunGivesEachCaseASubtestNamedAfterIt/second",
	}

	if strings.Join(seen, ",") != strings.Join(want, ",") {
		t.Fatalf("subtests = %v, want %v", seen, want)
	}
}


func TestRunSkipsACaseWithSince(t *testing.T) {
	// Arrange
	file := File{Cases: []Case{{Name: "device", Since: "v2"}}}
	called := false

	// Act
	Run(t, file, func(t *testing.T, c Case) { called = true })

	// Assert
	if called {
		t.Fatal("rule ran for a case marked since v2")
	}
}


func TestRunSkipsAFileWithoutARule(t *testing.T) {
	// Arrange
	file := File{Topic: "t", Rule: "r"}
	var skipped bool

	// Act
	t.Run("file", func(t *testing.T) {
		// Arrange
		t.Cleanup(func() { skipped = t.Skipped() })

		// Act
		Run(t, file, nil)
	})

	// Assert
	if !skipped {
		t.Fatal("a file without a rule was not skipped")
	}
}


func TestEventsSignsTheArrivalsNamingRealIdsAndKeys(t *testing.T) {
	// Arrange
	file := Load(t, "authorship")
	c := file.Cases[0]

	// Act
	events := c.Events(t)

	// Assert
	if len(events) != len(c.Arrival) {
		t.Fatalf("Events = %d, want %d", len(events), len(c.Arrival))
	}

	if events[0].Env.Author != Author("alice") {
		t.Fatalf("author = %s, want alice", events[0].Env.Author)
	}

	var payload struct{ Replaces string }
	if err := json.Unmarshal(events[1].Env.Payload, &payload); err != nil {
		t.Fatalf("payload: %v", err)
	}

	if payload.Replaces != events[0].Env.EventID {
		t.Fatalf("replaces = %s, want %s",
		         payload.Replaces,
		         events[0].Env.EventID)
	}
}


func TestEventsNamesATargetThatArrivesLater(t *testing.T) {
	// Arrange
	file := Load(t, "orphan-buffer")
	c := file.Cases[1]

	// Act
	events := c.Events(t)

	// Assert
	var payload struct{ Target string }
	if err := json.Unmarshal(events[0].Env.Payload, &payload); err != nil {
		t.Fatalf("payload: %v", err)
	}

	if payload.Target != events[1].Env.EventID {
		t.Fatalf("target = %s, want %s",
		         payload.Target,
		         events[1].Env.EventID)
	}
}


func TestEventsGivesAnAbsentTargetAValidId(t *testing.T) {
	// Arrange
	file := Load(t, "orphan-buffer")
	c := file.Cases[3]

	// Act
	events := c.Events(t)

	// Assert
	var payload struct{ Target string }
	if err := json.Unmarshal(events[0].Env.Payload, &payload); err != nil {
		t.Fatalf("payload: %v", err)
	}

	if payload.Target != EventID("never_existed") {
		t.Fatalf("target = %s, want %s",
		         payload.Target,
		         EventID("never_existed"))
	}
}


func TestListAndRejectedReadTheExpectOfACase(t *testing.T) {
	// Arrange
	c := Case{Expect: map[string]any{
		"accepted": []any{"e1", "e2"},
		"rejected": []any{map[string]any{"id": "e3", "reason": "r"}},
	}}

	// Act
	accepted := c.List("accepted")
	absent := c.List("buffered")
	rejected := c.Rejected()

	// Assert
	if strings.Join(accepted, ",") != "e1,e2" {
		t.Fatalf("List accepted = %v", accepted)
	}

	if absent != nil {
		t.Fatalf("List absent = %v, want nil", absent)
	}

	if strings.Join(rejected, ",") != "e3" {
		t.Fatalf("Rejected = %v", rejected)
	}
}
