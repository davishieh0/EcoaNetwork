package entity


import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)


// --- CODE ---

// fold is what search compares: a name lowercased, without accents, with its
// spaces collapsed, so "Padaria do João" is found by "joao" and by "pada".
func fold(s string) string {
	strip := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)),
	                         norm.NFC)

	plain, _, err := transform.String(strip, s)
	if err != nil {
		plain = s
	}

	return strings.Join(strings.Fields(strings.ToLower(plain)), " ")
}


// likePattern escapes what LIKE reads as wildcards in a folded search text.
func likePattern(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
