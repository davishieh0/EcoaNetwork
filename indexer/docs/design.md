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

- **The indexer is never the source of an event.** The node's log is the
  event store. The indexer does keep every envelope, because every item of
  its API carries one for the client to verify, but as a projection like any
  other table: it is never written to except from the log, and it comes back
  whole from reading the log again. A copy that could be written to on its
  own would be a second truth that can drift.
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
  and calls a key contested when it has two different successors
  (`identities.Resolve`, [The identities](#handing-an-event-to-its-subject)).
  If this rule is wanted in the library too, it is a small addition to
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

The first migration (`migrations/00001_base.sql`) creates only what every
event goes through, whatever its type: `events` (each envelope, as `bytea`),
`event_observations` and `projector_cursors`. The tables of each subject come
in a migration of their own, with the step that projects or reads it.

**The envelope is `bytea`, never `jsonb`.** `jsonb` reorders keys, drops
spacing and rewrites numbers; the bytes a client checks the signature over
would no longer be the bytes that were signed.

**Deterministic:**

| Table | From | Holds |
|---|---|---|
| `entities` | `entity.updated`, and the first review of an entity | the identifier, and the earliest `createdAt` of an event that named it |
| `entity_proposals` | `entity.updated` | every metadata proposal as its own row: `name`, `category`, `description`, `descriptor`, and `search_name`, the name folded for search |
| `reviews` | `review.created`, `review.replaced` | one row per version: `event_id`, `chain_root`, `replaces`, `author`, `entity` (from the root), `rating`, `title`, `body`, `lang`, `created_at` |
| `review_chains` | the versions above, `review.withdrawn` | per chain: `head`, `withdrawn`, `truncated` |
| `review_withdrawals` | `review.withdrawn` | `event_id`, `chain_root`, `target`: kept so a chain can be resolved again with every event that decides it |
| `votes` | `review.endorsed`, `review.disputed` | `event_id`, `target` (the exact version), `voter`, `kind` |
| `version_votes` | the votes above | per version: the endorsements and disputes that count, one per voter, the latest |
| `responses` | `review.response_created` | `event_id`, `chain_root`, `target` (a version or a response), `responded_version`, `depth`, `author`, `body`, `sort_at` |
| `labels` | `moderation.label_created`, `moderation.label_revoked` | `event_id`, `target`, `chain_root` or `response`, `moderator`, `label`, `note`, `revoked`; every label, whoever signed it |
| `profiles` | `profile.updated` | per identity, its current `profile.updated` and what it says; the one before is replaced entirely |
| `key_rotations`, `key_revocations` | `identity.key_rotated`, `identity.key_revoked` | every rotation and revocation any node accepted |
| `identities` | the rotations above | for each key that rotated or was rotated to, its identity (the first key of the chain), its place in it, and whether the identity is contested; worked out again from every rotation when one arrives |
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

### How it runs

`indexer project` is its own process, beside `indexer serve`, on the same
database.

```
node A --stream--> follower A -.
                                +--> channel --> applier (one) --> Postgres
node B --stream--> follower B -'
```

- **One follower per node** (`internal/stream`) opens `SubscribeEvents` from
  that node's saved cursor and hands every event to a channel. When a
  stream ends, it opens another after a wait that doubles from 1 s to 30 s,
  each wait drawn between half and all of it, so the readers of a node that
  went down do not all come back in the same instant. A stream that
  delivered something starts the wait over.
- **A stream always resumes from the cursor saved in the database**, never
  from the last event received: events received but not applied yet come
  again, and the upserts absorb them; none is lost.
- **One applier** (`internal/projector`) writes one event at a time, so two
  copies of an event from two nodes never race. The channel holds 64; past
  that the followers wait, and gRPC holds the nodes back.
- **A database error is retried, never skipped**: the events after it would
  be applied before it. **An envelope that cannot be read is skipped**, its
  cursor still saved, so the projector does not stop on it forever; the
  node validated it, so this should not happen.
- **A node is known by its address in `NODE_ADDRS`.** A node whose address
  changes is read from the start again: duplicates, which cost little.
- **The `log_id` needs nothing here**: a node whose log is another one sends
  its log from the start, and the cursor saved with the first event of it
  carries the new `log_id`.
- **Progress goes to the log** every 10 seconds per node: sequence, head,
  how far behind. Showing it in `/health` comes when something reads it.

### Handing an event to its subject

Every event goes into `events` and `event_observations`. Then, **the first
time the indexer sees it**, the applier hands it to the subject of its type,
in the same transaction:

- **One list of the types the indexer understands**, `subjects` in
  `internal/projector/applier.go`, as `api.go` is the one list of routes. A
  type missing from it is stored in `events` and nothing more; once it is
  added, a rebuild projects what was stored.
- **Only the first time.** `Save` reports whether the event was new; a copy
  from another node, or the same event read again after a reconnect, is not
  applied twice.
- **An event with no place is skipped, not retried**: an edit or a
  withdrawal whose target is no review the indexer holds
  (`review.ErrUnplaced`), or a payload that cannot be read
  (`review.ErrMalformed`). The node checked both, so neither should happen,
  and trying again would fail the same way. The event stays in `events`,
  the warning goes to the log, and the cursor moves on.

**The reviews** (`internal/review/projector.go`):

| Event | Does |
|---|---|
| `review.created` | a row in `reviews`, its own chain root, and the chain it opens, with itself as head |
| `review.replaced` | a row in `reviews`, in the chain of the version it edits and with the entity of that chain's root, then the chain resolved again |
| `review.withdrawn` | a row in `review_withdrawals`, then the chain resolved again |

**The votes** (`internal/vote/projector.go`): a `review.endorsed` or
`review.disputed` is stored against the exact version it names, and that
version's tally is worked out again with `projection.CurrentVote`: one vote
per voter, the latest by the author's ordering. A vote never follows an
edit, so a new version starts at zero and the old one keeps its votes. A
vote on anything but a review is valid and has no effect: it stays in
`events`, as the protocol's fixture says ("stored and ignored").

**The responses** (`internal/response/projector.go`): a
`review.response_created` lands in the chain of what it answers, a version
of a review or another response, so a conversation stays together across
edits. It records the version the conversation answered, so a client can
say when the text has changed since, and its depth. A response to anything
else is valid and has no effect, as a vote is. A response never sorts before
what it answers: `createdAt` counts whole seconds, so a quick reply shares
its parent's, and an author's clock can be behind; `sort_at` is then one
microsecond after the parent's.

**The labels** (`internal/label/projector.go`): every
`moderation.label_created` is stored, whoever signed it, against a review's
whole chain when it names one of its versions, so an edit does not escape
it, or against the response it names. A label on anything else stays in
`events` for now; labels on `entity.updated` come with the entities. A
`moderation.label_revoked` takes the label back, only from the moderator
who published it: the node checked it, and the update checks it again.
Reports (`moderation.report_created`) feed a moderator's queue, not the
reading path, and stay in `events`.

**The entities** (`internal/entity/projector.go`): an entity needs no event
to exist; the first `review.created` or `entity.updated` that names it
records it, with the earliest `createdAt` among them. Every `entity.updated`
is stored as a proposal, never as a fact. A label can name a proposal too,
and a proposal labeled by a followed moderator, whatever the label, leaves
the contest: a label on an edit says the edit is bad. Aliases
(`entity.alias_proposed`) are not projected by any v1 indexer and stay in
`events`.

**The profiles** (`internal/profile/projector.go`): a profile is always its
signer's. A `profile.updated` becomes the author's profile when it outranks
the current one by the author's own ordering (`projection.Event.Outranks`:
the later `createdAt`, then the lower eventId), so an old profile arriving
late, from a node that was away, does not overwrite the current one. A
profile is total: a field it leaves out is cleared. Until key rotations are
projected, a profile stays with the key that signed it.

**The identities** (`internal/identities`): one person can sign with several
keys over time. Every rotation and revocation is stored, and on each
rotation every identity is worked out again from all of them
(`identities.Resolve`, a pure function), since one rotation can join two
keys or contest an identity. Rotations are rare, so this is cheap, and the
result is the rule's whatever order they came in:

- A key with one successor passes its identity on; the identity is named by
  the first key of its chain.
- A key with two different successors, which the indexer can hold from two
  nodes that each kept a different first one, makes its identity
  **contested**: neither successor is honored, each starts an identity of
  its own, and what came before stays.
- A rotation that would close a cycle is dropped, taking them in the
  author's order.
- A key named as successor by two different keys joins neither: anyone can
  name any key, so honoring one would let a stranger take a key, and
  contesting would let a stranger contest anyone. The cost, accepted: such a
  stranger can split a successor off its identity, which only makes that
  person's keys count apart.

What follows the person rather than the key: **votes** count one per
identity, and a rotation counts again every version its keys voted on;
**profiles** are one per identity, the latest among all its keys', worked
out again when a rotation changes it; a **label** is revoked by the same
identity that published it; and the **moderators file** follows every key
of each identity it names, resolved when the API answers. Editing and
withdrawing a review need nothing here: the node checks them through
rotations. A revocation is stored and shown, and hides nothing: which events
fall after it is each node's observation.

**Resolving a chain again** loads every event that decides it, its versions
and its withdrawals, from their stored envelopes, and hands them to
`projection.ResolveHead`. The envelopes, not the columns: the rule compares
`createdAt` as the signed text and reads the payload. The whole chain is
resolved each time rather than patched, so the head is always the rule's,
whatever order the edits came in.

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
tables, so a client knows which part another indexer must agree with. Both
are always present, empty when there is nothing in them, so every item has
the same shape. Field names are camelCase, as in the protocol.

**A failure has the backend's shape**, a slug a program can branch on and a
message, so the gateway can pass it on or translate it without converting:

```json
{ "error": "event_not_found_error", "message": "No event with this id." }
```

| Case | Status | `error` |
|---|---|---|
| malformed id | 400 | `invalid_event_id_error`, `invalid_entity_id_error` |
| malformed page | 400 | `invalid_limit_error`, `invalid_cursor_error` |
| nothing with that id | 404 | `event_not_found_error`, `review_not_found_error`, `entity_not_found_error`, `profile_not_found_error` |
| not a did:key, or `?ids=` empty or over 100 | 400 | `invalid_identity_error`, `invalid_ids_error` |
| `q` longer than 120 characters | 400 | `invalid_query_error` |
| a review its author withdrew | 410 | `review_withdrawn_error` |
| removed by a followed moderator | 451 | `removed_by_moderation_error` |
| `?labeled=` other than `show` | 400 | `invalid_labeled_error` |
| the database does not answer | 503 | `database_unavailable_error` |
| anything else | 500 | `internal_error` |

`/health` answers 200 when the database answers, and 503
`database_unavailable_error` when it does not. The API starts even with the
database down, and recovers when it comes back.

**Every version carries its votes** in `derived`: `endorsements` and
`disputes`, the ones that count on that very version. The head's are its
score; an old version's stay visible as history.

**An entity's reviews are listed newest original first**: the head of each
chain, ordered by the `createdAt` of the chain's original, never of its
head, so an edit does not move a review while someone pages through. That
date is the author's, but a node refuses one more than five minutes ahead,
so nobody pins a review to the top. Ranking, an observed order, replaces it
later. Withdrawn and truncated chains are left out. An entity with no
reviews answers an empty list, not 404: until the entities table exists,
the indexer cannot tell an unknown entity from one nobody reviewed yet.

**A review is returned with every version of its chain**, forks included,
oldest first, each with its envelope; `derived.head` marks the current one,
and any version can be asked for. **A withdrawn review answers 410** with
nothing of what it said: withdrawing is how an author takes a review back,
`privacy` among the reasons, and the review page is where that has to show.
Its envelope stays reachable at `/events/{eventId}`, for inspection, as it
stays on every node's log.

**A review's responses** (`GET /reviews/{eventId}/responses`, any version of
it) come in the order of the conversation, a page at a time, each with
`replyTo`, `respondedVersion`, `textChanged` (the version answered is no
longer the current one) and `depth`. The protocol leaves the cap on nesting
to each indexer: deeper than 2 is shown at 2, still saying what it answers,
so none is hidden and a client never indents more than twice; the stored
depth is the real one. A withdrawn review answers 410 here too.

