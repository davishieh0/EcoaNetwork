# Architecture

This describes the *roles* components play and how they relate. For the same material as diagrams — the publish path, the validation branches, sync, the determinism boundary — see [docs/diagrams/diagrams.md](./diagrams/diagrams.md). That's a decision for whenever each piece actually gets implemented, and different implementations of the same role are expected to coexist. See [docs/protocol.md](./protocol.md) for the one thing every implementation must actually agree on.

## High-level flow

```
                    User
                     │
                     ▼
        Client (e.g. a web frontend)
         signs locally, holds the key
                     │
        writes ──────┴────── reads
          │                    │
          ▼                    ▼
              Gateway API
          │                    │
          │ signed event       │ query
          ▼                    ▼
        Node ──────────────► Index
          │   local-sequence   ▲
          │   cursor           │
          │        ┌───────────┘
          │        │ projects
          │     Indexer
          │
          │ libp2p / Gossipsub + sync
          ▼
   P2P Network — Node A ↔ Node B ↔ Node C ↔ community nodes
```

Writes go left-to-right through a node and out to the network; reads never touch a node at all. The indexer sits between the two, pulling validated events off a node and projecting them into the index the gateway queries.

## Roles

| Role | Responsibility |
|---|---|
| **Protocol** | Shared contract: event types, canonical serialization, signing, versioning. See [docs/protocol.md](./protocol.md). |
| **Client** | UI, local identity, signs events client-side, verifies signatures it receives, lets the user pick a gateway/indexer. A web frontend is the obvious first client, but nothing about the protocol requires one. |
| **Gateway API** | HTTP entry point for light clients. Validates request shape, queries an index for reads, forwards signed events to a node for writes. Never signs on behalf of a user. **Read responses carry whole signed envelopes**, not flattened rows — a client that receives a prettified projection has nothing left to verify, which would quietly turn client-side verification into a decoration. Derived fields an index adds (scores, labels, resolved chain heads) travel alongside the envelope, clearly separated from it. |
| **Node** | The real network participant. Validates events cryptographically, stores them with a local receive sequence, gossips new events, syncs missed events with peers, and buffers events whose references haven't arrived yet. |
| **Indexer** | Consumes validated events from a node, projects them into a queryable index (reviews, entities, reputation, moderation labels). Rebuildable from scratch. |
| **Index** | Query layer only. Not the source of truth — the node's event log is. |

## Why this split

