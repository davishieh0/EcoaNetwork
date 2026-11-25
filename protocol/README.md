# Protocol

The one thing every Ecoa implementation must agree on. The specification is
[`docs/protocol.md`](../docs/protocol.md); everything here is derived from it
and checks it. Where a file here and the spec disagree, the spec is right and
the file is the bug.

| Path | What it holds |
|---|---|
| [`schemas/`](schemas/) | JSON Schema (draft 2020-12) for the envelope and the 18 event types |
| [`vectors/`](vectors/) | Byte-level test vectors: canonicalization, rejections, ZIP-215 signatures |
| [`fixtures/`](fixtures/) | Behavioural cases for the rules that are easy to get subtly wrong |
| [`go/`](go/) | The protocol as a Go library, used by the node and the indexer |

## Vectors and fixtures

Both are written to be run by any implementation, in any language, before it
exists. They answer different questions:

| File | Checks |
|---|---|
| `vectors/canonicalization.json` | An envelope, its canonical (JCS) bytes, `eventId` and signature |
| `vectors/rejections.json` | Raw bytes a node must refuse, with the validation step and the reason |
| `vectors/zip215.json` | The Ed25519 edge cases, each with its verdict; some must be **accepted** |
| `fixtures/*.json` | Chain heads, authorship, identity succession, orphans, votes and labels, entity metadata |

Vectors are cryptographic and exact. Fixtures use short symbolic ids (`e1`,
`alice`) so a failing case can be read; see
[`fixtures/README.md`](fixtures/README.md).

## Versions

Events carry their own `version` and the network name (`ecoa-dev` for now)
inside the signed bytes. The Go library is versioned on its own, in
[`go/VERSION`](go/VERSION).
