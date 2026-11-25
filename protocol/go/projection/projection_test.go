package projection


import (
	"strings"
	"testing"
)


// --- CODE ---

func TestDeriveEntityIDIgnoresCaseSpaceAndComposition(t *testing.T) {
	// Arrange
	plain := Descriptor{
		Name: "padaria cafe", Kind: "business",
		Country: "BR", Locality: "são paulo",
	}

	variants := map[string]Descriptor{
		"capitals": {
			Name: "Padaria Cafe", Kind: "Business",
			Country: "br", Locality: "São Paulo",
		},
		"spaces": {
			Name: "  padaria \t cafe ", Kind: "business",
			Country: "BR", Locality: " são   paulo",
		},
		"decomposed accent": {
			Name: "padaria cafe", Kind: "business",
			Country: "BR", Locality: "são paulo",
		},
	}

	want, err := DeriveEntityID(plain)
	if err != nil {
		t.Fatalf("DeriveEntityID: %v", err)
	}

	for name, variant := range variants {
		t.Run(name, func(t *testing.T) {
			// Act
			got, err := DeriveEntityID(variant)

			// Assert
			if err != nil || got != want {
				t.Fatalf("DeriveEntityID = %q, %v, want %q", got, err, want)
			}
		})
	}
}


func TestDeriveEntityIDSeparatesDifferentDescriptors(t *testing.T) {
	// Arrange
	base := Descriptor{Name: "Joe's Bakery", Kind: "business", Country: "BR"}

	others := map[string]Descriptor{
		"name":     {Name: "Joe's Market", Kind: "business", Country: "BR"},
		"kind":     {Name: "Joe's Bakery", Kind: "place", Country: "BR"},
		"country":  {Name: "Joe's Bakery", Kind: "business", Country: "PT"},
		"locality": {
			Name: "Joe's Bakery", Kind: "business",
			Country: "BR", Locality: "Recife",
		},
	}

	want, err := DeriveEntityID(base)
	if err != nil {
		t.Fatalf("DeriveEntityID: %v", err)
	}

	for name, other := range others {
		t.Run(name, func(t *testing.T) {
			// Act
			got, err := DeriveEntityID(other)

			// Assert
			if err != nil || got == want {
				t.Fatalf("DeriveEntityID = %q, %v, another", got, err)
			}
		})
	}
}


func TestDeriveEntityIDHasTheShapeOfAnEntityIdentifier(t *testing.T) {
	// Arrange
	d := Descriptor{Name: "Joe's Bakery", Kind: "business", Country: "BR"}

	// Act
	got, err := DeriveEntityID(d)

	// Assert
	hash, found := strings.CutPrefix(got, "entity:")
	lower := hash == strings.ToLower(hash)

	if err != nil || !found || len(hash) != 52 || !lower {
		t.Fatalf("DeriveEntityID = %q, %v", got, err)
	}
}


func TestResolveFieldsKeepsTierOneWhenRepresentativeRenewsClaim(t *testing.T) {
	// Arrange: same author holds two active recognized domain claims (renewal overlap)
	entity := "domain:aurora.test"
	owner := "did:key:owner"
	identities := map[string]Identity{
		owner: {
			Weight: 1.0,
			Claims: []Claim{
				{Entity: entity, Recognized: true, SelfVerifiable: true},
				{Entity: entity, Recognized: true, SelfVerifiable: true},
			},
		},
	}
	proposals := []Event{
		{
			ID:        "evt-1",
			Author:    owner,
			CreatedAt: "2026-11-24T12:00:00Z",
			Payload:   []byte(`{"entity":"domain:aurora.test","name":"Padaria Aurora"}`),
		},
	}

	// Act
	fields := ResolveFields(proposals, identities)

	// Assert
	got, ok := fields["name"]
	if !ok || got.Tier != 1 || got.Value != "Padaria Aurora" {
		t.Fatalf("fields[name] = %+v (ok=%v), want Tier 1 Padaria Aurora", got, ok)
	}
}