- **libp2p transports bytes; the protocol gives them meaning.** libp2p (the chosen P2P substrate — see [docs/concepts.md](./concepts.md#network-vs-protocol)) handles peer discovery, secure connections, multiplexing, pub/sub, NAT traversal. Ecoa Network defines what an event *means* and how peers *sync*. This part isn't stack-dependent: libp2p has independent, interoperable implementations in multiple languages.
- **The gateway API is not a node.** Mixing HTTP/auth/search concerns into the P2P layer would turn the gateway into a half-built node. Keep them separate: the gateway speaks HTTP and understands the app model; the node speaks libp2p and understands the distributed network.
- **A web client is a light client, not a full node.** Browsers can't hold long-lived TCP connections, have limited storage, and get suspended/closed. So in v1 it signs locally and talks to a gateway — it does not run its own P2P stack. A different kind of client (desktop, mobile) could make a different call here.

## Node communication patterns

- **Gateway → Node:** one *primary* node configured per gateway deployment, with a fallback list. Not a random peer lookup — the gateway always knows exactly which node it talks to.
- **Publishing:** the gateway sends to the primary node; if unavailable, falls back to the next. Because `eventId` is a content hash, publishing is idempotent — sending to multiple nodes is safe (duplicates just return `already_exists`).
- **Node → Node:** libp2p. New events propagate via **Gossipsub** (real-time). Missed events are recovered via a small **sync protocol** over libp2p streams — two calls, `GetEventsSince(cursor)` to catch up and `GetEvents(ids)` to fill a known hole — because pub/sub alone loses messages for any peer that was offline. See [docs/protocol.md](./protocol.md#node-to-node-sync).
- **Node → Indexer:** the indexer polls the node for new events using a cursor over the node's **local receive sequence** — never over `createdAt`, for the same reason node-to-node sync doesn't ([docs/protocol.md](./protocol.md#node-to-node-sync)). The node can push a notification to speed things up. Because the cursor is a position in one node's sequence, an indexer that switches to a different node re-reads from zero rather than resuming; at v1 volumes that's a full rebuild, which is a thing the design already requires to work.
- **Reads never touch the node.** `Client → Gateway → Index` for search/browse. The node is only involved in publishing, fetching a raw event by ID, and feeding the indexer.

## Integrity is verifiable; completeness is not

A client can check everything the index hands it — signature, hash, author, references. It has no way to notice an event the index chose *not* to hand it, because there's nothing to check. That's the structural limit of this read path, and it's worth naming rather than glossing: **querying an index means trusting it for completeness, and no amount of client-side cryptography changes that.**

The available answers are plural rather than cryptographic — query more than one indexer and compare, which only works once more than one exists. A completeness commitment an indexer could publish and a client could audit (a periodic signed event count, or a Merkle root over what it holds) would turn omission into something detectable, but that's an open design question, not something v1 solves.

Fetching a raw event by ID straight from a node is the one read that bypasses the index entirely. It's useful for confirming a specific event exists; it's useless for discovering one you were never shown.

**Derived fields are unverifiable by construction.** A read response carries whole signed envelopes plus whatever the index computed alongside them — scores, applied labels, the resolved head of a chain. The envelopes can be checked; the derived fields cannot, because there is no signature over a computation. A gateway that reports the wrong chain head, or omits a label, or inflates a score, passes every check a client can run.

A client that actually cares has one option and it's an honest one: fetch the envelopes and recompute. Chain head resolution, vote-to-version binding and withdrawal state are all on the deterministic side of the boundary below, so a client holding the relevant events can derive them itself and compare. Anything on the *local observation* side — weight, ranking, an entity's rating — cannot be checked even in principle, because there is no single right answer to check against.

## What must be identical across indexers, and what must not be

"The index is a projection, rebuildable from the log" is the project's central claim, and it is only true of *part* of the index. Being precise about which part is the difference between a testable property and a slogan — and it's what tells an implementer whether a difference between two indexers is a bug or expected behavior.

**Deterministic — two honest indexers holding the same events must agree, and a rebuild must reproduce it exactly:**

- Which events exist, their content, and their authorship.
- Structural validity: size, envelope shape, digest, signature, schema, and whether the author was entitled to act on the target.
- The head of any replace chain, and whether a chain is withdrawn.
- Which votes and responses attach to which exact version of a review.
- Which entity metadata proposals exist and what each one asserts.
- Every ordering *within a single author's* events.

**Local observation — two honest indexers may legitimately differ, and a rebuild is not expected to reproduce it:**

- `receivedAt` and identity age, which depend on when *this* node first saw an event.
- Whether a `domain` claim currently verifies, which depends on a DNS lookup at the moment of asking.
- Reputation and vote weight, since both are built on the two above.
- Which verifiers and moderators are recognized, which is policy and always was.
- **Which entity metadata proposal is canonical**, because with no votes it resolves on the author's weight ([docs/entities.md](./entities.md#2-unverified-edit--a-proposal-not-a-fact)), and weight is a local observation. The *inputs* are deterministic and the rule is deterministic; the weights feeding it are not.
- **An entity's rating.** The protocol defines what one review's `rating` means and nothing about how a set of them becomes a number ([docs/protocol.md](./protocol.md#the-protocol-does-not-define-an-entitys-rating)) — mean, median, distribution, weighted or not, which reviews are excluded. There is no canonical average, deliberately.
- **The revocation cut** — whether an event signed shortly after an `identity.key_revoked` is accepted depends on whether this node had seen the revocation yet ([docs/protocol.md](./protocol.md#when-rotation-and-revocation-take-effect)). This is the one place validity itself is an observation, and it's a deliberate trade: the alternative lets an attacker place the cut.
- Ranking and default sort order, which are the whole point of allowing plural indexers.

Note the shape of the two entries in bold. Neither is a rule that differs between indexers — the rules are identical and written down. What differs is an *input* that no implementation can derive from the log alone. That's the honest test for which list something belongs in: not "is the rule shared" but "can it be recomputed from the events and nothing else".

Two consequences worth stating. First, "the index can be dropped and rebuilt from the logs" is an acceptance criterion about the **deterministic** half — the second half is *re-derived*, not replayed, and may honestly come out different. Second, a disagreement between indexers in the first list is a bug in one of them; a disagreement in the second list is the design working. Anything that would move an item from the second list to the first requires electing an authority over clocks, DNS, or ranking, which is the thing this architecture exists to avoid.

## Multiple operators

Nothing in the *architecture* is a single point of control — though in v1 a single operator runs all of it, so this is a property of the design rather than of the deployment ([docs/concepts.md](./concepts.md#progressive-decentralization)):

```
official-site.com  →  official gateway    →  Node A
community.org      →  community gateway   →  Node B
university.edu     →  university gateway  →  Node C
```

All of them read/write the same network and can independently verify every event's signature. A user can switch gateway, indexer, or client without losing their identity or history — that substitutability *is* the decentralization, more than the raw node count.

### Substitutability only counts if the client isn't served by the gateway

For a web client, "you can point it at another gateway" is close to meaningless if the same operator also ships the JavaScript. Whoever serves the code can serve code that signs something else, or exfiltrates the key outright — no CSP, non-extractable key, or signing prompt survives an attacker who controls the application itself. The threat isn't hypothetical or exotic; it's the ordinary consequence of one party controlling both the code and the data path.

So the split is structural, not cosmetic:

- **The client is a static bundle** — versioned, ideally reproducible from source, and hostable by anyone, including the user. It ships no server-side rendering, no per-request code generation, and no gateway-injected configuration.
- **The gateway is a data-only API.** It never serves application code.
- **Gateway choice is client-side runtime configuration**, so switching gateways changes where data comes from and does not change which code is running.

This is what makes the MVP's "the client can be pointed at a different gateway without code changes" a security property rather than a convenience feature. It is also why key handling in [docs/identity.md](./identity.md#key-storage-client-dependent) is a client concern that no gateway can help with — and no gateway should be in a position to hurt.

## Deployment and repository layout

Deliberately not specified here. Where things get deployed, how they're packaged, and how the repository is organized all follow from stack decisions that haven't been made yet — see [docs/mvp.md](./mvp.md). Once an implementation exists, its own deployment story belongs next to it, not baked into this document.
