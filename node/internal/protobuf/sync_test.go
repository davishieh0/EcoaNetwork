package protobuf


import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/proto"
)


func TestGetEventsSinceResponseKeepsEventBytes(t *testing.T) {
	// Arrange
	raw := []byte(`{"eventType":"x","extra":[1,2,3]}`)
	sent := &GetEventsSinceResponse{
		Events:     [][]byte{raw},
		NextCursor: 7,
		HasMore:    true,
	}

	// Act
	wire, err := proto.Marshal(sent)
	if err != nil {
		t.Fatal(err)
	}

	var got GetEventsSinceResponse
	err = proto.Unmarshal(wire, &got)

	// Assert
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Events) != 1 || !bytes.Equal(got.Events[0], raw) {
		t.Errorf("event bytes changed: %q", got.Events)
	}

	if got.NextCursor != 7 || !got.HasMore {
		t.Errorf("cursor or has_more lost: %v", &got)
	}
}
