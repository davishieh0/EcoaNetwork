package orphan


import (
	"slices"
	"testing"
	"time"
)


// --- GLOBALS ---

// clock is the time every test buffer reads.
var clock = time.Date(2026, 8, 5, 17, 0, 0, 0, time.UTC)


// --- CODE ---

// entry returns an entry of origin waiting for target.
func entry(origin string, target string) Entry {
	return Entry{
		Raw:     []byte(origin + ">" + target),
		Missing: []string{target},
		Origin:  origin,
	}
}


// raws returns the bytes of entries.
func raws(entries []Entry) []string {
	var out []string
	for _, e := range entries {
		out = append(out, string(e.Raw))
	}

	return out
}


func TestAddEvictsTheOldestWhenTheBufferIsFull(t *testing.T) {
	// Arrange
	b := New(3, 3, time.Hour, func() time.Time { return clock })
	for _, target := range []string{"t1", "t2", "t3"} {
		b.Add(entry("a", target))
	}

	// Act
	b.Add(entry("a", "t4"))

	// Assert
	if b.Release("t1") != nil {
		t.Fatal("the oldest entry was kept")
	}

	if b.Release("t2") == nil || b.Release("t3") == nil ||
	   b.Release("t4") == nil {
		t.Fatal("an entry newer than the oldest was evicted")
	}
}


func TestAddEvictsAnEntryOlderThanTheOrphanAge(t *testing.T) {
	// Arrange
	now := clock
	b := New(10, 10, time.Hour, func() time.Time { return now })
	b.Add(entry("a", "t1"))
	now = now.Add(time.Hour)

	// Act
	got := b.Release("t1")

	// Assert
	if got != nil {
		t.Fatalf("Release = %v after the max age, want nothing", raws(got))
	}
}


func TestAddEvictsTheOldestOfAPeerOverItsShare(t *testing.T) {
	// Arrange
	b := New(10, 2, time.Hour, func() time.Time { return clock })
	b.Add(entry("bob", "t0"))
	b.Add(entry("mallory", "t1"))
	b.Add(entry("mallory", "t2"))

	// Act
	b.Add(entry("mallory", "t3"))

	// Assert
	if b.Release("t1") != nil {
		t.Fatal("the oldest entry of the peer over its share was kept")
	}

	if b.Release("t0") == nil || b.Release("t2") == nil ||
	   b.Release("t3") == nil {
		t.Fatal("an entry within its share was evicted")
	}
}


func TestReleaseReturnsEveryEntryWaitingForTheTarget(t *testing.T) {
	// Arrange
	b := New(10, 10, time.Hour, func() time.Time { return clock })
	b.Add(entry("a", "t1"))
	b.Add(entry("b", "t2"))
	b.Add(entry("c", "t1"))

	// Act
	released := b.Release("t1")

	// Assert
	if want := []string{"a>t1", "c>t1"}; !slices.Equal(raws(released), want) {
		t.Fatalf("Release = %v, want %v", raws(released), want)
	}

	if b.Release("t1") != nil || b.Release("t2") == nil {
		t.Fatal("want only the entry of t2 left")
	}
}


func TestAddChargesAReheldEntryToItsOriginalPeer(t *testing.T) {
	// Arrange
	b := New(10, 1, time.Hour, func() time.Time { return clock })
	b.Add(entry("mallory", "t1"))
	released := b.Release("t1")

	// Act
	b.Add(Entry{
		Raw:     released[0].Raw,
		Missing: []string{"t2"},
		Origin:  released[0].Origin,
	})
	b.Add(entry("mallory", "t3"))

	// Assert
	if b.Release("t2") != nil || b.Release("t3") == nil {
		t.Fatal("the reheld entry was not charged to its peer")
	}
}


func TestHasRoomIsFalseOnlyForAnOriginAtItsShare(t *testing.T) {
	// Arrange
	b := New(10, 2, time.Hour, func() time.Time { return clock })
	b.Add(entry("a", "t1"))
	b.Add(entry("a", "t2"))
	b.Add(entry("b", "t3"))

	// Act
	full := b.HasRoom("a")
	roomy := b.HasRoom("b")
	empty := b.HasRoom("c")
	b.Release("t1")
	freed := b.HasRoom("a")

	// Assert
	if full || !roomy || !empty || !freed {
		t.Fatalf("HasRoom = %t, %t, %t, %t, want false, true, true, true",
		         full, roomy, empty, freed)
	}
}
