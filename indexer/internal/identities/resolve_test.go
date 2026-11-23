package identities


import (
	"maps"
	"testing"
)


// --- CODE ---

// rot is a rotation of key to successor, written at minute at.
func rot(id, key, successor string, at string) Rotation {
	return Rotation{EventID: id, Key: key, Successor: successor,
	                CreatedAt: "2026-01-01T10:" + at + ":00Z"}
}


func TestResolve(t *testing.T) {
	cases := map[string]struct {
		rotations []Rotation
		want      map[string]Assignment
	}{
		"no rotation": {
			nil, map[string]Assignment{},
		},
		"one rotation": {
			[]Rotation{rot("r1", "k1", "k2", "00")},
			map[string]Assignment{
				"k1": {"k1", 0, false},
				"k2": {"k1", 1, false},
			},
		},
		"a transitive chain": {
			[]Rotation{rot("r2", "k2", "k3", "01"), rot("r1", "k1", "k2", "00")},
			map[string]Assignment{
				"k1": {"k1", 0, false},
				"k2": {"k1", 1, false},
				"k3": {"k1", 2, false},
			},
		},
		"two successors contest the identity": {
			[]Rotation{rot("a", "k1", "k2", "00"), rot("b", "k1", "m", "01")},
			map[string]Assignment{
				"k1": {"k1", 0, true},
				"k2": {"k2", 0, false},
				"m":  {"m", 0, false},
			},
		},
		"a contest midway keeps what came before": {
			[]Rotation{rot("r1", "k1", "k2", "00"), rot("a", "k2", "k3", "01"),
			           rot("b", "k2", "m", "02"), rot("r4", "k3", "k4", "03")},
			map[string]Assignment{
				"k1": {"k1", 0, true},
				"k2": {"k1", 1, true},
				"k3": {"k3", 0, false},
				"k4": {"k3", 1, false},
				"m":  {"m", 0, false},
			},
		},
		"a cycle is dropped, the earlier rotation kept": {
			[]Rotation{rot("r1", "k1", "k2", "00"), rot("r2", "k2", "k1", "01")},
			map[string]Assignment{
				"k1": {"k1", 0, false},
				"k2": {"k1", 1, false},
			},
		},
		"a key named by two keys joins neither": {
			[]Rotation{rot("r1", "k1", "k2", "00"), rot("x", "m", "k2", "01")},
			map[string]Assignment{
				"k1": {"k1", 0, false},
				"k2": {"k2", 0, false},
				"m":  {"m", 0, false},
			},
		},
		"the same rotation from two nodes counts once": {
			[]Rotation{rot("r1", "k1", "k2", "00"), rot("r1", "k1", "k2", "00")},
			map[string]Assignment{
				"k1": {"k1", 0, false},
				"k2": {"k1", 1, false},
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			got := Resolve(c.rotations)

			// Assert
			if !maps.Equal(got, c.want) {
				t.Fatalf("Resolve = %v, want %v", got, c.want)
			}
		})
	}
}
