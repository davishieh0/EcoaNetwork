package api


import (
	"net/http"
	"net/http/httptest"
	"testing"
)


// --- CODE ---

func TestHealthAnswersOK(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Act
	Handler(nil).ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusOK ||
	   rec.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("GET /health = %d %q", rec.Code, rec.Body.String())
	}
}
