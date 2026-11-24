package claim


import (
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
)


// --- CODE ---

// Policy is whose word counts for the delegated claims: the verifiers this
// indexer recognizes for each type. A domain claim needs nobody's word: the
// indexer looks the record up itself.
type Policy struct {
	human []string
}


// file is the shape of the verifiers file.
type file struct {
	Human []string `json:"human"`
}


// LoadPolicy reads the verifiers file at path; "" recognizes no verifier,
// so delegated claims are stored and count for nothing.
func LoadPolicy(path string) (Policy, error) {
	if path == "" {
		return Policy{}, nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("VERIFIERS_FILE: %w", err)
	}

	var f file
	if err := json.Unmarshal(raw, &f); err != nil {
		return Policy{}, fmt.Errorf("VERIFIERS_FILE %s: %w", path, err)
	}

	for _, did := range f.Human {
		if _, err := identity.Parse(did); err != nil {
			return Policy{}, fmt.Errorf("VERIFIERS_FILE %s: %q is not a "+
			                            "did:key", path, did)
		}
	}

	return Recognize(f.Human...), nil
}


// Recognize returns a policy that recognizes human claims issued by
// verifiers.
func Recognize(verifiers ...string) Policy {
	return Policy{human: slices.Clone(verifiers)}
}


// Human returns the human verifiers recognized, never nil.
func (p Policy) Human() []string {
	if p.human == nil {
		return []string{}
	}

	return slices.Clone(p.human)
}
