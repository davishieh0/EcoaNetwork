package seed


import (
	"strings"
	"testing"
)


// --- CODE ---

func TestReadReceitaKeepsActiveEstablishmentsNamed(t *testing.T) {
	// Arrange: a trade name, a legal name only, and one closed (08)
	est := `"11111111";"0001";"91";"1";"PADARIA BOM DIA";"02";"";"";"";"";"19980301";"4721102";"";"";"";"";"";"PINHEIROS";"";"SP";"";"";"";"";"";"";"";"";"";""
"22222222";"0001";"39";"1";"";"02";"";"";"";"";"20100101";"5611201";"";"";"";"";"";"SAVASSI";"";"MG";"";"";"";"";"";"";"";"";"";""
"33333333";"0001";"87";"1";"FECHADA";"08";"";"";"";"";"20000101";"4711302";"";"";"";"";"";"";"";"RJ";"";"";"";"";"";"";"";"";"";""
`
	emp := "\"22222222\";\"RESTAURANTE S\xc3O JORGE LTDA\";\"\";\"\";\"\";\"\";\"\"\n"

	// Act
	got, err := ReadReceita(strings.NewReader(est), strings.NewReader(emp), 0)

	// Assert
	if err != nil {
		t.Fatalf("ReadReceita: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d companies, want the 2 active: %+v", len(got), got)
	}

	if got[0].CNPJ != "11111111000191" || got[0].Name != "Padaria Bom Dia" ||
		got[0].Since != "1998" || got[0].District != "Pinheiros" {
		t.Fatalf("first = %+v", got[0])
	}

	// the legal name, read as ISO-8859-1 (\xc3 is Ã there)
	if got[1].Name != "Restaurante São Jorge Ltda" {
		t.Fatalf("second name = %q, want the legal name decoded", got[1].Name)
	}
}


func TestCategoryOfFollowsTheCNAEDivision(t *testing.T) {
	cases := map[string]string{
		"5611201": "food-and-drink",
		"4711302": "retail",
		"6201501": "technology",
		"6422100": "financial-services",
		"8630503": "health",
		"0111301": "other",
	}

	for cnae, want := range cases {
		if got := CategoryOf(cnae); got != want {
			t.Errorf("CategoryOf(%s) = %s, want %s", cnae, got, want)
		}
	}
}
