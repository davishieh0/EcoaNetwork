# protobuf

The gRPC contracts. Each service generates its own code from these files;
the generated code is committed, and these files are the source.

| File | Between | What it carries |
|---|---|---|
| `node_api.proto` | node ↔ gateways and indexers | `Publish`, a signed event in and its outcome out; `SubscribeEvents`, the node's log from a cursor and every new event after it |
| `sync.proto` | node ↔ node, over libp2p (`/ecoa/sync/1.0.0`) | `GetEventsSince(cursor)` to catch up, `GetEvents(ids)` to fill a known hole |

The node's gRPC port has no authentication; see
[`node/README.md`](../node/README.md#ports).

## Generating the code

| Service | Command | Files | Output |
|---|---|---|---|
| node | `make proto` | both | `node/internal/protobuf/` |
| indexer | `make proto` | `node_api.proto` | `indexer/internal/protobuf/` |
| backend | `scripts/proto` | `node_api.proto` | `backend/src/grpc/proto/` |

The Go services need `protoc`, plus `protoc-gen-go` v1.36.12 and
`protoc-gen-go-grpc` v1.6.2 from `go install`. The backend brings its own
`protoc` through `grpcio-tools`, pinned to the `grpcio` it runs.

After changing a `.proto`, regenerate in every service that uses it, in the
same change.
