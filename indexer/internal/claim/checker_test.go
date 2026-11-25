package claim


import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)


// --- CODE ---

// fixed answers every lookup with the same records, or err.
type fixed struct {
	answer Answer
	err    error
}


func (f fixed) TXT(context.Context, string) (Answer, error) {
	return f.answer, f.err
}


func TestUnquoteJoinsTheStringsOfATXTRecord(t *testing.T) {
	cases := map[string]string{
		`"ecoa-identity=did:key:z6Mk"`:          "ecoa-identity=did:key:z6Mk",
		`"ecoa-identity=did:key:" "z6Mkabc"`:    "ecoa-identity=did:key:z6Mkabc",
		`"say \"hi\""`:                          `say "hi"`,
		`bare`:                                  "bare",
	}

	for in, want := range cases {
		if got := unquote(in); got != want {
			t.Errorf("unquote(%s) = %q, want %q", in, got, want)
		}
	}
}


func TestVerifyFindsOnlyTheExactRecord(t *testing.T) {
	ctx := context.Background()
	did := "did:key:z6MkSubject"

	cases := []struct {
		name   string
		lookup Lookup
		want   bool
	}{
		{"the record", fixed{answer: Answer{Records: []string{
			"v=spf1 -all", "ecoa-identity=" + did}}}, true},
		{"another identity", fixed{answer: Answer{Records: []string{
			"ecoa-identity=did:key:z6MkOther"}}}, false},
		{"a prefix of it", fixed{answer: Answer{Records: []string{
			"ecoa-identity=" + did + "x"}}}, false},
		{"no record", fixed{}, false},
		{"a failed lookup", fixed{err: errors.New("timeout")}, false},
	}

	for _, c := range cases {
		got, _, detail := Verify(ctx, c.lookup, "example.test", did)
		if got != c.want {
			t.Errorf("%s: Verify = %v (%s), want %v", c.name, got, detail,
			         c.want)
		}
	}
}


func TestDoHReadsTheJSONAnswer(t *testing.T) {
	// Arrange: a resolver answering as the public ones do
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("name") != "_ecoa-verify.example.test" ||
				r.URL.Query().Get("type") != "TXT" ||
				r.Header.Get("Accept") != "application/dns-json" {
				http.Error(w, "bad question", http.StatusBadRequest)
				return
			}

			_, _ = w.Write([]byte(`{"Status":0,"AD":true,"Answer":[
				{"name":"_ecoa-verify.example.test.","type":16,
				 "data":"\"ecoa-identity=did:key:\" \"z6MkSubject\""},
				{"name":"x","type":5,"data":"cname.example."}]}`))
		}))
	defer srv.Close()

	// Act
	answer, err := DoH{URL: srv.URL}.TXT(context.Background(),
	                                    "_ecoa-verify.example.test")

	// Assert
	if err != nil {
		t.Fatalf("TXT: %v", err)
	}

	if !answer.DNSSEC || len(answer.Records) != 1 ||
		answer.Records[0] != "ecoa-identity=did:key:z6MkSubject" {
		t.Fatalf("answer = %+v", answer)
	}
}
