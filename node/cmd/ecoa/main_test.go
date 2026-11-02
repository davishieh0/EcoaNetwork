package main


import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)


// --- CODE ---

func TestRunRefusesArguments(t *testing.T) {
	// Arrange
	args := []string{"-network", "x"}
	path := filepath.Join(t.TempDir(), "bolt.db")
	t.Setenv("BOLT_DB_PATH", path)

	// Act
	err := run(args)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "unexpected arguments") {
		t.Fatalf("run(%q) = %v, want an unexpected arguments error", args, err)
	}

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("store at %s: %v, want it never opened", path, err)
	}
}



