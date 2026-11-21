package review


import (
	"errors"
	"testing"
	"time"
)


// --- CODE ---

func TestCursorRoundTrips(t *testing.T) {
	// Arrange
	want := Cursor{
		CreatedAt: time.Date(2026, 11, 21, 14, 2, 11, 500, time.UTC),
		ChainRoot: evt("a1"),
	}

	// Act
	got, err := DecodeCursor(want.Encode())

	// Assert
	if err != nil || !got.CreatedAt.Equal(want.CreatedAt) ||
	   got.ChainRoot != want.ChainRoot {
		t.Fatalf("DecodeCursor = %+v, %v, want %+v", got, err, want)
	}
}


func TestDecodeCursorRefusesWhatEncodeDidNotWrite(t *testing.T) {
	for _, bad := range []string{"", "!!!", "bm8tc2VwYXJhdG9y", "eHxldnRfYQ"} {
		if _, err := DecodeCursor(bad); !errors.Is(err, ErrBadCursor) {
			t.Errorf("DecodeCursor(%q) = %v, want ErrBadCursor", bad, err)
		}
	}
}
