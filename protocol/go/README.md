# protocol/go

The Ecoa protocol as a Go library: decoding, canonicalizing, verifying and
projecting events. No network and no storage, only pure functions, so it can
pass every vector and fixture in [`protocol/`](..) with nothing running. The
[node](../../node) and the [indexer](../../indexer) both build on it.

    module github.com/eduardokairalla1/EcoaNetwork/protocol/go

## Packages

| Package | Does |
|---|---|
| `envelope` | Decodes a signed envelope, canonicalizes it (JCS), recomputes `eventId` and verifies the signature (ZIP-215) |
| `schema` | Validates a payload against the JSON Schemas, embedded in the binary |
| `event` | Turns an envelope into an event the protocol accepts: every check that needs no stored state |
| `identity` | The Ed25519 identity as `did:key`; key rotation and revocation by first observation |
| `references` | Resolves the events an event names and checks the author may act on them |
| `projection` | The deterministic rules: chain head, withdrawal, votes, labels, responses, entity id and metadata |
| `testevent` | Builds validly signed events, for tests |
| `fixturetest` | Runs the cases of `protocol/fixtures/` against a rule |

## Using it

The node and the indexer live in the same repository and point at this folder
with a `replace` in their `go.mod`:

    replace github.com/eduardokairalla1/EcoaNetwork/protocol/go => ../protocol/go

That is why their Docker images are built from the repository root.

## Schemas

`schema` embeds a copy of `protocol/schemas/` in `schema/files/`, since
`go:embed` cannot reach outside the module. After changing a schema, copy
them again:

    cd schema && ./copy.sh

A test fails when the copy and `protocol/schemas/` differ.

## Tests

    go test ./...

The tests run the vectors in `protocol/vectors/` and the fixtures in
`protocol/fixtures/`.
