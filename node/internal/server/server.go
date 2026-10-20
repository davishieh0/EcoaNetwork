// Package server is the gRPC door a gateway uses to submit events.
package server


import (
	"context"
	"errors"
	"log/slog"

	"github.com/eduardokairalla1/EcoaNetwork/internal/node"
	pb "github.com/eduardokairalla1/EcoaNetwork/internal/protobuf"
	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
	"google.golang.org/grpc"
)


// --- GLOBALS ---

// MaxMessage is the 16 KB envelope ceiling plus its protobuf field header.
const MaxMessage = 16<<10 + 8


// Server answers Publish by handing the envelope to the node.
type Server struct {
	pb.UnimplementedNodeServiceServer

	// node lifetime context the gossip goroutines run under
	ctx  context.Context
	node *node.Node
}


// --- CODE ---

// New returns a gRPC server with the node service registered on it.
func New(ctx context.Context, n *node.Node) *grpc.Server {
	g := grpc.NewServer(grpc.MaxRecvMsgSize(MaxMessage))
	pb.RegisterNodeServiceServer(g, &Server{ctx: ctx, node: n})

	return g
}


// Publish stores the envelope, then gossips it if it was new.
func (s *Server) Publish(_ context.Context,
                         event *pb.Event) (*pb.PublishResponse, error) {
	raw := event.GetEnvelope()

	err := s.node.Handle(raw)
	if errors.Is(err, store.ErrDuplicate) {
		return &pb.PublishResponse{Outcome: pb.Outcome_ALREADY_EXISTS}, nil
	}

	if err != nil {
		slog.Info("publish", "outcome", "rejected", "bytes", len(raw))
		return &pb.PublishResponse{
			Outcome: pb.Outcome_REJECTED,
			Reason:  err.Error(),
		}, nil
	}

	// gossip the event in the background, after answering
	go func() {
		if err := s.node.Publish(s.ctx, raw); err != nil {
			slog.Warn("gossip", "err", err)
		}
	}()

	slog.Info("publish", "outcome", "accepted", "bytes", len(raw))
	return &pb.PublishResponse{Outcome: pb.Outcome_ACCEPTED}, nil
}
