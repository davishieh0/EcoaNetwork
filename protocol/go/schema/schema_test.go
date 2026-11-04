package schema


import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
)


// --- CONSTANTS ---

// did is a well-formed did:key.
const did = "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK"


// evt is a well-formed event id.
const evt = "evt_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"


// --- CODE ---

// loadSet returns the compiled schemas or stops the test.
func loadSet(t *testing.T) *Set {
	t.Helper()

	set, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	return set
}


// differs returns the first file that is not the same in both trees.
func differs(embeddedTree, source fs.FS) error {
	return fs.WalkDir(source, ".", func(name string,
	                                    entry fs.DirEntry,
	                                    err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		want, err := fs.ReadFile(source, name)
		if err != nil {
			return err
		}

		got, err := fs.ReadFile(embeddedTree, name)
		if err != nil {
			return err
		}

		if string(got) != string(want) {
			return fmt.Errorf("%s differs", name)
		}

		return nil
	})
}


// sourceSchemas is protocol/schemas/ without the files that are not a schema.
func sourceSchemas(t *testing.T) fstest.MapFS {
	t.Helper()

	tree := fstest.MapFS{}
	root := os.DirFS("../../schemas")

	err := fs.WalkDir(root, ".", func(name string,
	                                  entry fs.DirEntry,
	                                  err error) error {
		if err != nil || entry.IsDir() || name == "README.md" {
			return err
		}

		raw, err := fs.ReadFile(root, name)
		if err != nil {
			return err
		}

		tree[name] = &fstest.MapFile{Data: raw}
		return nil
	})

	if err != nil {
		t.Fatalf("read schemas: %v", err)
	}

	return tree
}


func TestEmbeddedCopyMatchesTheRepository(t *testing.T) {
	// Arrange
	files, err := fs.Sub(embedded, "files")
	if err != nil {
		t.Fatalf("fs.Sub: %v", err)
	}

	source := sourceSchemas(t)

	// Act
	err = differs(files, source)

	// Assert
	if err != nil {
		t.Fatalf("run go generate ./schema: %v", err)
	}

	if len(source) != 20 {
		t.Fatalf("protocol/schemas/ holds %d files, want 20", len(source))
	}
}


func TestOneDifferentByteIsReported(t *testing.T) {
	// Arrange
	source := fstest.MapFS{"a.json": {Data: []byte("{}")}}
	copied := fstest.MapFS{"a.json": {Data: []byte("{ ")}}

	// Act
	err := differs(copied, source)

	// Assert
	if err == nil {
		t.Fatal("a changed byte went unnoticed")
	}
}


