package node


import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)


// --- CODE ---

func TestTheLogNamesTheKindOfArrival(t *testing.T) {
	cases := map[ArrivalKind]string{
		Gossip:  "gossip",
		Gateway: "gateway",
		Sync:    "sync",
	}

	for kind, want := range cases {
		t.Run(want, func(t *testing.T) {
			// Arrange
			var out bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&out, nil))

			// Act
			logger.Info("stored", "arrival", kind)

			// Assert
			var line struct {
				Arrival any `json:"arrival"`
			}

			if err := json.Unmarshal(out.Bytes(), &line); err != nil {
				t.Fatalf("decode %q: %v", out.String(), err)
			}

			if line.Arrival != want {
				t.Fatalf("arrival = %v, want %q", line.Arrival, want)
			}
		})
	}
}
