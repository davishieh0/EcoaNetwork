package event


import (
	"bytes"
	"context"
	"errors"
	"testing"
)


// --- CODE ---

func TestByIDReturnsTheEventAndWhenItWasReceived(t *testing.T) {
	// Arrange
	pool := migrated(t)
	insert(t, pool, id("b"), tricky, true)

	// Act
	got, err := NewRepository(pool).ByID(context.Background(), id("b"))

	// Assert
	if err != nil || got.ID != id("b") || !bytes.Equal(got.Envelope, tricky) {
		t.Fatalf("ByID = %+v, %v", got, err)
	}

	if got.ReceivedAt == nil || !got.ReceivedAt.Equal(receivedAt) {
		t.Fatalf("ReceivedAt = %v, want %v", got.ReceivedAt, receivedAt)
	}
}


func TestByIDLeavesReceivedAtEmptyWithoutAnObservation(t *testing.T) {
	// Arrange
	pool := migrated(t)
	insert(t, pool, id("b"), tricky, false)

	// Act
	got, err := NewRepository(pool).ByID(context.Background(), id("b"))

	// Assert
	if err != nil || got.ReceivedAt != nil {
		t.Fatalf("ByID = %+v, %v, want no ReceivedAt", got, err)
	}
}


func TestByIDReturnsErrNotFoundForAnUnknownID(t *testing.T) {
	// Arrange
	pool := migrated(t)

	// Act
	_, err := NewRepository(pool).ByID(context.Background(), id("b"))

	// Assert
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("ByID = %v, want ErrNotFound", err)
	}
}