func TestLoadIndexesTheSchemaOfEveryPayload(t *testing.T) {
	// Arrange
	files, err := fs.ReadDir(embedded, payloadsDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	// Act
	set, err := Load()

	// Assert
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	for _, file := range files {
		eventType := strings.TrimSuffix(file.Name(), suffix)
		if set.ValidatePayload(eventType, []byte(`[]`)) == nil {
			t.Errorf("%s took a payload that is not an object", eventType)
		}
	}

	if err := set.ValidatePayload("thing.happened", []byte(`[]`)); err != nil {
		t.Errorf("an unknown eventType was checked: %v", err)
	}
}


func TestValidatePayloadAccepts(t *testing.T) {
	set := loadSet(t)

	review := func(entity string) string {
		return fmt.Sprintf(`{"entity":%q,"rating":1,"body":"x"}`, entity)
	}

	claim := func(expiresAt string) string {
		return fmt.Sprintf(`{"subject":%q,"expiresAt":%q,"claimType":"human"}`,
		                   did,
		                   expiresAt)
	}

	cases := []struct {
		name      string
		eventType string
		payload   string
	}{
		{
			"valid review", "review.created",
			`{"entity":"cnpj:12345678000199","rating":5,"body":"ok"}`,
		},
		{
			"unknown field", "review.created",
			`{"entity":"gtin:00012345678905","rating":1,"body":"x","new":1}`,
		},
		{"unknown eventType", "future.thing", `{"rating":"anything"}`},
		{"empty revocation", "identity.key_revoked", `{}`},
		{
			"replacement", "review.replaced",
			`{"replaces":"` + evt + `","rating":3,"body":"x"}`,
		},
		{
			"derived entity with descriptor", "entity.updated",
			`{"entity":"entity:` + strings.Repeat("a", 52) + `",` +
				`"descriptor":{"name":"x","kind":"other","country":"BR"}}`,
		},
		{
			"public entity without descriptor", "entity.updated",
			`{"entity":"cnpj:12345678000199","name":"x"}`,
		},
		{
			"domain claim", "verification.claim_created",
			`{"subject":"` + did + `","expiresAt":"2027-01-01T00:00:00Z",` +
				`"claimType":"domain","entity":"domain:ex.com"}`,
		},
		{
			"human claim", "verification.claim_created",
			`{"subject":"` + did + `","expiresAt":"2027-01-01T00:00:00Z",` +
				`"claimType":"human"}`,
		},
		{
			"withdrawal with enum", "review.withdrawn",
			`{"target":"` + evt + `","reason":"mistake"}`,
		},
		{
			"whitespace inside a title", "review.created",
			`{"entity":"gtin:00012345678905","rating":1,"body":"x",` +
				`"title":"a\u2003b\u00a0c"}`,
		},
		{
			"newline inside a body", "review.created",
			`{"entity":"gtin:00012345678905","rating":1,"body":"a\nb"}`,
		},
		{"www inside a domain", "review.created", review("domain:ex.www.com")},
		{
			"february 30 passes the pattern", "verification.claim_created",
			claim("2027-02-30T00:00:00Z"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			payload := json.RawMessage(c.payload)

			// Act
			err := set.ValidatePayload(c.eventType, payload)

			// Assert
			if err != nil {
				t.Fatalf("ValidatePayload: %v", err)
			}
		})
	}
}


func TestValidatePayloadKeepsTheBytes(t *testing.T) {
	// Arrange
	set := loadSet(t)
	text := `{"entity":"cnpj:12345678000199", "rating":5,"body":"x", "new" : 1}`
	payload := json.RawMessage(text)

	// Act
	err := set.ValidatePayload("review.created", payload)

	// Assert
	if err != nil {
		t.Fatalf("ValidatePayload: %v", err)
	}

	if string(payload) != text {
		t.Fatalf("payload became %s", payload)
	}
}


func TestValidatePayloadRejects(t *testing.T) {
	set := loadSet(t)

	review := func(entity, rating string) string {
		return fmt.Sprintf(`{"entity":%q,"rating":%s,"body":"x"}`,
		                   entity,
		                   rating)
	}

	cnpj := "cnpj:12345678000199"
	claim := func(rest string) string {
		return fmt.Sprintf(
			`{"subject":%q,"expiresAt":"2027-01-01T00:00:00Z",%s}`,
			did, rest,
		)
	}

	cases := []struct {
		name      string
		eventType string
		payload   string
	}{
		{"rating below", "review.created", review(cnpj, "0")},
		{"rating above", "review.created", review(cnpj, "6")},
		{"rating fractional", "review.created", review(cnpj, "4.5")},
		{"rating as text", "review.created", review(cnpj, `"3"`)},
		{
			"cnpj punctuated", "review.created",
			review("cnpj:12.345.678/0001-99", "3"),
		},
		{"domain uppercase", "review.created", review("domain:EX.com", "3")},
		{
			"domain with scheme", "review.created",
			review("domain:https://ex.com", "3"),
		},
		{"gtin short", "review.created", review("gtin:123", "3")},
		{
			"replacement carrying entity", "review.replaced",
			fmt.Sprintf(
				`{"replaces":%q,"entity":%q,"rating":3,"body":"x"}`,
				evt, cnpj,
			),
		},
		{
			"derived entity without descriptor", "entity.updated",
			`{"entity":"entity:` + strings.Repeat("a", 52) + `"}`,
		},
		{
			"public entity with descriptor", "entity.updated",
			fmt.Sprintf(
				`{"entity":%q,"descriptor":{"name":"x","kind":"other",`+
					`"country":"BR"}}`,
				cnpj,
			),
		},
		{
			"domain claim without entity", "verification.claim_created",
			claim(`"claimType":"domain"`),
		},
		{
			"human claim with entity", "verification.claim_created",
			claim(`"claimType":"human","entity":"domain:ex.com"`),
		},
		{
			"withdrawal with free text", "review.withdrawn",
			fmt.Sprintf(`{"target":%q,"reason":"because"}`, evt),
		},
		{"domain with www", "review.created", review("domain:www.ex.com", "3")},
		{
			"title carrying a C1 control", "review.created",
			`{"entity":"gtin:00012345678905","rating":1,"body":"x",` +
				`"title":"a\u0085b"}`,
		},
		{
			"body carrying a C1 control", "review.created",
			`{"entity":"gtin:00012345678905","rating":1,"body":"a\u0080b"}`,
		},
		{
			"title starting with an em space", "review.created",
			`{"entity":"gtin:00012345678905","rating":1,"body":"x",` +
				`"title":"\u2003x"}`,
		},
		{
			"title ending with a no-break space", "review.created",
			`{"entity":"gtin:00012345678905","rating":1,"body":"x",` +
				`"title":"x\u00a0"}`,
		},
		{
			"body ending with an ideographic space", "review.created",
			`{"entity":"gtin:00012345678905","rating":1,"body":"x\u3000"}`,
		},
		{
			"body ending with a newline", "review.created",
			`{"entity":"gtin:00012345678905","rating":1,"body":"x\n"}`,
		},
		{
			"expiresAt out of range", "verification.claim_created",
			fmt.Sprintf(`{"subject":%q,"expiresAt":"2027-13-45T99:99:99Z",`+
				`"claimType":"human"}`, did),
		},
		{
			"expiresAt with a leap second", "verification.claim_created",
			fmt.Sprintf(`{"subject":%q,"expiresAt":"2027-01-01T00:00:60Z",`+
				`"claimType":"human"}`, did),
		},
		{
			"expiresAt in year zero", "verification.claim_created",
			fmt.Sprintf(`{"subject":%q,"expiresAt":"0000-01-01T00:00:00Z",`+
				`"claimType":"human"}`, did),
		},
		{"not json", "review.created", `{`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			payload := json.RawMessage(c.payload)

			// Act
			err := set.ValidatePayload(c.eventType, payload)

			// Assert
			if !badPayload(err) {
				t.Fatalf("ValidatePayload = %v, want badPayload", err)
			}
		})
	}
}
