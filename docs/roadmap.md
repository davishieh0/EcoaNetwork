# Roadmap

The order to build things in, and why that order. [mvp.md](./mvp.md) says *what* v1 contains and *when it's done*; this says *what to do next*.

Two principles run through the ordering:

- **Push the risky thing later, but not last.** libp2p is the single largest schedule risk. Everything that can be correct without a network is built and tested first, so when networking starts, a bug there is unambiguously a networking bug — and if libp2p has to be swapped for the declared fallback, nothing built before it is wasted.
- **Instrument while building, not after.** The evaluation metrics are decided in step 0 so counters go in as each piece is written. Retrofitting measurement onto finished code is how a results chapter turns into guesswork.

## Status

As of 2026-11-24. The acceptance criteria, one by one, are in [mvp.md](./mvp.md#where-v1-stands); what remains in the indexer, in [its roadmap](../indexer/docs/roadmap.md#what-remains-for-the-mvp-indexer).

| Step | State | What is open |
|---|---|---|
| 0. Decisions | Done | — |
| 1. Spec artifacts | Done | two fixture cases cannot be signed as written ([indexer roadmap](../indexer/docs/roadmap.md#what-remains-for-the-mvp-indexer)) |
| 2. Core library | Done: `protocol/go` | `protocol/ts`, a second implementation |
| 3. Node storage and validation | Done | — |
| 4. Networking | Done: libp2p, Gossipsub, the two-call sync, three nodes | — |
| 5. Indexer and index | Done: every v1 type, the read API, `rebuild`, the guarantees tested | unknown types counted, lag in `/health` |
| 6. Gateway | Done: publish with failover, whole envelopes on reads | rate limiting |
| 7. Client | Done, including the inspection view, fresh identities, and replace, withdraw and responses | signature checks under ZIP-215 (WebCrypto checks under RFC 8032 until `protocol/ts`) |
| 8. Domain verification | Done, DNS over DoH or a configured server | the `.well-known` fallback |
| 9. Seeding and the demonstration | Done: the generator, the second indexer with another ranking | importing the real Receita dump, rehearsing |
| 10. Cuttable social features | Done: profiles, labels, moderators chosen by the reader | reports (`moderation.report_created` is not projected) |
| 11. Self-hosting | Started: a container image, operator docs in `node/README.md` | a published image, a public bootstrap node, metrics, a status dashboard |
| Academic track | Related work written | the metrics below are not collected yet beyond logs (the indexer logs progress and rebuild time; a rebuild of 1712 events took about 5 s); legal policy |

## Dependencies

```mermaid
flowchart TD
    S0["0. Decisions<br/>payloads, stack, network name"]
    S1["1. Spec artifacts<br/>schemas + test vectors"]
    S2["2. Core library<br/>envelope, canonicalization, signing"]
    S3["3. Node storage and validation<br/>single node, no network"]
    S4["4. Networking<br/>libp2p, Gossipsub, sync"]
    S5["5. Indexer and index"]
    S6["6. Gateway"]
    S7["7. Client"]
    S8["8. Domain verification"]
    S9["9. Seeding and demo"]
    S10["10. Cuttable social features"]
    S11["11. Self-hosting"]
    A["Academic track<br/>related work, evaluation, legal"]

    S0 --> S1 --> S2 --> S3 --> S4
    S3 --> S5
    S4 --> S5
    S5 --> S6 --> S7
    S7 --> S8
    S8 --> S9
    S5 --> S9
    S7 --> S10
    S4 --> S11
    S0 -.->|"runs in parallel throughout"| A
```

Note that **step 5 needs step 3 but not step 4**: an indexer can be built and tested against a single node's log while networking is still in progress. If libp2p stalls, the indexer, gateway and client are not blocked.

---

## 0. Decisions that block everything

Nothing here is code, and none of it can be deferred past the first line of it.

| Decision | Why it can't wait |
|---|---|
| ~~Payload schema per event type~~ | **Settled** ([protocol.md](./protocol.md#payloads)) — no longer a blocker. The remaining decisions below are all about implementation, not design. |
| ~~Language for the node~~ | **Settled: Go**, with `ed25519consensus` for verification and `gowebpki/jcs` for canonicalization ([decisions.md](./decisions.md#132-the-node-is-written-in-go--call)). `go-libp2p` being the reference implementation is the cheapest available reduction of the largest risk on the board. |
| **Language for gateway, indexer, client** | Independent of the node — they speak HTTP and SQL, not libp2p. Can be whatever you're fastest in. |
| **Development network name** | `ecoa-dev`, or whatever you pick. It goes inside the signed bytes ([protocol.md](./protocol.md#network-separation)), so choosing it after events exist means re-signing all of them. |
| **Which metrics to collect** | See [the academic track](#academic-track-runs-in-parallel). Deciding now means counters get written alongside the code that needs them. |

**Done when:** all four are written down somewhere you'll look again. With payloads settled, none of these is a design question any more — they're choices about how to build, and the protocol no longer waits on any of them.

---

## 1. Spec artifacts

The interop contract. Everything downstream is tested against these, and for an academic project they're a citable artifact in their own right.

1. **JSON Schema per event type** — a mechanical transcription of [the payload spec](./protocol.md#payloads), including the enums and the per-field length limits.
2. **Canonicalization and signing test vectors** — input object, expected JCS bytes, expected `signing_input`, expected `eventId`, expected signature. This is what makes two independent implementations provably compatible.
3. **ZIP-215 signature vectors** — the edge cases where implementations disagree, each with its expected verdict ([protocol.md](./protocol.md#signature-verification-follows-zip-215)). Note that these are *not* all rejections: non-canonical `S` must be rejected, but a non-canonical point encoding and a small-order point must be **accepted**, and a stricter library will fail those. Getting a vector's expected verdict backwards here produces exactly the partition the vectors exist to prevent.
4. **Behavioral fixtures** for the rules that are easy to implement subtly differently: chain head resolution with a fork, conflicting rotations, orphan buffering and release, vote-to-version binding across an edit, per-field entity resolution.

**Done when:** a fixture file exists for each, with expected outputs, runnable before any implementation exists.

---

## 2. Core library

Not the node — the protocol, as a library with no network and no storage. Everything in this step is a pure function.

1. Envelope encode/decode, closed-set validation, required fields present.
2. JCS canonicalization, NFC normalization, duplicate-key rejection, nesting depth cap.
3. `eventId` and the domain-separated `signing_input`.
4. Strict Ed25519 signing and verification, `did:key` encode/decode.
5. Validation checklist steps 1–8 — everything that needs no stored state.

**Why separate from the node:** you can pass the entire vector set from step 1 with zero infrastructure. Every later bug is then not a canonicalization bug, which is the hardest kind to find once a network is in the picture.

**Done when:** every vector from step 1 passes, including all the rejection cases.

---

## 3. Node storage and the rest of validation

Still no network. A single process that accepts events and gets every rule right.

1. Event store, with a local receive timestamp and monotonic local sequence.
2. Validation steps 9–11: reference resolution, authorship checks, the orphan buffer with its bounds, admission policy.
3. Derived state the node itself needs: replace chains and head resolution, rotation and revocation state with first-observed semantics.
4. Tombstones — the shape only. No v1 node prunes ([protocol.md](./protocol.md#sync-against-a-peer-that-pruned-data)).

**Done when:** feeding the behavioral fixtures from step 1 through a single node produces the expected accept/reject/buffer outcome for each. Acceptance criteria covered: signature and schema rejection, oversize, unknown envelope field, future-dated `createdAt`, out-of-order references, conflicting rotations.

---

## 4. Networking

The risky step, and deliberately isolated so that its risk is contained.

1. libp2p host, identity, transport, connection manager limits.
2. Bootstrap and peer discovery, with the list overridable from configuration on day one.
3. Gossipsub on the single topic `/ecoa/events/1`.
4. Sync: `GetEventsSince(cursor)` and `GetEvents(ids)`, with per-peer rate limits.
5. Three nodes running locally.

**Done when:** an event published to one node reaches the other two; a node taken down and brought back catches up through sync; taking one node down doesn't stop propagation between the others.

**If this stalls:** the declared fallback is plain gossip over WebSocket with a static peer list ([mvp.md](./mvp.md#key-risks)). It replaces this step and only this step — steps 2 and 3 are untouched, because the protocol is meaning and libp2p is only transport.

---

## 5. Indexer and index

Can start as soon as step 3 is done; doesn't wait for networking.

1. Cursor consumer over the node's local sequence.
2. The deterministic projection: which events exist, chain heads, withdrawal state, vote-to-version binding, entity proposals ([architecture.md](./architecture.md#what-must-be-identical-across-indexers-and-what-must-not-be)).
3. Local observations, kept explicitly separate: identity age, claim weight, entity metadata resolution.
4. Full-text search.
5. Rebuild from log, as a first-class operation rather than a script someone wrote once.

**Done when:** dropping the index and rebuilding reproduces the deterministic half exactly, and two indexer instances fed the same log agree on all of it. This is also where the second-indexer demo comes from — same binary, two config files.

---

## 6. Gateway

Small, and worth keeping small.

1. Publish endpoint: validate shape, forward to the node, never sign, never modify.
2. Read endpoints returning **whole signed envelopes**, with derived fields alongside and clearly separated.
3. Rate limiting **without logging source addresses next to event content** ([identity.md](./identity.md#and-the-network-layer-knows-more-than-the-protocol-does)).
4. Configurable primary node with a fallback list.

**Done when:** a review published through the gateway lands on all three nodes, and reads return envelopes a client can verify independently.

---

## 7. Client

1. Key generation and storage, non-extractable where the platform allows.
2. Build, sign and publish an event locally.
3. Browse, search, entity page.
4. Client-side signature verification of everything received.
5. **Inspection view** — raw envelope beside the rendered review, live verification, which nodes hold it. Demo-critical.
6. Fresh-identity publishing as a visible option, with the trade stated at that moment.
7. Review replace and withdraw.
8. Review responses.

Items 7 and 8 sit here rather than in step 10 because the demonstration turns on them: editing after an endorsement, and a company response with an official badge.

**Done when:** identity creation with no signup, publishing without the gateway ever touching the key, and pointing the client at a different gateway by configuration alone.

---

## 8. Domain verification

After the client, because it needs a client to publish the claim and an indexer to check it.

1. DNS-over-HTTPS lookup in the indexer, DNSSEC preferred where the zone is signed.
2. Claim recognition, `expiresAt`, TTL re-check.
3. Tier-1 entity authority and the official-response badge.

**Done when:** publishing a TXT record and a self-signed claim produces a verified representative who can edit the entity and post a badged response — with no verifier service anywhere in the loop.

---

## 9. Seeding and the demonstration

1. Import entities from public Receita Federal data.
2. Synthetic review generator producing properly-signed events across them.
3. Second indexer instance with a different ranking configuration.
4. Rehearse the ten steps in [mvp.md](./mvp.md#the-demonstration).

**Why this is a step and not an afterthought:** a demonstration with four reviews reads as a toy regardless of what's underneath it, and the second indexer — the step that shows *why* the project is built this way — costs one config file and is the easiest thing to forget.

---

## 10. Cuttable social features

Profiles, reports, moderation labels and filters.

**This is the cut line.** If the schedule slips, these are described as designed-but-not-implemented and the central demonstration is unaffected. Everything before this point is not cuttable.

---

## 11. Self-hosting

Container image, node operator documentation, bootstrap list, metrics endpoint, status dashboard.

Mostly free if the node is a single binary, and it's what makes "a third party can run a node" a demonstrated claim rather than an asserted one.

---

## Academic track, runs in parallel

Not a phase. These are worth starting early and are actively harmed by being left to the end.

**Related work — do this first, before step 2.** The nearest neighbours are [Nostr](https://github.com/nostr-protocol/nips) (signed events, content-addressed ids, dumb relays, pluggable clients), [AT Protocol](https://atproto.com/) (signed repositories as truth, relay firehose, AppView as projection — architecturally very close to the log/index split here), [Secure Scuttlebutt](https://scuttlebutt.nz/) (append-only signed feeds, gossip replication), and ActivityPub (federated but server-authoritative, and the furthest away).

Writing this early is not administrative work: it forces the question of what this design actually contributes, and it already changed the answer. See [related-work.md](./related-work.md).

The research corrected an overclaim before it reached a defence. Moderation-as-pluggable-labels was being described here as a contribution; **AT Protocol shipped it first and at scale**, as subscribable labelers. Converging on the same design independently is a signal that the design is right, not evidence of originality.

What survived as genuinely different is narrower: authority over data nobody owns, weighted claims as the Sybil defence, vote-to-version binding, and a written determinism boundary. What did *not* survive is worth knowing before someone in a defence points it out.

**Evaluation metrics — decided in step 0, collected throughout.** Deciding them before building is what lets each counter go in beside the code that produces it; retrofitting measurement onto finished code is how a results chapter turns into guesswork.

| Metric | Unit | Where the counter lives | Why this one |
|---|---|---|---|
| Propagation latency | ms, as a distribution | Node: accept timestamp, compared across the three | Quantifies the claim that the network actually replicates. Reported separately for Gossipsub and for sync recovery, because they answer different questions |
| Index rebuild time | seconds, against event count | Indexer: around the rebuild operation | The project's central claim is that the index is a projection. This says what that costs, and at what volume it stops being cheap |
| Validation throughput | events/second | Node: around the full checklist | The ceiling on what a node can absorb. Measured over the **whole** checklist, not signature verification alone — canonicalization is part of the cost |
| Storage per event | bytes, node and index separately | Node store and index, sampled | No v1 node prunes, so the log only grows. This is what turns "pruning is unnecessary for now" into a dated claim |
| Orphan buffer occupancy | peak entries under controlled disorder | Node: buffer gauge | Says whether the default bound evicts legitimate orphans, which is the difference between a delay and a loss |
| Rejections by reason | count per reason | Node: at each rejection point | Not a headline number, but the one that makes every other measurement debuggable |

Two rules for the numbers to be worth anything. **Distributions, not just means** — a mean propagation latency hides exactly the tail that matters. And **each measurement's method written down as it is taken**, because a number nobody can reproduce is not a result.

The honest limitation to state alongside them: the three nodes run in one environment, so propagation latency measures the protocol and the implementation, not the internet.

**Legal policy** — LGPD, Marco Civil, and the illegal-content process ([mvp.md](./mvp.md#legal-exposure)). Independent of every implementation step, and the section most likely to be asked about in a defence.
