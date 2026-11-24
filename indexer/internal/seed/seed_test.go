package seed


import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/seed/sample"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	protocolevent "github.com/eduardokairalla1/EcoaNetwork/protocol/go/event"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/schema"
)


// --- CODE ---

// plan is a small plan over the sample.
func plan(t *testing.T, seedValue int64) Plan {
	t.Helper()

	companies, err := ReadReceita(sample.Estabelecimentos(),
	                              sample.Empresas(), 20)
	if err != nil {
		t.Fatalf("ReadReceita: %v", err)
	}

	return Plan{
		Companies: companies,
		Reviewers: 15,
		Reviews:   60,
		Until:     time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		Days:      30,
		Seed:      seedValue,
		Operator:  KeyOf(seedValue, "operator", 0),
		Network:   "ecoa-dev",
	}
}


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


func TestGenerateIsTheSameForTheSamePlan(t *testing.T) {
	// Act
	first, err := Generate(plan(t, 3))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	again, err := Generate(plan(t, 3))
	if err != nil {
		t.Fatalf("Generate again: %v", err)
	}

	other, err := Generate(plan(t, 4))
	if err != nil {
		t.Fatalf("Generate other: %v", err)
	}

	// Assert
	if len(first.Envelopes) != len(again.Envelopes) {
		t.Fatalf("%d events, then %d", len(first.Envelopes), len(again.Envelopes))
	}

	for i := range first.Envelopes {
		if !bytes.Equal(first.Envelopes[i], again.Envelopes[i]) {
			t.Fatalf("event %d differs between two runs", i)
		}
	}

	if bytes.Equal(first.Envelopes[len(first.Envelopes)-1],
	               other.Envelopes[len(other.Envelopes)-1]) {
		t.Fatalf("another seed gave the same events")
	}
}


func TestGenerateSignsEveryEventValidly(t *testing.T) {
	// Arrange
	schemas, err := schema.Load()
	if err != nil {
		t.Fatalf("schema.Load: %v", err)
	}

	// Act
	generated, err := Generate(plan(t, 5))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Assert: the whole check a node runs before storing
	for i, raw := range generated.Envelopes {
		env, err := envelope.Decode(raw)
		if err != nil {
			t.Fatalf("event %d: Decode: %v", i, err)
		}

		if _, err := protocolevent.Verify(env, schemas); err != nil {
			t.Fatalf("event %d (%s): Verify: %v", i, env.EventType, err)
		}
	}

	if generated.Counts["review.created"] != 60 ||
		generated.Counts["entity.updated"] != 20 ||
		generated.Counts["profile.updated"] != 15 {
		t.Fatalf("counts = %v", generated.Counts)
	}
}
