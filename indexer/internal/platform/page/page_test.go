package page


import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)


// --- CODE ---

func TestCursorRoundTrips(t *testing.T) {
	// Arrange
	want := Cursor{At: time.Date(2026, 11, 21, 14, 2, 11, 500, time.UTC), ID: "evt_a"}

	// Act
	got, err := Decode(want.Encode())

	// Assert
	if err != nil || !got.At.Equal(want.At) || got.ID != want.ID {
		t.Fatalf("Decode = %+v, %v, want %+v", got, err, want)
	}
}


func TestDecodeRefusesWhatEncodeDidNotWrite(t *testing.T) {
	for _, bad := range []string{"", "!!!", "bm8tc2VwYXJhdG9y", "eHxldnRfYQ"} {
		if _, err := Decode(bad); !errors.Is(err, ErrBadCursor) {
			t.Errorf("Decode(%q) = %v, want ErrBadCursor", bad, err)
		}
	}
}


func TestFromRequestReadsLimitAndAfter(t *testing.T) {
	// Arrange
	after := Cursor{At: time.Now().UTC(), ID: "evt_a"}
	req := httptest.NewRequest(http.MethodGet,
		"/x?limit=5&after="+after.Encode(), nil)

	// Act
	p, ok := FromRequest(httptest.NewRecorder(), req)

	// Assert
	if !ok || p.Limit != 5 || p.After == nil || p.After.ID != "evt_a" {
		t.Fatalf("FromRequest = %+v, %v", p, ok)
	}
}


func TestFromRequestRefusesABadLimitOrCursor(t *testing.T) {
	for _, query := range []string{"limit=0", "limit=101", "limit=x", "after=!"} {
		// Arrange
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x?"+query, nil)

		// Act
		_, ok := FromRequest(rec, req)

		// Assert
		if ok || rec.Code != http.StatusBadRequest {
			t.Errorf("%s: ok = %v, status %d, want 400", query, ok, rec.Code)
		}
	}
}
