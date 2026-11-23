// Package page is how the API's lists page: by an opaque cursor that says
// "after this item", never by a page number, which shifts as items arrive.
package page


import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/respond"
)


// --- CONSTANTS ---

const (
	// defaultLimit is a page's size when ?limit= is absent.
	defaultLimit = 20

	// maxLimit is the largest ?limit= accepted.
	maxLimit = 100
)


// --- GLOBALS ---

// ErrBadCursor is returned for a cursor this indexer did not hand out.
var ErrBadCursor = errors.New("malformed cursor")


// --- CODE ---

// Cursor is where a page ended: the last item it showed, by the time and id
// the list is ordered by. Clients hold it as an opaque string.
type Cursor struct {
	// At is the time the list is ordered by.
	At time.Time

	// ID breaks ties between items at the same instant.
	ID string
}


// Page is where a list starts and how much of it to return.
type Page struct {
	// After is the cursor the previous page ended on, nil for the first.
	After *Cursor

	// Limit is the most items to return.
	Limit int
}


// Encode returns c as the opaque string a client sends back as ?after=.
func (c Cursor) Encode() string {
	plain := c.At.UTC().Format(time.RFC3339Nano) + "|" + c.ID

	return base64.RawURLEncoding.EncodeToString([]byte(plain))
}


// Decode reads a string Encode returned, or ErrBadCursor.
func Decode(s string) (Cursor, error) {
	plain, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, ErrBadCursor
	}

	at, id, found := strings.Cut(string(plain), "|")
	if !found || id == "" {
		return Cursor{}, ErrBadCursor
	}

	t, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return Cursor{}, ErrBadCursor
	}

	return Cursor{At: t, ID: id}, nil
}


// FromRequest reads ?after= and ?limit=, or answers 400 and reports false.
func FromRequest(w http.ResponseWriter, r *http.Request) (Page, bool) {
	p := Page{Limit: defaultLimit}

	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > maxLimit {
			respond.Error(w, http.StatusBadRequest, "invalid_limit_error",
			              "limit is a whole number from 1 to 100.")
			return Page{}, false
		}

		p.Limit = limit
	}

	if raw := r.URL.Query().Get("after"); raw != "" {
		cursor, err := Decode(raw)
		if err != nil {
			respond.Error(w, http.StatusBadRequest, "invalid_cursor_error",
			              "after is the next of a previous page.")
			return Page{}, false
		}

		p.After = &cursor
	}

	return p, true
}


// Next returns the encoded cursor of the page after one that ended on c, or
// nil when there is none.
func Next(c *Cursor) *string {
	if c == nil {
		return nil
	}

	encoded := c.Encode()

	return &encoded
}