**Labels act at query time.** Which ones count is who this indexer follows
(`MODERATORS_FILE`, read when `serve` starts), so changing the list rebuilds
nothing, and the labels that count are **observed**, not derived: another
indexer, following others, shows others.

| Label (by a followed moderator) | Lists | The item's own route, `/events/{eventId}` |
|---|---|---|
| `illegal`, `personal-data` | left out, even with `?labeled=show` | `451 removed_by_moderation_error`, nothing of it; its responses too |
| `spam`, `harassment`, `impersonation` | left out, unless `?labeled=show`, then marked | shown, marked: asking for it by id is asking to see it |
| `off-topic`, `other` | shown, marked | shown, marked |

The filter is in the query, so a page is never short or skipped because of
what it left out. 451 is the status for what is withheld for legal reasons,
which is what `illegal` and `personal-data` are about; `/events/{eventId}`
answers it too, or removed content would still leave the indexer by that
route.

**An entity's metadata is worked out when the API answers.** For each field,
`projection.ResolveFields` picks the value from the proposals still in the
contest: a recognized representative's latest first, otherwise the first
value unless a challenger outweighs it by 20%. Weights and claims are what
this indexer observes, so the result is `observed`, with the proposal that
set each field; changing who is followed, or later the weights, rebuilds
nothing. v1 has neither weights nor claims, so each field holds its first
proposal; when they come, the same call starts using them.

