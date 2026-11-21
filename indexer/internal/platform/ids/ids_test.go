package ids


import (
	"strings"
	"testing"
)


// --- CODE ---

func TestIsEventID(t *testing.T) {
	cases := map[string]bool{
		"evt_" + strings.Repeat("a", 52): true,
		"evt_" + strings.Repeat("7", 52): true,
		"evt_" + strings.Repeat("a", 51): false,
		"evt_" + strings.Repeat("A", 52): false,
		"evt_" + strings.Repeat("1", 52): false,
		"abc":                            false,
	}

	for s, want := range cases {
		if got := IsEventID(s); got != want {
			t.Errorf("IsEventID(%q) = %v, want %v", s, got, want)
		}
	}
}
