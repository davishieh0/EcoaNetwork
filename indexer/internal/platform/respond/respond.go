// Package respond writes the API's answers, in the shapes every route shares.
//
// It sits apart from package api because the subjects use it too, and api
// imports the subjects to register their routes.
package respond


import (
	"encoding/json"
	"log/slog"
	"net/http"
)


// --- CODE ---

// errorBody is the shape of every failure, the same as the backend's.
type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}


// JSON sends body as JSON with the given status.
func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Warn("write response", "err", err)
	}
}


// Error sends a failure: a slug a program can branch on, and a message.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, errorBody{Error: code, Message: message})
}