`GET /entities/{entity}` answers the value of each field, how many proposals
there are and, for an `entity:` identifier, its descriptor and whether its
hash is the identifier. An entity nothing named answers 404; one only
reviewed answers its fields empty. `GET /entities?q=` finds entities with a
name, any of their proposals' not labeled, that holds `q` ignoring case and
accents, newest first, every entity without `q`. Filtering by category waits
for step 6: the category that holds is worked out at answer time, and
filtering by it in SQL would repeat the rule there.

**A profile** comes from `GET /profiles/{did}`, as a signed item, or many at
once from `GET /profiles?ids=` (up to 100, in the order asked, those with
none left out), so a client gets the authors of a page in one call; reviews
do not embed them. A display name is verified by nothing: a client shows it
beside the identity, never as if it were one. A label names one
`profile.updated` and stays with it; a new profile, a new statement, does not
inherit it. `illegal` and `personal-data` answer 451 and leave a list; the
others mark it.

**An identity** comes from `GET /identities/{did}`, through any of its keys:
its name, its keys in the order of its chain, the one it signs with now,
whether it is contested, and its revocations. A contested identity is the
sign of a stolen key, and a client should say so.

**Lists page by cursor**, `?after=<cursor>&limit=20` (at most 100), never by
page number. A review published while someone scrolls shifts every page by
one, so a page number shows an item twice or skips one; a cursor says
"after this one" and does neither.

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
| Each fixture in `protocol/fixtures/` through the projector, with `fixturetest` | the projector feeds the rules what they need and stores what they say; the rules themselves are tested in `protocol/go` |
| Rebuild, then compare the deterministic tables row by row | the index is a projection of the log |
| Two instances fed the same log, compared the same way | two indexers agree on what they must |
| The same event applied twice | applying is idempotent |
| A crash between applying and committing, then a restart | no event is lost or counted twice |

