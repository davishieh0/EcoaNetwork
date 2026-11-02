package envelope


import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
)


// --- CONSTANTS ---

// maxDepth is the nesting cap of the protocol.
const maxDepth = 32


// --- CODE ---

// frame is one open object or array.
type frame struct {
	// isObject tells an object from an array
	isObject bool

	// expectKey is true when the next string of an object is a key
	expectKey bool

	// keys holds the keys already seen in an object
	keys map[string]bool
}


// walker follows the tokens of a document without recursion.
type walker struct {
	// frames is the stack of containers still open
	frames []frame

	// started is true once the top-level value has begun
	started bool
}


// checkStructure rejects input that would leave a choice to the parser.
func checkStructure(raw []byte) error {
	if !utf8.Valid(raw) {
		return &MalformedJSONError{Detail: "invalid UTF-8"}
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var w walker

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return w.finish()
		}

		if err != nil {
			return &MalformedJSONError{Detail: err.Error()}
		}

		if err := w.visit(token); err != nil {
			return err
		}
	}
}


// finish rejects a document that ended with nothing or with something open.
func (w *walker) finish() error {
	if !w.started || len(w.frames) > 0 {
		return &MalformedJSONError{Detail: "unexpected end"}
	}

	return nil
}


// visit checks one token and updates the stack.
func (w *walker) visit(token json.Token) error {
	// a value after the top-level one has closed
	if w.started && len(w.frames) == 0 {
		return &MalformedJSONError{Detail: "trailing data"}
	}

	w.started = true

	switch t := token.(type) {
	case json.Delim:
		return w.delimiter(t)

	case string:
		return w.text(t)
	}

	w.valueDone()
	return nil
}


// delimiter opens or closes a container.
func (w *walker) delimiter(d json.Delim) error {
	if d == '}' || d == ']' {
		w.frames = w.frames[:len(w.frames)-1]
		w.valueDone()
		return nil
	}

	if len(w.frames) == maxDepth {
		return &TooDeepError{Limit: maxDepth}
	}

	opened := frame{}
	if d == '{' {
		opened = frame{
			isObject:  true,
			expectKey: true,
			keys:      map[string]bool{},
		}
	}

	w.frames = append(w.frames, opened)
	return nil
}


// text handles a string, which is either a key or a value.
func (w *walker) text(s string) error {
	if len(w.frames) == 0 || !w.frames[len(w.frames)-1].expectKey {
		w.valueDone()
		return nil
	}

	top := &w.frames[len(w.frames)-1]

	if top.keys[s] {
		return &DuplicateKeyError{Key: s}
	}

	top.keys[s] = true
	top.expectKey = false

	return nil
}


// valueDone records that the value of the enclosing object has ended.
func (w *walker) valueDone() {
	if len(w.frames) > 0 && w.frames[len(w.frames)-1].isObject {
		w.frames[len(w.frames)-1].expectKey = true
	}
}
