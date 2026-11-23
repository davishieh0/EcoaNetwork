package review


import (
	"bytes"
	"net/http"
	"testing"
)


// --- CODE ---

func TestListShowsTheHeadOfEachChainAndWhetherItWasEdited(t *testing.T) {
	// Arrange: one review edited twice, one never edited
	pool := migrated(t)

	version(t, pool, "a1", "a1", "", bakery, 0)
	version(t, pool, "a2", "a1", "a1", bakery, 1)
	version(t, pool, "a3", "a1", "a2", bakery, 2)
	chain(t, pool, "a1", bakery, "a3", 0, false, false)
	review(t, pool, "b1", bakery, 1)

	// Act
	rec := call(t, pool, list(""))

	// Assert
	a := decode(t, rec)

	if rec.Code != http.StatusOK || !same(names(a.Items), []string{"b1", "a3"}) {
		t.Fatalf("GET = %d, items %v, want b1, a3", rec.Code, names(a.Items))
	}

	if *a.Items[0].Derived.Edited || !*a.Items[1].Derived.Edited {
		t.Fatalf("edited = %v, %v, want false, true",
		         *a.Items[0].Derived.Edited, *a.Items[1].Derived.Edited)
	}

	if a.Items[1].Derived.ChainRoot != evt("a1") || !a.Items[1].Derived.Head {
		t.Fatalf("derived = %+v, want chain a1, head", a.Items[1].Derived)
	}
}


func TestListLeavesOutWithdrawnTruncatedAndOtherEntities(t *testing.T) {
	// Arrange
	pool := migrated(t)

	review(t, pool, "shown", bakery, 0)
	review(t, pool, "elsewhere", market, 1)

	version(t, pool, "gone", "gone", "", bakery, 2)
	chain(t, pool, "gone", bakery, "gone", 2, true, false)

	version(t, pool, "long", "long", "", bakery, 3)
	chain(t, pool, "long", bakery, "", 3, false, true)

	// Act
	a := decode(t, call(t, pool, list("")))

	// Assert
	if !same(names(a.Items), []string{"shown"}) {
		t.Fatalf("items = %v, want only shown", names(a.Items))
	}
}


func TestListPagesThroughEveryReviewOnceEvenWhenOneArrivesMidway(t *testing.T) {
	// Arrange: five reviews, newest last
	pool := migrated(t)

	for day, name := range []string{"r1", "r2", "r3", "r4", "r5"} {
		review(t, pool, name, bakery, day)
	}

	// Act: two at a time, with a newer review published after the first page
	first := decode(t, call(t, pool, list("?limit=2")))
	review(t, pool, "r6", bakery, 10)
	second := decode(t, call(t, pool, list("?limit=2&after="+*first.Next)))
	third := decode(t, call(t, pool, list("?limit=2&after="+*second.Next)))

	// Assert
	got := append(append(names(first.Items), names(second.Items)...),
	              names(third.Items)...)

	if !same(got, []string{"r5", "r4", "r3", "r2", "r1"}) {
		t.Fatalf("pages = %v, want r5 down to r1, each once", got)
	}

	if third.Next != nil {
		t.Fatalf("next of the last page = %q, want null", *third.Next)
	}
}


func TestListOfAnEntityWithNoReviewsIsEmpty(t *testing.T) {
	// Arrange
	pool := migrated(t)

	// Act
	rec := call(t, pool, list(""))

	// Assert
	if rec.Code != http.StatusOK ||
	   rec.Body.String() != "{\"items\":[],\"next\":null}\n" {
		t.Fatalf("GET = %d %q", rec.Code, rec.Body.String())
	}
}


