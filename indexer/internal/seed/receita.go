// Package seed generates the MVP's seeded dataset: entities from the Receita
// Federal's public CNPJ data, and a few hundred properly signed synthetic
// reviews across them, published to a node like anyone's events.
package seed


import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
)


// --- GLOBALS ---

// the fields of a row of the Receita's Estabelecimentos layout, counted
// from 0
const (
	estBasic     = 0
	estOrder     = 1
	estDigits    = 2
	estTradeName = 4
	estStatus    = 5
	estOpened    = 10
	estCNAE      = 11
	estDistrict  = 17
	estState     = 19
	estFields    = 30
)

// the fields of a row of the Empresas layout
const (
	empBasic     = 0
	empLegalName = 1
	empFields    = 7
)

// statusActive is the situação cadastral of an active establishment.
const statusActive = "02"


// --- CODE ---

// Company is one active establishment, as the seed names it.
type Company struct {
	// CNPJ is the 14 digits.
	CNPJ string

	// Name is the trade name, or the legal name when it has none.
	Name string

	// CNAE is the main activity's code, 7 digits.
	CNAE string

	// Since is the year it opened.
	Since string

	// District and State are where it is.
	District string
	State    string
}


// latin1 reads ISO-8859-1, the Receita's encoding: each byte is the code
// point of the same number.
type latin1 struct {
	r *bufio.Reader
}


// Read decodes what it reads into UTF-8.
func (l latin1) Read(p []byte) (int, error) {
	n := 0

	for n+2 <= len(p) {
		b, err := l.r.ReadByte()
		if err != nil {
			if n > 0 && errors.Is(err, io.EOF) {
				return n, nil
			}

			return n, err
		}

		n += copy(p[n:], string(rune(b)))
	}

	return n, nil
}


// reader returns a CSV reader of the Receita's layout over r.
func reader(r io.Reader) *csv.Reader {
	c := csv.NewReader(latin1{r: bufio.NewReader(r)})
	c.Comma = ';'
	c.LazyQuotes = true
	c.FieldsPerRecord = -1

	return c
}


// ReadReceita reads the active establishments of an Estabelecimentos file,
// named after their trade name or, through empresas when given, their legal
// name; at most limit of them, all when limit is 0.
func ReadReceita(estabelecimentos io.Reader,
                 empresas io.Reader,
                 limit int) ([]Company, error) {
	legal := map[string]string{}

	if empresas != nil {
		rows := reader(empresas)

		for {
			row, err := rows.Read()
			if errors.Is(err, io.EOF) {
				break
			}

			if err != nil {
				return nil, fmt.Errorf("empresas: %w", err)
			}

			if len(row) < empFields {
				continue
			}

			legal[row[empBasic]] = row[empLegalName]
		}
	}

	var companies []Company
	rows := reader(estabelecimentos)

	for limit == 0 || len(companies) < limit {
		row, err := rows.Read()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("estabelecimentos: %w", err)
		}

		if len(row) < estFields || row[estStatus] != statusActive {
			continue
		}

		cnpj := row[estBasic] + row[estOrder] + row[estDigits]
		if len(cnpj) != 14 {
			continue
		}

		name := strings.TrimSpace(row[estTradeName])
		if name == "" {
			name = strings.TrimSpace(legal[row[estBasic]])
		}

		if name == "" {
			continue
		}

		since := ""
		if len(row[estOpened]) >= 4 {
			since = row[estOpened][:4]
		}

		companies = append(companies, Company{
			CNPJ:     cnpj,
			Name:     titleCase(name),
			CNAE:     row[estCNAE],
			Since:    since,
			District: titleCase(row[estDistrict]),
			State:    row[estState],
		})
	}

	return companies, nil
}


// titleCase writes an upper-case name as people read it, keeping short
// connectives lower case: "PADARIA DO BAIRRO LTDA" reads "Padaria do Bairro
// Ltda".
func titleCase(s string) string {
	small := map[string]bool{"da": true, "de": true, "do": true, "das": true,
	                         "dos": true, "e": true}

	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		if i > 0 && small[w] {
			continue
		}

		runes := []rune(w)
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		words[i] = string(runes)
	}

	return strings.Join(words, " ")
}


// CategoryOf maps a CNAE to the protocol's fixed list of categories, by its
// division: the first two digits.
func CategoryOf(cnae string) string {
	if len(cnae) < 2 {
		return "other"
	}

	division := cnae[:2]

	switch {
	case division >= "10" && division <= "33":
		return "manufacturing"
	case division == "35" || division == "36" || division == "37":
		return "utilities"
	case division >= "45" && division <= "47":
		return "retail"
	case division >= "49" && division <= "53":
		return "transport"
	case division == "55":
		return "travel-and-lodging"
	case division == "56":
		return "food-and-drink"
	case division == "59" || (division >= "90" && division <= "93"):
		return "entertainment"
	case division == "61":
		return "telecom"
	case division == "62" || division == "63":
		return "technology"
	case division >= "64" && division <= "66":
		return "financial-services"
	case division == "68":
		return "real-estate"
	case division >= "69" && division <= "75":
		return "professional-services"
	case division == "84":
		return "government"
	case division == "85":
		return "education"
	case division >= "86" && division <= "88":
		return "health"
	case division == "94":
		return "nonprofit"
	}

	return "other"
}
