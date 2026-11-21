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
