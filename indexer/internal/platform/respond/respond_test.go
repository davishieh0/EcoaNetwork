package respond


import (
	"net/http"
	"net/http/httptest"
	"testing"
)


// --- CODE ---

func TestErrorHasTheBackendsShape(t *testing.T) {
	// Arrange
	rec := httptest.NewRecorder()

	// Act
	Error(rec, http.StatusNotFound, "event_not_found", "No event with this id.")

	// Assert
	want := "{\"error\":\"event_not_found\",\"message\":\"No event with this id.\"}\n"

	if rec.Code != http.StatusNotFound ||
	   rec.Header().Get("Content-Type") != "application/json" ||
	   rec.Body.String() != want {
		t.Fatalf("Error = %d %q", rec.Code, rec.Body.String())
	}
}


func TestItemAlwaysHasDerivedAndObserved(t *testing.T) {
	// Arrange
	rec := httptest.NewRecorder()

	// Act
	JSON(rec, http.StatusOK, Item{
		Envelope: `{"a":1}`,
		Derived:  struct{}{},
		Observed: struct{}{},
	})

	// Assert
	want := "{\"envelope\":\"{\\\"a\\\":1}\",\"derived\":{},\"observed\":{}}\n"

	if rec.Body.String() != want {
		t.Fatalf("Item = %q, want %q", rec.Body.String(), want)
	}
}
