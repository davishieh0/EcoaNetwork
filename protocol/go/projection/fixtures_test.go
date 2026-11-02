package projection


import (
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/fixturetest"
)


// --- CODE ---

func TestFixtures(t *testing.T) {
	names := []string{
		"chain-head",
		"vote-and-label-binding",
		"entity-metadata",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			// Arrange
			file := fixturetest.Load(t, name)

			// Act
			fixturetest.Run(t, file, nil)

			// Assert
			t.Fatal("a file without a rule did not skip")
		})
	}
}
