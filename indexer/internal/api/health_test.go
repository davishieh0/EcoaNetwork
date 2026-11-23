package api


import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)


// --- CODE ---

// fakeDB answers Ping with err.
type fakeDB struct {
	err error
}


func (f fakeDB) Ping(context.Context) error {
	return f.err
}


func TestHealthAnswersOKWhenTheDatabaseAnswers(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Act
	health(fakeDB{})(rec, req)

	// Assert
	if rec.Code != http.StatusOK ||
	   rec.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("GET /health = %d %q", rec.Code, rec.Body.String())
	}
}


func TestHealthAnswers503WhenTheDatabaseDoesNot(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Act
	health(fakeDB{err: errors.New("connection refused")})(rec, req)

	// Assert
	want := "{\"error\":\"database_unavailable\"," +
	        "\"message\":\"The database does not answer.\"}\n"

	if rec.Code != http.StatusServiceUnavailable || rec.Body.String() != want {
		t.Fatalf("GET /health = %d %q", rec.Code, rec.Body.String())
	}
}
