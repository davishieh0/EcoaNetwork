// Package respond writes the API's answers, in the shapes every route shares.
//
// It sits apart from package api because the subjects use it too, and api
// imports the subjects to register their routes.
package respond


import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
)


// --- CODE ---

// Item is one thing the API returns: its signed envelope, and beside it what
// the indexer worked out, deterministic apart from observed.
type Item struct {
	// Envelope is the stored bytes as a string, never a parsed object, so no
	// JSON parser on the way can change what the signature covers.
	Envelope string `json:"envelope"`

	// Derived is what any honest indexer must agree on.
	Derived any `json:"derived"`

	// Observed depends on when and where this indexer looked.
	Observed any `json:"observed"`
}


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


// Failed answers a read the database could not serve: 503 when it does not
// answer, so the client tries again, and 500 for anything else.
func Failed(w http.ResponseWriter, err error) {
	var unreachable *pgconn.ConnectError

	if errors.As(err, &unreachable) {
		slog.Warn("database unavailable", "err", err)
		Error(w, http.StatusServiceUnavailable, "database_unavailable",
		      "The database does not answer; try again later.")
		return
	}

	slog.Error("read", "err", err)
	Error(w, http.StatusInternalServerError, "internal_error",
	      "Something went wrong.")
}
