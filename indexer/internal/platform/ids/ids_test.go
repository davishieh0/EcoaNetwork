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


func TestIsEntityID(t *testing.T) {
	cases := map[string]bool{
		"cnpj:12345678000199":              true,
		"gtin:07891000100103":              true,
		"domain:padaria-joao.com.br":       true,
		"place:osm:n123":                   true,
		"entity:" + strings.Repeat("a", 52): true,
		"cnpj:123":                         false,
		"domain:localhost":                 false,
		"domain:Padaria.com":               false,
		"place:osm:n0":                     false,
		"twitter:someone":                  false,
		"":                                 false,
	}

	for s, want := range cases {
		if got := IsEntityID(s); got != want {
			t.Errorf("IsEntityID(%q) = %v, want %v", s, got, want)
		}
	}
}
