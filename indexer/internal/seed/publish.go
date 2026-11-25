package seed


import (
	"context"
	"fmt"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)


// --- GLOBALS ---

// how long one Publish may take
const publishTimeout = 5 * time.Second


// --- CODE ---

// Published counts what the node did with each envelope, by outcome, and
// keeps the reasons it refused any.
type Published struct {
	Outcomes map[string]int
	Refused  []string
}


// Publish hands each envelope to the node at addr, in order, as any client
// does.
func Publish(ctx context.Context,
             addr string,
             envelopes [][]byte) (Published, error) {
	conn, err := grpc.NewClient(addr,
	                            grpc.WithTransportCredentials(
	                                insecure.NewCredentials()))
	if err != nil {
		return Published{}, err
	}
	defer conn.Close()

	client := protobuf.NewNodeServiceClient(conn)
	out := Published{Outcomes: map[string]int{}}

	for i, envelope := range envelopes {
		call, cancel := context.WithTimeout(ctx, publishTimeout)
		answer, err := client.Publish(call,
		                              &protobuf.Event{Envelope: envelope})
		cancel()

		if err != nil {
			return out, fmt.Errorf("publish %d of %d: %w",
			                       i+1, len(envelopes), err)
		}

		out.Outcomes[answer.GetOutcome().String()]++

		if answer.GetOutcome() == protobuf.Outcome_REJECTED {
			out.Refused = append(out.Refused, answer.GetReason())
		}
	}

	return out, nil
}