A fixture's events are signed for real by `fixturetest`, so their eventIds
are real hashes, not the fixture's names. Each case is checked against what
`projection.ResolveHead` says of those events, and against the fixture's
own expectation except where it depends on the order of the names (a tie on
`createdAt`). Events are applied with every target before what names it, as
a node's log has them.

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
| Stream from the node | `google.golang.org/grpc` | The node's library. v1.85, one minor above the node's v1.84, which goose needs; the wire protocol is the same |
| Database | PostgreSQL 18, through `pgx/v5` (`pgxpool`) | Already in `infra/local`; `pgx` is the native driver, no `database/sql` layer |
| Queries | `sqlc`, from the first step with a query (the API) | Projections are upserts and the API is selects: plain SQL, checked against the schema, with typed Go generated from it |
| Migrations | `goose`, SQL files embedded in the binary | Plain SQL, the same files `sqlc` reads the schema from. goose runs on `database/sql`, so `migrate` uses pgx's `database/sql` driver; everything else uses pgx directly |
| HTTP API | `net/http` | Its pattern routing (`GET /entities/{entity}`) is all the API needs; no framework |
| Search | Entity names: by any part, ignoring case and accents, against a folded copy of each name (`golang.org/x/text`) | A stemmer cuts names ("padaria" becomes "padar") and people search a name by a piece of it. Postgres's `unaccent` and `pg_trgm` would do it in the database, but extensions are per database and the tests run in parallel, each in a schema of its own; the trigram index can come when the volume asks for it. Searching review text, with full text and the stemmer chosen by `lang`, is later |
| Logging | `log/slog`, JSON | As the node |
| Configuration | environment variables only, with a Makefile that reads `.env` | As the node |
| Tests | `go test`, against `ecoa_indexer_test` from `infra/local` | A real database, emptied by each test, as the backend does; no extra dependency. Without it, the database tests skip and say why |

