# Indexer: design

**Status: designed, not built.** Written in Go ([Stack](#stack)). The node
side, the stream the indexer reads from, is built
([What the node has to provide](#what-the-node-has-to-provide)). The order to
build it in, across every service, is in [roadmap.md](./roadmap.md).

## What it is

The indexer is the read side of Ecoa. It follows the event logs of one or more
nodes and turns them into tables that can be searched and browsed: entities, reviews, the
current version of each review, votes, responses, labels.

It is the only thing reads go through. The node is never queried for browsing
([docs/architecture.md](../../docs/architecture.md#node-communication-patterns)).

```
WRITE   Client --HTTP--> Gateway --gRPC--> Node --libp2p--> other nodes
                                            |
                                       event log (the truth)
                                            |
                                  gRPC stream (sequence order)
                                            v
READ    Client --HTTP--> Gateway --HTTP--> Indexer --> its own Postgres
```

Two things follow from the log being the truth:

- **The indexer keeps no events of its own.** The node's log is the event
  store. The indexer holds projections of it and the position it has read up
  to, nothing more. A second copy of the log would be a second truth that can
  drift.
- **Everything the indexer holds can be thrown away.** Dropping the database
  and reading the log again from the start must rebuild it. That is an
  acceptance criterion of the MVP
  ([docs/mvp.md](../../docs/mvp.md#acceptance-criteria)), not an
  implementation detail.

## Shape

A separate service, in its own directory, with its own database. Other
services never read its tables; they ask its HTTP API.

It has two parts that share the database and nothing else:

| Part | Does | Talks to |
|---|---|---|
| **Projector** | Reads each node's stream, applies each event to the tables, saves that node's cursor | The nodes, over gRPC |
| **API** | Answers queries; never writes | The gateway, over HTTP |

They can run as two processes, so the API scales without touching the
projector. This is the only part of CQRS worth taking: there are no commands
here, only events that already happened, so a command bus or aggregates would
be weight with nothing to carry.

## Input: the stream from the node

### Contract

The node already serves `NodeService` to the gateway
([protobuf/node_api.proto](../../protobuf/node_api.proto)). The indexer adds
one call to it:

```proto
service NodeService {
    rpc Publish(Event) returns (PublishResponse);

    // Every event this node accepted after `after`, in sequence order, then
    // each new one as it is accepted. The stream does not end on its own.
    rpc SubscribeEvents(Cursor) returns (stream StoredEvent);
}

message Cursor {
    uint64 after  = 1;  // 0 for the whole log
    string log_id = 2;  // the log `after` counts in; empty the first time
}

message StoredEvent {
    bytes  envelope    = 1;  // byte for byte as it was signed and stored
    uint64 sequence    = 2;  // this node's position for it: the cursor
    int64  received_at = 3;  // unix ns; when this node accepted it
    string log_id      = 4;  // which log `sequence` counts in; see below
    string event_id    = 5;  // the envelope's, already read by the node
    string event_type  = 6;  // the envelope's, already read by the node
    uint64 head        = 7;  // the node's last sequence when this was sent
}
```

The contract is in [protobuf/node_api.proto](../../protobuf/node_api.proto).
`head` is what lets `/health` say how far behind each node the indexer is:
`head - sequence`, without asking the node anything else.

**When in doubt, a field goes in.** A field the indexer turns out not to need
costs a few bytes per event; a field it needs and does not have costs a
change to both sides and, for anything observed, a rebuild that cannot bring
it back. `event_id` and `event_type` are in the envelope already, but sending
them lets the indexer skip a duplicate or an unknown type without parsing.

It is the same question node-to-node sync asks (`GetEventsSince(cursor)`,
[docs/protocol.md](../../docs/protocol.md#node-to-node-sync)), so the node
can answer both with one implementation.

### Why it is shaped like this

**The envelope travels as raw bytes.** The signature covers the JCS bytes of
the envelope, and fields the node does not understand must survive
byte-exactly ([docs/protocol.md](../../docs/protocol.md#field-rules)). If the
node split the event into protobuf fields and rebuilt it, nobody could verify
the signature again. `sequence` and `received_at` sit outside the envelope
because they are this node's observations, not part of what was signed.

**The cursor is the node's local sequence, never `createdAt`.** `createdAt` is
chosen by the author, so an event stamped in the past and published today
would fall behind a cursor that had already passed it, and never be read
([docs/decisions.md](../../docs/decisions.md), 2.4).

**No acknowledgements.** The indexer saves its cursor in its own database, in
the same transaction as the tables it changed. After a crash it reconnects
with the last cursor it saved and carries on. The node keeps no state about
who reads from it, which is what lets anyone write another indexer against the
same `.proto`. Backpressure comes from gRPC: a stream rides on HTTP/2 flow
control, so when the indexer reads slowly, the node's send waits.

**`log_id` names the log a sequence counts in.** A node creates it, at
random, when it creates its store. A node that loses its data starts counting
from 1 again, and a cursor saved as "after 500" would then skip the first 500
events of the new log.

**The node compares it, on connecting.** The indexer sends the `log_id` its
cursor counts in with the cursor. A node whose log is another one ignores
`after` and sends its log from the start; the indexer sees the new `log_id`
on the events and keeps its cursor in that log from then on. The comparison
cannot wait for the events: a new log shorter than the old cursor would send
nothing after "500" until it grew past it, and the indexer would never learn
the log had changed, then skip its first 500 events.

### Reading from more than one node

The indexer follows a list of nodes, all at once, so one node going down
stops nothing. One node is the same design with a list of one.

- **One cursor per node.** Sequences are local
  ([docs/architecture.md](../../docs/architecture.md#node-communication-patterns)):
  the same event can be 501 on one node and 812 on another. The indexer keeps
  `(node, log_id, sequence)` for each, saved in the same transaction as the
  event it came with.
- **The same event arrives once per node.** Applying is idempotent by
  `eventId`, so every copy after the first changes nothing but that node's
  cursor.
- **Each stream is applied in its own order.** Any event on a stream has its
  targets earlier on that same stream, so applying each stream in order never
  meets a vote before its review, whichever node delivers what first. One
  goroutine per stream reads; a single applier writes, so two copies of one
  event never race.
- **`received_at` is the earliest any of its nodes reports.** It is the
  indexer's observation of when the network had the event, and it never moves
  later.
- **Adding a node** means reading it from 0. Everything it sends that the
  indexer already holds is a duplicate, which is cheap.

### What the node has to provide

All of it is in the node ([docs/decisions.md](../../docs/decisions.md)
§15.2 and §15.8):

| Needed | State |
|---|---|
| A monotonic local sequence and a receive time for every stored event (validation step 11) | **Done.** `AddEvent` writes the `events`, `log` and `meta` buckets in one bbolt transaction; the receive time is kept in nanoseconds |
| Reading the log in sequence order from a given position | **Done.** `Store.EventsSince(cursor, limit)`, which node-to-node sync (`GetEventsSince`) already serves from |
| Every arrival stored by the same path | **Done.** Gossipsub, the gateway, sync and released orphans all end in one function (§15.7), so the stream sees every stored event whichever door it came through |
| A `log_id`, created at random with the store and kept with it | **Done.** `Store.LogID()`, kept in the `info` bucket |
| Telling open streams when a new event is stored (step 12, "notify the indexer") | **Done.** `Store.Changed()`: a channel closed after each stored event. It carries no event; whoever wakes reads the log from its own cursor |
| `SubscribeEvents`, `Cursor` and `StoredEvent` in `node_api.proto` | **Done**, with the Go and Python code regenerated (`make proto` in `node/`, `scripts/proto` in `backend/`) |
| Serving `SubscribeEvents` | **Done** (`node/internal/server/subscribe.go`, [docs/decisions.md](../../docs/decisions.md) §15.8) |
| A small, configurable cap on open streams | **Done.** `SUBSCRIBE_MAX_STREAMS`, 16 by default. Each stream may read the whole log from disk; one over the cap is refused with `RESOURCE_EXHAUSTED`, and the indexer retries later |

`receivedAt` is stored per event, but the sync contract between nodes does not
carry it (`GetEventsSinceResponse` sends envelopes only), on purpose: a peer's
receive time is a hint forgeable in the direction an attacker wants
([docs/decisions.md](../../docs/decisions.md) §2.1). The indexer's stream is
different: the indexer trusts the nodes it chose to follow, so it takes
their `received_at`.

### What the indexer may assume

**Every event on the stream was validated by the node:** size, envelope,
schema, digest, signature, authorship
([docs/protocol.md](../../docs/protocol.md#validation-checklist-every-node-runs-this-on-receipt)).
The indexer does not repeat that work.

It reads each envelope with `envelope.Decode`, never `envelope.Open`.
`Open` applies the receiving node's rules, among them the five-minute limit
on a `createdAt` in the future, measured against the clock at the moment of
reading. That is right for a node at receipt and wrong for a projector:
nothing in a rebuild may depend on when it runs.

**Except where nodes may disagree.** Reading several nodes, the indexer holds
the union of what each accepted, and in two places honest nodes can accept
different things:

- **Conflicting key rotations.** A node keeps the first rotation it sees and
  rejects a later one, so two nodes can each hold a different successor. The
  indexer, seeing both, applies the protocol's rule for a node that receives
  both at once: the identity is contested, and neither successor is honored
  ([docs/protocol.md](../../docs/protocol.md#who-may-sign-what),
  `protocol/fixtures/identity-succession.json`).

  `identity.Tracker` in `protocol/go` models one node: it contests a key
  only when both rotations come in the same batch (`ObserveBackfill`), and
  refuses the second one once a successor is held (`Observe`). An indexer
  reading several nodes meets the second rotation later, on another stream,
  so it cannot use the tracker as it is. It keeps every rotation in a table
  and calls a key contested when it has two different successors. If this
  rule is wanted in the library too, it is a small addition to
  `protocol/go`.

  v1 has one key per identity and rejects `deviceKey`
  ([docs/decisions.md](../../docs/decisions.md) §3.5), so rotation and
  revocation are the only identity events the indexer follows.
- **The revocation cut.** An event signed just after its key was revoked is
  accepted by a node that had not seen the revocation yet, and rejected by one
  that had. The indexer holds it if any node does. Which events fall on which
  side of the cut is already an observation, not a deterministic fact
  ([docs/architecture.md](../../docs/architecture.md#what-must-be-identical-across-indexers-and-what-must-not-be)).

**A target comes before what points at it.** A node stores an event only once
its references resolve; until then it waits in the orphan buffer
([docs/protocol.md](../../docs/protocol.md#unresolved-references)). The
sequence is assigned on storing, so in sequence order a review always comes
before its votes, edits and responses.

**An `eventType` it does not know** is stored and relayed by nodes. The
indexer skips it, advances the cursor, and counts it. When support for that
type is added later, a rebuild picks up what was skipped.

## Projections

### Two kinds of data, kept in separate tables

Some of what the indexer holds is computed from the events alone. Some depends
on when and where the indexer looked. The architecture lists both
([docs/architecture.md](../../docs/architecture.md#what-must-be-identical-across-indexers-and-what-must-not-be)):

| Kind | Examples | On a rebuild |
|---|---|---|
| **Deterministic** | which events exist, author, rating, body; the head of each chain; whether a chain is withdrawn; which votes count for which version; which responses and labels attach to which chain | must come out identical |
| **Observed** | `received_at`, identity age, whether a domain claim verifies today, weights, an entity's rating, ranking, which moderators are followed | may come out different |

They go in separate tables from the first migration. On a rebuild every event
arrives at once, so every `received_at` becomes the time of the rebuild and
every weight built on it changes. With both kinds in one table, the rebuild
test would always find differences and could not tell a bug from the design
working. With them apart, the test compares the deterministic tables row by
row and ignores the observed ones.

### First tables

A starting point for the MVP's event types, not a schema. Each event is
written by `eventId`, so applying the same event twice changes nothing.

**Deterministic:**

| Table | From | Holds |
|---|---|---|
| `entities` | `entity.updated`, and the first review of an entity | the identifier, and every metadata proposal as its own row |
| `reviews` | `review.created`, `review.replaced` | one row per version: `event_id`, `chain_root`, `replaces`, `author`, `entity` (from the root), `rating`, `title`, `body`, `lang`, `created_at` |
| `review_chains` | the versions above, `review.withdrawn` | per chain: `head`, `withdrawn`, `truncated` |
| `votes` | `review.endorsed`, `review.disputed` | `event_id`, `target` (the exact version), `author`, `kind` |
| `responses` | `review.response_created` | `event_id`, `chain_root`, `target`, `author`, `body` |
| `labels` | `moderation.label_created`, `moderation.label_revoked` | `event_id`, `chain_root`, `moderator`, `label`, `revoked` |
| `profiles` | `profile.updated` | the latest per identity |
| `projector_cursors` | every event | per node: its `log_id` and the last `sequence` applied |

**Observed:**

| Table | Holds |
|---|---|
| `event_observations` | `event_id`, `received_at` (the earliest any node reported) |
| `identity_observations` | `identity`, first seen, age |
| `entity_summaries` | how many reviews gave each rating from 1 to 5, the mean, and the count ([Decided](#decided)) |
| `rankings` | the score each list is sorted by |

### Rules the projections follow

All of them are in the protocol, and most are already code: the
`projection` package of `protocol/go` implements them as pure functions and
runs them against the fixtures in
[protocol/fixtures/](../../protocol/fixtures/). The indexer calls those
functions and does not write its own version of any of them; two copies of
a rule are two places to disagree, and agreeing with other indexers is the
whole point of the deterministic tables.

| Rule | In `protocol/go` |
|---|---|
| **Chain head:** walk forward from the root; at a fork, higher `createdAt` wins and a tie goes to the lower `eventId`. Stop at 64 links and mark the chain truncated ([docs/protocol.md](../../docs/protocol.md#resolving-the-head-of-a-chain), `chain-head.json`) | `projection.ResolveHead`, `projection.Chain` |
| **Withdrawal** hides the whole chain, whichever version it named ([docs/protocol.md](../../docs/protocol.md#editing-and-deleting)) | `Head.Withdrawn`, from `ResolveHead` |
| **Votes bind to the exact version** they targeted; the head's score counts only votes on the head ([docs/protocol.md](../../docs/protocol.md#votes-do-not-follow-an-edit), `vote-and-label-binding.json`) | `projection.HeadScore` |
| **One vote per voter and version:** a voter's latest vote on a target replaces the earlier ones, by the same ordering as chain heads | `projection.CurrentVote` |
| **Labels and responses bind to the chain**; a revoked label no longer counts (`vote-and-label-binding.json`) | `projection.LabelsOnChain`, `projection.ResponsesOnChain` |
| **An entity identifier is derived from its descriptor**, and the first `entity.updated` for it carries the descriptor, so the indexer can check the derivation and treat an identifier that does not match with suspicion ([docs/decisions.md](../../docs/decisions.md) §1) | `projection.DeriveEntityID` |
| **Which entity metadata proposal is shown**, field by field: a recognized representative's latest value first (tier 1), else the first value, unless a challenger weighs over 1.2 times the incumbent (tier 2) (`entity-metadata.json`) | `projection.ResolveFields` |
| **A review's entity is fixed by the root of its chain.** `review.replaced` carries none ([docs/protocol.md](../../docs/protocol.md#reviews)) | indexer |
| **Profiles are last-writer-wins** by the same ordering as chain heads, which only ever compares one author's own events | indexer, with `projection.Event.Outranks` |
| **Authors are compared as identities, through key rotations**, not as raw keys ([docs/protocol.md](../../docs/protocol.md#who-may-sign-what), `identity-succession.json`) | indexer ([above](#what-the-indexer-may-assume)) |

The `projection` functions take a list of events and return an answer; they
keep no state. The indexer gives them what it read from its own tables: for
an edit or a withdrawal, every event of that chain, found by `chain_root`; for
a vote, the votes on that version. The tables hold the inputs and the last
answer, and the rule stays in the library.

**Metadata resolution takes weights and claims as input.** In v1 the indexer
has neither ([Decided](#decided)), so every weight is 0 and nobody is a
recognized representative: each field shows its first proposal, by
`createdAt`. That is the rule working with empty inputs, not a special case,
and when weights and claims arrive, the same call starts using them. Which
proposal is shown lives with the observed data; the proposals themselves are
deterministic.

### Applying an event

For each event on the stream, in one transaction:

1. Upsert what it changes in the deterministic tables.
2. Recompute what depends on it: the chain head, the vote counts of a version.
3. Record its observations.
4. Save `sequence` as the cursor of the node it came from.

The transaction is what makes a crash harmless. A crash before the commit
loses the work and the cursor together, so the same events arrive again; the
upserts make the second pass a no-op. Committing per batch rather than per
event is a later optimization and changes none of this.

## Output: the HTTP API

Read only, called by the gateway, which passes the answers on to the client.

```
GET /entities?q=…&category=…        search
GET /entities/{entity}              metadata, summary
GET /entities/{entity}/reviews      the head of each chain, ranked; withdrawn hidden
GET /reviews/{eventId}              one version, with its chain
GET /reviews/{eventId}/responses
GET /events/{eventId}               the envelope alone
GET /health                         includes how far behind each node the projector is
```

**Every item carries its whole signed envelope**, so the client can verify it
on its own. What the indexer worked out travels beside it, never mixed in:

```json
{
  "envelope": "{\"author\":\"did:key:z6Mk…\",…}",
  "derived": { "chainRoot": "evt_…", "head": true, "withdrawn": false, "endorsements": 3 },
  "observed": { "receivedAt": "2026-10-31T14:02:11Z", "rank": 0.82 }
}
```

`envelope` is the stored bytes as a string, not a parsed object. A JSON
parser on the way, in the API or in the gateway, could change a number, an
escape or the order of keys, and the client's check would fail on an event
that was valid. The gateway passes it through untouched.

`derived` is deterministic and `observed` is not, in the same split as the
tables, so a client knows which part another indexer must agree with.

### Reading what you just wrote

The projector runs a little behind the log, so a review published a moment
ago may not be in the listing yet. The client does not wait for it: after
publishing, it shows the review from the envelope it just signed, and lets
the listing catch up.

## Rebuilding

A first-class operation of the indexer, not a script:

1. Build into new tables (or a new schema) from cursor 0, alongside the live
   ones, which keep serving.
2. When the new tables have caught up with the nodes, switch the API to them
   and drop the old ones.

The same path is how a change to a projection's rules ships: change the rule,
rebuild, switch.

## Testing

| Test | Proves |
|---|---|
| Each fixture in `protocol/fixtures/` through the projector, with `fixturetest` | the projector feeds the rules what they need; the rules themselves are tested in `protocol/go` |
| Rebuild, then compare the deterministic tables row by row | the index is a projection of the log |
| Two instances fed the same log, compared the same way | two indexers agree on what they must |
| The same event applied twice | applying is idempotent |
| A crash between applying and committing, then a restart | no event is lost or counted twice |

The events come from a small generator that signs synthetic reviews with
`testevent.Sign` from `protocol/go`, which is also what the MVP's seeded
dataset needs ([docs/mvp.md](../../docs/mvp.md#in-scope)). The node's
admission limit, 100 events per author per hour, was set loose so that this
generator fits under it ([docs/decisions.md](../../docs/decisions.md) §15.4).

## Stack

Go, following the node's conventions where there is one to follow.

| Concern | Choice | Why |
|---|---|---|
| Language | Go 1.26 | The node's language and version; one language on the network side |
| Protocol | `protocol/go` (0.1.0) | The repository's protocol library; the same code the node validates with, and the projection rules |
| Stream from the node | `google.golang.org/grpc` | Same library and version as the node |
| Database | PostgreSQL 18, through `pgx/v5` (`pgxpool`) | Already in `infra/local`; `pgx` is the native driver, no `database/sql` layer |
| Queries | `sqlc` | Projections are upserts and the API is selects: plain SQL, checked against the schema, with typed Go generated from it |
| Migrations | `goose`, SQL files embedded in the binary | Plain SQL, the same files `sqlc` reads the schema from |
| HTTP API | `net/http` | Its pattern routing (`GET /entities/{entity}`) is all the API needs; no framework |
| Search | Postgres full text, the stemmer chosen by `lang` | What the MVP asks for, with no extra service |
| Logging | `log/slog`, JSON | As the node |
| Configuration | flags, with a Makefile that reads `.env` | As the node |
| Tests | `go test`, Postgres from `testcontainers-go` | Each test run gets a real, empty database |

One binary, with a subcommand per job:

```
indexer migrate     apply the migrations
indexer project     follow the node and write the projections
indexer serve       answer the HTTP API
indexer rebuild     rebuild the projections from cursor 0, then switch
```

```
indexer/
  cmd/indexer/      main: flags and subcommands
  internal/
    stream/         the SubscribeEvents client
    projector/      one applier per event type, and the transaction
    store/          sqlc's generated code
    api/            HTTP handlers
    protobuf/       generated from protobuf/node_api.proto
  migrations/       goose SQL
  queries/          sqlc SQL
  sqlc.yaml
  Makefile
  go.mod
```

**The protobuf code is generated again here, not imported.** The node's copy
lives in `node/internal/protobuf`, and Go forbids importing an `internal`
package from another module. Both generate from the same
`protobuf/node_api.proto`; the indexer maps its own `go_package` when it
runs `protoc`.

**The protocol comes from `protocol/go`.** The indexer imports it and
reimplements none of it:

| Package | The indexer uses it for |
|---|---|
| `envelope` | `Decode`: reading the fields of a stored envelope |
| `identity` | `Parse`: did:key; rotations are handled as [above](#what-the-indexer-may-assume) |
| `projection` | the deterministic rules ([Rules the projections follow](#rules-the-projections-follow)) |
| `testevent` | signing the generator's events |
| `fixturetest` | loading and running `protocol/fixtures/` in tests |
| `event`, `schema`, `references` | nothing at runtime: the node already ran them |

What the indexer adds on top is what the library leaves out: the tables,
incremental application, profiles, identities across rotations, an entity's
rating, moderation, ranking.

**The module follows the node's layout.** The node's module is
`github.com/eduardokairalla1/EcoaNetwork` and reaches `protocol/go` through
a `replace` to `../protocol/go`, which is why its image builds from the
repository root. The indexer does the same: module
`github.com/eduardokairalla1/EcoaNetwork/indexer`, a `replace` to
`../protocol/go`, a Dockerfile built from the root. No `go.work`: with each
module carrying its own `replace`, there is nothing left for it to link.

## Decided

- **The stream contract** as above: `SubscribeEvents`, `Cursor` and
  `StoredEvent`, with every field that may help; a cap on open streams per
  node.
- **More than one node**, each with its own cursor
  ([Reading from more than one node](#reading-from-more-than-one-node)).
- **The protocol is imported from `protocol/go`**, not rewritten, and that
  includes the projection rules it already has; the indexer adds only what
  the library leaves out ([Stack](#stack)).
- **An entity's rating** is this indexer's choice: the protocol defines one
  review's rating and nothing about combining them
  ([docs/protocol.md](../../docs/protocol.md#the-protocol-does-not-define-an-entitys-rating)).
  - **Shown:** the distribution, how many reviews gave each rating from 1
    to 5, beside the mean. A mean alone hides whether a 3 is everyone
    giving 3 or half giving 1 and half giving 5.
  - **Counted:** the head of each chain only. A withdrawn chain, a review
    labeled by a moderator this indexer follows, and a review by an identity
    holding a claim for that same entity (a conflict of interest,
    [docs/protocol.md](../../docs/protocol.md#conflicts-of-interest-are-visible-and-indexers-should-surface-them))
    are left out. The last is still listed, marked, just not counted.
  - **Not weighted yet.** Every counted review weighs the same. Weights come
    with reputation (identity age, claims); the summary is an observed table,
    so adding them later is a rebuild of that table and nothing else.
- **Moderation.** Anyone can publish a label; one counts only if its signer
  is followed.
  - **Who is followed:** a list of identities in a configuration file, read
    at query time. Changing it rebuilds nothing: every label is stored,
    whoever signed it, and only which ones count changes.
  - **One list for every query.** The configured list applies to everyone;
    a client cannot send its own yet ([Later](#later)).
  - **What a label does.** The protocol says what something is, never what to
    do about it
    ([docs/protocol.md](../../docs/protocol.md#moderation-and-votes)):

    | Label | Effect |
    |---|---|
    | `illegal`, `personal-data` | hidden, with no way to show it |
    | `spam`, `harassment`, `impersonation` | hidden by default; a client may ask to show it, marked |
    | `off-topic`, `other` | shown, marked |

    `illegal` and `personal-data` are the cases a court order or an erasure
    request is about, and the index is where such content leaves the read
    path without touching the log
    ([docs/mvp.md](../../docs/mvp.md#legal-exposure)).

## Later

Decided, and left out of the first version on purpose:

- **Clients choose their own moderators.** A query carries the moderators to
  follow, `?moderators=did:key:…,did:key:…`, and falls back to the configured
  list when it carries none. Every label is already stored, so this is a
  query change and no rebuild. **Needed before the MVP is done:** its
  acceptance criteria ask that a label change what a client following the
  moderator sees, and nothing for one that does not
  ([docs/mvp.md](../../docs/mvp.md#acceptance-criteria)), which one shared
  list cannot show.
- **The moderator list moves to a table edited through an admin API**, so it
  is managed without a deploy.
- **Weights in an entity's rating**, with reputation: identity age and
  claims. A rebuild of the summary table, nothing else.
