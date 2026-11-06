# Ecoa node

The node stores and relays Ecoa events. It is configured only by environment
variables.

## Run

Build from the repository root, since the module depends on `protocol/go`:

    docker build -f node/Dockerfile -t ecoa-node .
    docker run -d -v ecoa-data:/data -p 4001:4001 -p 50051:50051 \
        -e NETWORK=ecoa-dev \
        -e BOOTSTRAP_ADDRS=/ip4/203.0.113.7/tcp/4001/p2p/<peerId> \
        ecoa-node

The log prints the node's own multiaddrs on start (`listening`). Another node
can use one of them in `BOOTSTRAP_ADDRS`.

## Ports

| Port  | Use                                                 |
|-------|-----------------------------------------------------|
| 4001  | libp2p: peers connect here                          |
| 50051 | gRPC: the gateway submits events here               |

The gRPC port has no authentication. Publish it only to the gateway that needs
it, not to the internet.

## Volume

`/data` holds the event log (`bolt.db`) and the peer key (`peer.key`). Keep
the volume between runs: the key is the node's peer ID, and the log is the
node's copy of the network. Losing the key gives the node a new identity.

## Configuration

| Variable                | Default                  | Meaning                                       |
|-------------------------|--------------------------|-----------------------------------------------|
| `NETWORK`               | `ecoa-dev`               | `ecoa` or `ecoa-<name>`; names the topic      |
| `LISTEN_ADDR`           | `/ip4/127.0.0.1/tcp/4001` | libp2p listen multiaddr (image: `0.0.0.0`)   |
| `GRPC_ADDR`             | `127.0.0.1:50051`        | gRPC listen address (image: `0.0.0.0`)        |
| `BOOTSTRAP_ADDRS`       | embedded list            | Peers to dial, replacing the embedded list    |
| `BOOTSTRAP_EXTRA_ADDRS` | empty                    | Peers to dial in addition                     |
| `BOLT_DB_PATH`          | `data/bbolt/bolt.db`     | Event log (image: `/data/bolt.db`)            |
| `PEER_KEY_PATH`         | `data/peer.key`          | Peer private key (image: `/data/peer.key`)    |
| `CONN_LOW`              | `32`                     | Connection count the manager trims down to    |
| `CONN_HIGH`             | `96`                     | Connection count that starts the trimming     |
| `MAX_EVENT_SIZE`        | `16384`                  | Largest event accepted, in bytes              |

Address lists are comma-separated. The embedded bootstrap list is empty until
a public node exists, so a new node needs `BOOTSTRAP_ADDRS` to find the
network.

## Legal exposure

Running a node means storing and relaying content written by strangers, and
the log is append-only. The compliance point of the design is the indexer and
the gateway, not the node, but whoever hosts a node still carries hosting
exposure for content they did not choose. Read
[Legal exposure](../docs/mvp.md#legal-exposure) before exposing a node.
