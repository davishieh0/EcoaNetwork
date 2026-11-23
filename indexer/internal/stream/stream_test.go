package stream


import (
	"testing"
	"time"
)


// --- CODE ---

func TestJitterStaysBetweenHalfAndAll(t *testing.T) {
	for range 1000 {
		got := jitter(time.Second)

		if got < 500*time.Millisecond || got > time.Second {
			t.Fatalf("jitter(1s) = %v, want between 500ms and 1s", got)
		}
	}
}
