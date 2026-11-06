// Package server is the gRPC entry a gateway uses to submit events.
package server


import (
	"context"
	"log/slog"

	"github.com/eduardokairalla1/EcoaNetwork/internal/node"
	pb "github.com/eduardokairalla1/EcoaNetwork/internal/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)


// --- CONSTANTS ---

// MaxMessage is the 16 KB envelope ceiling plus its protobuf field header.
const MaxMessage = 16<<10 + 8


// --- CODE ---

// Server answers Publish by handing the envelope to the node.
type Server struct {
	pb.UnimplementedNodeServiceServer

	// node lifetime context the gossip goroutines run under
	ctx  context.Context
	node *node.Node
}


// New returns a gRPC server with the node service registered on it.
func New(ctx context.Context, n *node.Node) *grpc.Server {
	g := grpc.NewServer(grpc.MaxRecvMsgSize(MaxMessage))
	pb.RegisterNodeServiceServer(g, &Server{ctx: ctx, node: n})

	return g
}


// Publish answers with what the node did to the envelope, gossiping it if new.
func (s *Server) Publish(_ context.Context,
                         event *pb.Event) (*pb.PublishResponse, error) {
	raw := event.GetEnvelope()

	outcome, err := s.node.Receive(raw, node.Arrival{Kind: node.Gateway})

	switch outcome {
	case node.Stored:
		s.gossip(raw)
		return &pb.PublishResponse{Outcome: pb.Outcome_ACCEPTED}, nil
	case node.Duplicate:
		return &pb.PublishResponse{Outcome: pb.Outcome_ALREADY_EXISTS}, nil
	case node.Orphaned:
		return &pb.PublishResponse{Outcome: pb.Outcome_ORPHAN}, nil
	case node.Failed:
		return nil, status.Error(codes.Unavailable, err.Error())
	}

	return &pb.PublishResponse{
		Outcome: pb.Outcome_REJECTED,
		Reason:  err.Error(),
	}, nil
}


// gossip publishes raw in the background, after the answer.
func (s *Server) gossip(raw []byte) {
	go func() {
		if err := s.node.Publish(s.ctx, raw); err != nil {
			slog.Warn("gossip", "err", err)
		}
	}()
}