func TestListRefusesAMalformedRequest(t *testing.T) {
	cases := map[string]string{
		"/entities/twitter:x/reviews": "invalid_entity_id_error",
		list("?limit=0"):              "invalid_limit_error",
		list("?limit=101"):            "invalid_limit_error",
		list("?limit=many"):           "invalid_limit_error",
		list("?after=not-a-cursor"):   "invalid_cursor_error",
	}

	for path, code := range cases {
		t.Run(path, func(t *testing.T) {
			// Act: refused before the database is asked anything
			rec := call(t, nil, path)

			// Assert
			if rec.Code != http.StatusBadRequest || decode(t, rec).Error != code {
				t.Fatalf("GET = %d %q, want 400 %s",
				         rec.Code, rec.Body.String(), code)
			}
		})
	}
}


func TestGetReturnsAnOldVersionWithEveryVersionOfItsChain(t *testing.T) {
	// Arrange: a1 edited twice from the same version, a3 winning the fork
	pool := migrated(t)

	version(t, pool, "a1", "a1", "", bakery, 0)
	version(t, pool, "a2", "a1", "a1", bakery, 1)
	version(t, pool, "a3", "a1", "a1", bakery, 2)
	chain(t, pool, "a1", bakery, "a3", 0, false, false)

	// Act
	rec := call(t, pool, "/reviews/"+evt("a1"))

	// Assert
	a := decode(t, rec)

	if rec.Code != http.StatusOK ||
	   a.Review.Envelope != string(envelopeOf("a1")) ||
	   a.Review.Derived.Head {
		t.Fatalf("review = %d %+v, want a1, not the head", rec.Code, a.Review)
	}

	if !same(names(a.Versions), []string{"a1", "a2", "a3"}) {
		t.Fatalf("versions = %v, want a1, a2, a3", names(a.Versions))
	}

	heads := [3]bool{a.Versions[0].Derived.Head,
	                 a.Versions[1].Derived.Head,
	                 a.Versions[2].Derived.Head}

	if heads != [3]bool{false, false, true} {
		t.Fatalf("heads = %v, want only a3", heads)
	}
}


func TestGetKeepsEveryEnvelopeByteForByte(t *testing.T) {
	// Arrange
	pool := migrated(t)
	review(t, pool, "a1", bakery, 0)

	// Act
	rec := call(t, pool, "/reviews/"+evt("a1"))

	// Assert
	a := decode(t, rec)

	if !bytes.Equal([]byte(a.Review.Envelope), envelopeOf("a1")) ||
	   !bytes.Equal([]byte(a.Versions[0].Envelope), envelopeOf("a1")) {
		t.Fatalf("envelopes = %q, %q", a.Review.Envelope, a.Versions[0].Envelope)
	}
}


func TestGetAnswersGoneForAWithdrawnReview(t *testing.T) {
	// Arrange
	pool := migrated(t)

	version(t, pool, "gone", "gone", "", bakery, 0)
	chain(t, pool, "gone", bakery, "gone", 0, true, false)

	// Act
	rec := call(t, pool, "/reviews/"+evt("gone"))

	// Assert: nothing of what it said comes back
	if rec.Code != http.StatusGone || decode(t, rec).Error != "review_withdrawn_error" ||
	   bytes.Contains(rec.Body.Bytes(), []byte("envelope")) {
		t.Fatalf("GET = %d %q", rec.Code, rec.Body.String())
	}
}


func TestGetAnswersNotFoundAndBadRequest(t *testing.T) {
	// Arrange
	pool := migrated(t)

	// Act
	unknown := call(t, pool, "/reviews/"+evt("nobody"))
	malformed := call(t, pool, "/reviews/evt_short")

	// Assert
	if unknown.Code != http.StatusNotFound ||
	   decode(t, unknown).Error != "review_not_found_error" {
		t.Fatalf("unknown = %d %q", unknown.Code, unknown.Body.String())
	}

	if malformed.Code != http.StatusBadRequest ||
	   decode(t, malformed).Error != "invalid_event_id_error" {
		t.Fatalf("malformed = %d %q", malformed.Code, malformed.Body.String())
	}
}
