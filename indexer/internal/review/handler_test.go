package review


import (
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
		"/entities/twitter:x/reviews": "invalid_entity_id",
		list("?limit=0"):              "invalid_limit",
		list("?limit=101"):            "invalid_limit",
		list("?limit=many"):           "invalid_limit",
		list("?after=not-a-cursor"):   "invalid_cursor",
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
