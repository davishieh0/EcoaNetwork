package api


import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/database/dbtest"
)


// --- CODE ---

func TestAnUnknownPathIsNotFound(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/nothing", nil)
	rec := httptest.NewRecorder()

	// Act
	Handler(nil).ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /nothing = %d, want 404", rec.Code)
	}
}


func TestServeAnswersAndStopsWithItsContext(t *testing.T) {
	// Arrange
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	addr := lis.Addr().String()
	lis.Close()

	pool := dbtest.Pool(t, dbtest.URL(t))

	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- Serve(ctx, addr, pool)
	}()

	// Act
	body := getWhenUp(t, "http://"+addr+"/health")
	stop()

	// Assert
	if body != "{\"status\":\"ok\"}\n" {
		t.Fatalf("GET /health = %q", body)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve = %v, want a clean stop", err)
		}
	case <-time.After(shutdownTimeout):
		t.Fatal("Serve did not stop with its context")
	}
}


// getWhenUp retries url until the server answers, and returns the body.
func getWhenUp(t *testing.T, url string) string {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)

	for {
		resp, err := http.Get(url)
		if err == nil {
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			return string(body)
		}

		if time.Now().After(deadline) {
			t.Fatalf("GET %s: %v", url, err)
		}

		time.Sleep(20 * time.Millisecond)
	}
}
