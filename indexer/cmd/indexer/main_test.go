package main


import (
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
			if err == nil {
				t.Fatalf("run(%q) = nil, want a usage error", args)
			}
		})
	}
}
