// Command demo runs the steps of the MVP's demonstration that need an event
// no client would make (docs/mvp.md, "The demonstration"):
//
//	demo tamper        step 5: a review with one byte of its payload flipped,
//	                   the old signature kept, refused and never relayed
//	demo foreign-edit  step 6: an edit of someone else's review, with a valid
//	                   signature, schema and target, refused by authorship
//
// Each publishes to -node and asks -peers whether the refused event reached
// them.
package main


import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/protobuf"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/testevent"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)


// --- CONSTANTS ---

// callTimeout bounds one call to a node.
const callTimeout = 5 * time.Second


// --- CODE ---

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "demo:", err)
		os.Exit(1)
	}
}


// run picks the step named in args and runs it.
func run(args []string) error {
	if len(args) == 0 {
		return errors.New("want a step: tamper or foreign-edit")
	}

	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	node := flags.String("node", "localhost:50051", "the node to publish to")
	peers := flags.String("peers",
	                      "localhost:50052,localhost:50053,localhost:50054,localhost:50055",
	                      "nodes to ask whether a refused event reached them")
	entity := flags.String("entity", "cnpj:33444555000166",
	                       "the entity the demo's review is about")
	network := flags.String("network", "ecoa-dev", "the protocol field")

	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	d := demo{network: *network, entity: *entity, peers: split(*peers)}

	conn, err := dial(*node)
	if err != nil {
		return err
	}
	defer conn.Close()
	d.node = protobuf.NewNodeServiceClient(conn)

	switch args[0] {
	case "tamper":
		return d.tamper()
	case "foreign-edit":
		return d.foreignEdit()
	}

	return fmt.Errorf("unknown step %q: tamper or foreign-edit", args[0])
}


// demo is what every step needs: where to publish and whom to ask.
type demo struct {
	node    protobuf.NodeServiceClient
	peers   []string
	network string
	entity  string
}


// tamper is step 5: flip one byte of a signed review's payload and keep the
// old signature.
func (d demo) tamper() error {
	key := freshKey()
	original, id, err := d.sign(key, "review.created",
	                            `{"entity":"`+d.entity+`","rating":4,`+
	                            `"body":"Bom atendimento."}`)
	if err != nil {
		return err
	}

	// "Bom" becomes "Bum": one byte, inside the signed payload
	tampered := bytes.Replace(original, []byte("Bom atendimento"),
	                          []byte("Bum atendimento"), 1)

	fmt.Println("step 5 · a tampered event")
	fmt.Println("  eventId:", id)
	fmt.Println("  one byte of the payload flipped: \"Bom\" → \"Bum\";",
	            "signature kept")

	if err := d.publish("tampered", tampered); err != nil {
		return err
	}

	if err := d.publish("the original, untouched", original); err != nil {
		return err
	}

	return d.reached(id, "the original")
}


// foreignEdit is step 6: one key writes a review, another edits it.
func (d demo) foreignEdit() error {
	author, intruder := freshKey(), freshKey()

	review, reviewID, err := d.sign(author, "review.created",
	                                `{"entity":"`+d.entity+`","rating":5,`+
	                                `"body":"Atendimento excelente."}`)
	if err != nil {
		return err
	}

	edit, editID, err := d.sign(intruder, "review.replaced",
	                            `{"replaces":"`+reviewID+`","rating":1,`+
	                            `"body":"Atendimento péssimo."}`)
	if err != nil {
		return err
	}

	fmt.Println("step 6 · an edit of someone else's review")
	fmt.Println("  review by", didOf(author))
	fmt.Println("  edit by  ", didOf(intruder),
	            "— valid signature, valid schema, the target exists")

	if err := d.publish("the review", review); err != nil {
		return err
	}

	// the review has to be stored first, so the target exists
	time.Sleep(500 * time.Millisecond)

	if err := d.publish("the foreign edit", edit); err != nil {
		return err
	}

	return d.absent(editID, "the foreign edit")
}


// sign signs an event of eventType by key, now.
func (d demo) sign(key ed25519.PrivateKey,
                   eventType string,
                   payload string) ([]byte, string, error) {
	raw, err := testevent.Sign(key, testevent.Fields{
		Protocol:  d.network,
		Version:   1,
		EventType: eventType,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Payload:   payload,
	})
	if err != nil {
		return nil, "", err
	}

	var head struct {
		EventID string `json:"eventId"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, "", err
	}

	return raw, head.EventID, nil
}


// publish hands raw to the node and prints what it did.
func (d demo) publish(what string, raw []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()

	answer, err := d.node.Publish(ctx, &protobuf.Event{Envelope: raw})
	if err != nil {
		return fmt.Errorf("publish %s: %w", what, err)
	}

	line := fmt.Sprintf("  %-26s %s", what+":", answer.GetOutcome())
	if reason := answer.GetReason(); reason != "" {
		line += " — " + reason
	}
	fmt.Println(line)

	return nil
}


// absent checks that no peer holds the event id: a refused event is never
// relayed.
func (d demo) absent(id, what string) error {
	time.Sleep(2 * time.Second)

	for _, peer := range d.peers {
		held, err := holds(peer, id)
		if err != nil {
			return err
		}

		state := "never reached it"
		if held {
			state = "HOLDS IT"
		}

		fmt.Printf("  %-26s %s %s\n", peer+":", what, state)
	}

	return nil
}


// reached prints whether each peer holds the event id: the valid original
// spreads, as every valid event does.
func (d demo) reached(id, what string) error {
	time.Sleep(2 * time.Second)

	for _, peer := range d.peers {
		held, err := holds(peer, id)
		if err != nil {
			return err
		}

		state := "does not hold it"
		if held {
			state = "holds it, over gossip"
		}

		fmt.Printf("  %-26s %s %s\n", peer+":", what, state)
	}

	fmt.Println("  the tampered copy carried the same eventId, and no node took it")

	return nil
}


// holds reports whether the node at addr has an event with eventId id in its
// log, reading its log from the start.
func holds(addr, id string) (bool, error) {
	conn, err := dial(addr)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()

	stream, err := protobuf.NewNodeServiceClient(conn).
		SubscribeEvents(ctx, &protobuf.Cursor{})
	if err != nil {
		return false, err
	}

	for {
		e, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.DeadlineExceeded {
				// the log ended while the stream waits for more: not there
				return false, nil
			}
			return false, fmt.Errorf("subscribe %s: %w", addr, err)
		}

		if e.GetEventId() == id {
			return true, nil
		}

		if e.GetSequence() >= e.GetHead() {
			return false, nil
		}
	}
}


// dial opens a connection to a node's gRPC API.
func dial(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(addr,
	                      grpc.WithTransportCredentials(insecure.NewCredentials()))
}


// freshKey is a new identity, so every run is a new author.
func freshKey() ed25519.PrivateKey {
	seed := make([]byte, ed25519.SeedSize)
	_, _ = rand.Read(seed)

	return ed25519.NewKeyFromSeed(seed)
}


// didOf is the did:key of key, shortened for a line of output.
func didOf(key ed25519.PrivateKey) string {
	public, _ := key.Public().(ed25519.PublicKey)
	did := identity.FromKey(public).String()

	return did[:20] + "…" + did[len(did)-6:]
}


// split reads a comma-separated list.
func split(list string) []string {
	var out []string
	for _, item := range strings.Split(list, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}

	return out
}

