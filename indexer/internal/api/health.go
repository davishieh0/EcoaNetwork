package api


import (
	"net/http"
)


// --- CODE ---

// health answers that the API is up.
func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
