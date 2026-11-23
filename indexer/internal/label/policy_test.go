package label


import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)


// --- CONSTANTS ---

// mod is a well-formed did:key.
const mod = "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK"


// --- CODE ---

// write puts content in a file of the test and returns its path.
func write(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "moderators.json")

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	return path
}


func TestNoFileFollowsNobody(t *testing.T) {
	// Act
	p, err := LoadPolicy("")

	// Assert
	if err != nil || len(p.Moderators()) != 0 || p.Moderators() == nil {
		t.Fatalf("LoadPolicy = %v, %v, want an empty list", p.Moderators(), err)
	}
}


func TestLoadPolicyReadsTheModerators(t *testing.T) {
	// Arrange
	path := write(t, `{"moderators":["`+mod+`"]}`)

	// Act
	p, err := LoadPolicy(path)

	// Assert
	if err != nil || !slices.Equal(p.Moderators(), []string{mod}) {
		t.Fatalf("LoadPolicy = %v, %v", p.Moderators(), err)
	}
}


func TestLoadPolicyRefusesABadFile(t *testing.T) {
	cases := map[string]string{
		"not a did:key": `{"moderators":["someone"]}`,
		"not json":      `moderators: [x]`,
	}

	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			_, err := LoadPolicy(write(t, content))

			// Assert
			if err == nil {
				t.Fatal("a bad moderators file should stop the API")
			}
		})
	}

	if _, err := LoadPolicy(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("a missing moderators file should stop the API")
	}
}


func TestWhatEachLabelDoes(t *testing.T) {
	// Assert
	if got := Removing(); !slices.Equal(got, []string{"illegal", "personal-data"}) {
		t.Errorf("Removing = %v", got)
	}

	if got := Hiding(false); !slices.Equal(got, []string{"harassment", "illegal",
		"impersonation", "personal-data", "spam"}) {
		t.Errorf("Hiding(false) = %v", got)
	}

	if got := Hiding(true); !slices.Equal(got, []string{"illegal", "personal-data"}) {
		t.Errorf("Hiding(true) = %v, want only what is removed", got)
	}

	if EffectOf("off-topic") != Marked || EffectOf("spam") != Hidden {
		t.Errorf("off-topic and spam = %v, %v", EffectOf("off-topic"),
		         EffectOf("spam"))
	}
}
