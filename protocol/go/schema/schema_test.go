package schema


import (
	"fmt"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"
)


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


// sourceSchemas is schemas/ without the files that are not a schema.
func sourceSchemas(t *testing.T) fstest.MapFS {
	t.Helper()

	tree := fstest.MapFS{}
	root := os.DirFS("../../../schemas")

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
		t.Fatalf("schemas/ holds %d files, want 20", len(source))
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

	if len(set.payloads) != len(files) {
		t.Fatalf("indexed %d payload schemas, want %d",
		         len(set.payloads), len(files))
	}
}
