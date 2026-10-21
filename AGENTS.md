# AGENTS.md

This file provides guidance to AI coding agents when working with code in this repository.

## What this repository is

A spec-first project. `docs/` holds ~2.600 lines of a closed design; the node that implements it is barely started. **The specification is the source of truth** — when code and docs disagree, the docs win unless the divergence is recorded in `docs/decisions.md`. Assertions about protocol rules belong in a doc reference, never in memory of this file.

Read `docs/protocol.md` before touching anything in `node/`. Most of what the node does is defined there, in *Field rules*, *Canonicalization, event ID, and signature*, and *Validation checklist*.

## Layout

| Path | What it is |
|---|---|
| `docs/` | The prose specification, in English. Style is deliberate — match it, don't rewrite it. |
| `protocol/` | Machine-consumable artifacts: `schemas/` (envelope + 18 payload schemas), `fixtures/` (behavioural cases). `vectors/` (ECO-36/37 test vectors) lands here. |
| `node/` | The Go module. Module path is `github.com/eduardokairalla1/EcoaNetwork` (upstream's), not the fork's. |
| `checklist.md` | The build plan for the node: 78 items in 9 sections, numbered `section.item` (`2.10`). Mark an item done only when it is committed, and record the short SHA next to it. |
| `docs/conceitos/`, `estudos/` | Study notes in Portuguese. The repo is also an Obsidian vault, so these use wikilinks with exact headings (`[[protocol#Field rules\|...]]`), not markdown links. |

## Commands

All Go commands run from `node/`:

```
make build          # go build -o bin/ecoa ./cmd/ecoa
make run            # build, then run
make test           # go test -v ./...
go vet ./...
go test ./internal/envelope -run '^TestCanonicalization$' -v    # a single test
```

`make build` is what refreshes `bin/ecoa`; `go build ./...` does not, so a stale binary will happily run old code.

There are no tests yet. The first ones should consume `protocol/fixtures/*.json` and, once they exist, `protocol/vectors/*.json` — the fixtures are the specification's own test cases and do not need to be invented.

## Code style

**One line per docstring, and it says what the thing is.** Not a paragraph, not
the reasoning that led to it, not `:param:`/`:return:` for names the signature
already states. Objectivity is the whole point: a docstring nobody reads is
noise, and a long one hides the line that mattered. The reasoning behind a
decision goes in [`docs/decisions.md`](docs/decisions.md), where it can be
argued with; a rule a reader would otherwise break goes in a one-line comment
next to the code it constrains.

**One domain per file.** A module holds one thing: one class with nothing
beside it, or the types of a single domain and no second responsibility. A
domain needing several classes is a package of one-class modules, the way
`backend/src/node/` and `backend/src/err/` are. Two responsibilities in a file
is two files.

**FastAPI validates with pydantic, not by hand.** Reach for `BaseModel`,
`TypeAdapter`, `model_validate_json`, `Field(max_length=...)` and the
validation exceptions, instead of a `try` around `json.loads` or a manual
`len()` against a limit. When the bytes must survive untouched — as a signed
envelope must — read them with `request.body()` and hand *those bytes* to
pydantic; never bind them to a model that re-encodes them, which is how a
signature dies on the way to a node.

## Guidelines

**Lines stop at 80 columns, counting a tab as 4.** `gofmt` has no opinion on
line length, so this one is enforced by hand — and because Go indents with
tabs, measuring bytes will tell you a line fits when it does not.

When a call overflows, break it at the argument list in one of two shapes. Keep
the first argument on the call line and align the rest under the opening
parenthesis, one per line:

```go
network := flag.String("network",
                       "ecoa-dev",
                       "network this node belongs to")
```

Or open the parenthesis alone, group the arguments across lines, and close on
its own line — this one takes a trailing comma:

```go
listen := flag.String(
	"listen", "/ip4/127.0.0.1/tcp/4001",
	"multiaddr to listen on",
)
```

The alignment in the first shape is spaces padded out with tabs, and **`gofmt`
does not accept it** — it rewrites those continuation lines to a single tab.
Every file using that shape is therefore reported by `gofmt -l` as unformatted,
by design, and a `gofmt`/CI check would fail on it. Never "fix" one by running
`gofmt -w`: that silently reverts the style this section is asking for.

Use the second shape whenever the first would still overflow. Aligning under
the parenthesis costs a wide indent, so a long argument can no longer fit —
which is why `bootstrap` in `cmd/ecoa/main.go` opens the parenthesis alone and
`publish`, right below it, does not.

When a single expression is what overflows, hoist it into a named local above
the statement rather than folding the line.

**Two blank lines between top-level declarations, one between blocks inside a
function.** The wide gap is what makes a function boundary visible when
scrolling; the single gap separates the steps within one.

`gofmt` collapses the double gap back to one, the same way it undoes the
argument alignment above — one more reason those files stay off `gofmt -w`.
The single gap inside a function it leaves alone.

## Complexidade (Go)

- Funções novas ou alteradas: complexidade ciclomática ≤ 10 e cognitiva ≤ 15.
- Antes de concluir uma tarefa, rode `golangci-lint run` (gocyclo + gocognit
  habilitados) nos pacotes modificados. Não finalize com violações novas.
- Para reduzir complexidade, prefira early return, extrair funções com
  responsabilidade própria e trocar cadeias de if/switch por map ou tabela quando
  fizer sentido.
- Não reduza a métrica artificialmente: nada de helpers sem significado só para
  dividir a contagem, e nunca agrupe, omita ou engula tratamento de erro.
- Exceções aceitas (CC até 15): switch plano de mapeamento ou CC vinda
  majoritariamente de `if err != nil`. Marque com `//nolint:gocyclo // motivo`.
- Não refatore código existente fora do escopo da tarefa só por complexidade;
  aponte as funções problemáticas no resumo final.

## Architecture

Writes flow `client → gateway → node → network`; reads never touch a node (`docs/architecture.md`). The node validates, stores, gossips and syncs. It does not answer queries, and it does not respond to whoever sent it an event.

The whole protocol reduces to four lines:

```
canonical     = JCS(envelope − eventId − signature)
signing_input = "ecoa-event-v1" || 0x00 || canonical
eventId       = "evt_" + base32_lower_nopad(SHA-256(signing_input))
signature     = Ed25519(signing_key, signing_input)
```

Events reach the node through **three doors** — Gossipsub, a sync response, and re-processing from the orphan buffer — and all three must funnel into one validation path. Two validation paths eventually disagree, and disagreement between implementations is the failure mode the entire design exists to prevent.

Validation is the 13 numbered steps in `docs/protocol.md` → *Validation checklist*. The order is normative, not advisory. Steps 1–4 and 6–8 are universal; 5 and 9 are type-specific and are **skipped** for an unrecognized `eventType`, which is what lets the network evolve without a flag day. Step 10 is local operator policy and binds nobody else.

An event's outcome is never a response to the sender: it is rejected silently, dropped as a duplicate, parked in the orphan buffer, or stored-indexed-relayed.

## Invariants that are easy to break

Each of these has already been decided; changing one is a decision-log edit, not a code change.

- **`deviceKey` is required-but-nullable** — model it as `*string`. Under JCS an absent key and a `null` key serialize differently and yield different `eventId`s (`decisions.md` §1.1).
- **The signing bytes exclude `eventId` and `signature`**, and both exclusions are load-bearing: the second is what makes Ed25519 malleability harmless here.
- **The `"ecoa-event-v1" || 0x00` prefix is domain separation.** It cannot be added later without invalidating every signature ever made.
- **Verification is ZIP-215 and therefore cofactored.** Use `github.com/hdevalence/ed25519consensus`. Go's `crypto/ed25519` is cofactorless and cannot be made to comply (`decisions.md` §13.1, §13.2). Small-order points and non-canonical point encodings are *accepted*; a non-canonical scalar `S` is rejected. `did:key` decoding stays strict — that is a separate question, not a contradiction.
- **Canonicalization is `github.com/gowebpki/jcs`.** It rejects duplicate keys for free but does not cap nesting depth and recurses, so the depth limit (32) is this project's own work.
- **No floats anywhere.** When stripping `eventId` and `signature` before canonicalizing, do not round-trip through `map[string]interface{}` — every number becomes a `float64` and large integers lose precision silently. Use `json.Decoder` with `UseNumber()`, or edit the raw JSON.
- **Duplicate detection (step 4) runs before canonicalization and signature verification**, because the cheap check must not be gated behind the expensive one. Until step 7 recomputes it, the received `eventId` is attacker-controlled, so step 4 is a lookup and never a trust decision.
- **Never rebroadcast before validation finishes.** Step 13 is last.
- **An unresolvable reference is buffered, not rejected** — the target may arrive later. An unknown `eventType`, by contrast, is not an orphan: it is stored and relayed as structurally sound but semantically opaque.
- **Identity succession: first observed wins**, and revocation takes effect at local observation, never at the declared `createdAt`. The author's clock belongs to the attacker.

## Git

Two remotes: `main` is upstream (`eduardokairalla1/EcoaNetwork`), `origin` is the fork (`davishieh0/EcoaNetwork`). Work on `davishieh0/<topic>` branches and land through PRs; do not commit to `main` directly.

Commit messages follow `~/code/CLAUDE.md`: a single English title line, no body, no `Co-Authored-By`, one atomic change per commit.

**Changing code that has not reached `main` is a fixup, never a new commit on
top.** `git commit --fixup <sha>` against the commit that introduced the code,
then `git rebase --autosquash` before the branch is opened as a PR. A branch
that reaches review should read as if the code had been written right the first
time: no commit that corrects an earlier one, no dead end followed by its own
reversal. Once a commit is on `main` the rule inverts — history there is shared,
so a correction is a new commit.

## Agent skills

### Issue tracker

Linear, reached through Linear's hosted MCP server. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical roles, of which only `ready-for-agent` exists — a bare workspace label,
kept clear of the `Type` and `Area` groups. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context, and this repo is spec-first: the prose in `docs/` outranks the code,
with `docs/decisions.md` as the decision log. See `docs/agents/domain.md`.
