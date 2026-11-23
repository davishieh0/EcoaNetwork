package review


import (
	"encoding/base64"
	"errors"
	"strings"
	"time"
)


// --- GLOBALS ---

// ErrBadCursor is returned for a cursor this indexer did not hand out.
var ErrBadCursor = errors.New("malformed cursor")


// --- CODE ---

// Cursor is where a page of a listing ended: the last chain it showed.
//
// Clients hold it as an opaque string; only this file reads what is in it.
type Cursor struct {
	// CreatedAt is the createdAt of the chain's original.
	CreatedAt time.Time

	// ChainRoot breaks ties between chains created at the same instant.
	ChainRoot string
}


// Encode returns c as the opaque string a client sends back as ?after=.
func (c Cursor) Encode() string {
	plain := c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + c.ChainRoot

	return base64.RawURLEncoding.EncodeToString([]byte(plain))
}


// DecodeCursor reads a string Encode returned, or ErrBadCursor.
func DecodeCursor(s string) (Cursor, error) {
	plain, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, ErrBadCursor
	}

	at, root, found := strings.Cut(string(plain), "|")
	if !found || root == "" {
		return Cursor{}, ErrBadCursor
	}

	createdAt, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return Cursor{}, ErrBadCursor
	}

	return Cursor{CreatedAt: createdAt, ChainRoot: root}, nil
}
