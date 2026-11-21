package api


import (
	"net/http"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/respond"
)


// --- CODE ---

// health answers that the API is up.
func health(w http.ResponseWriter, _ *http.Request) {
	respond.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