One binary, with a subcommand per job:

```
indexer migrate     apply the migrations
indexer project     follow the node and write the projections
indexer serve       answer the HTTP API
indexer rebuild     rebuild the projections from cursor 0, then switch
```

### Layout: one package per subject, in layers

The code is organized by subject (`review/`, `entity/`), and each subject is
layered: handler, service, repository. Platform code that no subject owns
sits apart.

```
indexer/
  cmd/indexer/          main: picks the subcommand, nothing else
  internal/
    config/             the environment variables
    platform/
      database/         the Postgres pool and the migrations
    api/                Serve, the router (the one list of routes), /health
    stream/             one follower per node: SubscribeEvents, reconnecting
    projector/          the one applier: opens the transaction, hands each
                        event to its subject, saves the cursor
    platform/
      respond/          the answer shapes every route shares: Item, Error
    platform/ids/       whether a string is a well-formed eventId or entity id
    platform/page/      the paging every list shares: the cursor, ?limit=
    label/              labels, the moderators file, what each label does
    entity/             entities, their proposals, GET /entities, search
    profile/            profiles, GET /profiles
    identities/         keys to identities, contests, GET /identities
    event/              GET /events/{eventId}, the events table
    review/
      handler.go        GET /entities/{entity}/reviews, GET /reviews/{eventId}
      service.go        only when there is logic to put in it
      projector.go      applies review.created, review.replaced, ...
      repository.go     methods over the generated queries
      model.go
      queries.sql       the SQL, written by hand
      sqlc/             generated from queries.sql, never edited
    entity/             the same shape
    protobuf/           generated from protobuf/node_api.proto
  migrations/           goose SQL
  Dockerfile
  Makefile
  go.mod
```

Three things differ from the usual shape of this pattern, because the
indexer is not a usual API:

- **Each subject has two ways in.** The HTTP API only reads, so the rules
  (which version is the head, which votes count) do not run on a request.
  They run when an event arrives from a node, in the subject's
  `projector.go`. The handler and the projector share the repository: CQRS
  inside each subject.
- **The projector owns the transaction.** One event changes several
  subjects at once (a vote: the event, the vote, the review's score, the
  node's cursor), in one transaction. So no repository opens its own: each
  takes the transaction, or the pool for a read, as a parameter.
- **Every database operation is a method of a subject's repository.** The
  SQL is written by hand in the subject's `queries.sql`; `sqlc` turns each
  query into a typed function in the subject's `sqlc/` folder, checked
  against the migrations; `repository.go` wraps those functions in methods
  named for the subject (`ByID`, `Head`). Handlers and projectors call the
  methods and never write SQL. The generated code sits in a folder of its
  own because `sqlc` also writes a struct for every table, which would mix
  with the subject's own code.
- **Shared answer shapes live in `platform/respond`,** not in `api`: `api`
  imports the subjects to register their routes, so a subject importing
  `api` back would be an import cycle.
- **`service.go` only where there is logic.** Most reads are "query and
  answer", and a service that only forwards to the repository is a file
  with nothing in it. One appears where a read has a rule to apply, such
  as filtering labels by the moderators followed.

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
