package main


import (
	"errors"
	"testing"
)


// --- CODE ---

func TestRunRefusesAMissingOrUnknownSubcommand(t *testing.T) {
	cases := map[string][]string{
		"none":    nil,
		"two":     {"serve", "project"},
		"unknown": {"index"},
	}

	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			err := run(args)

			// Assert
			if err == nil || errors.Is(err, errNotBuilt) {
				t.Fatalf("run(%q) = %v, want a usage error", args, err)
			}
		})
	}
}


func TestRunSaysWhichSubcommandsAreNotBuiltYet(t *testing.T) {
	for _, name := range []string{"rebuild"} {
		t.Run(name, func(t *testing.T) {
			// Act
			err := run([]string{name})

			// Assert
			if !errors.Is(err, errNotBuilt) {
				t.Fatalf("run(%s) = %v, want errNotBuilt", name, err)
			}
		})
	}
}
