package node


import (
	"github.com/eduardokairalla1/EcoaNetwork/internal/orphan"
	"github.com/libp2p/go-libp2p/core/peer"
)


// --- CONSTANTS ---

const (
	// Gossip is an event relayed on the network topic.
	Gossip ArrivalKind = iota + 1

	// Gateway is an event submitted over gRPC.
	Gateway

	// Sync is an event pulled from a peer by a sync call.
	Sync
)


// --- GLOBALS ---

// kindNames is the log name of each kind of arrival.
var kindNames = map[ArrivalKind]string{
	Gossip:  "gossip",
	Gateway: "gateway",
	Sync:    "sync",
}


// --- CODE ---

// ArrivalKind is the path an event entered the node through.
type ArrivalKind int


// Arrival is one event as this node observed it.
type Arrival struct {
	// Kind is the path the event entered through.
	Kind ArrivalKind

	// Peer is the peer that delivered it, empty for the gateway.
	Peer peer.ID
}


// String returns the log name of the kind.
func (k ArrivalKind) String() string {
	return kindNames[k]
}


// originOf returns the origin an orphan of arrival is charged to.
func originOf(arrival Arrival) string {
	if arrival.Kind == Gateway {
		return gatewayOrigin
	}

	return string(arrival.Peer)
}


// arrivalOf returns the arrival that an orphan was held as.
func arrivalOf(entry orphan.Entry) Arrival {
	if entry.Origin == gatewayOrigin {
		return Arrival{Kind: Gateway}
	}

	return Arrival{Kind: ArrivalKind(entry.Kind), Peer: peer.ID(entry.Origin)}
}
