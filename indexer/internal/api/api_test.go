package api


import (
	"net/http"
	"net/http/httptest"
	"testing"
)


// --- CODE ---

func TestAnUnknownPathIsNotFound(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/nothing", nil)
	rec := httptest.NewRecorder()

	// Act
	Handler().ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /nothing = %d, want 404", rec.Code)
	}
}
