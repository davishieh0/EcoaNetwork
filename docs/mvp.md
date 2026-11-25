# MVP

This describes what a first working version needs to *do*, not what it's built with. Language, frameworks, and databases for the node, gateway, indexer, and client are all open — see [docs/architecture.md](./architecture.md) for the roles these components play without committing to a stack. Only the protocol itself (see [docs/protocol.md](./protocol.md)) is fixed; everything downstream of it can be implemented, and reimplemented, however many different ways someone wants.

## Goal

Prove the architecture works end to end — signed events, real replication, an index that can be rebuilt from scratch — without solving every social and distributed-systems problem at once.

## In scope

- A client (a web frontend is the obvious first one)
- Local identity creation (Ed25519, non-exportable key where the platform supports it)
- Public read access, no signup required
- Profiles, entity metadata, and reviews — all as signed events (note there is no separate "created" event type for profiles or entities; see [protocol.md](./protocol.md#event-types-v1-candidates))
- A gateway API
- A node implementing the protocol
- At least 3 P2P nodes running Gossipsub + basic sync
- Local event storage per node
- An indexer + an index, with basic full-text search
- Entity page (reviews, and this indexer's own summary of the ratings — the protocol defines what a single `rating` means, never how to aggregate them; see [protocol.md](./protocol.md#the-protocol-does-not-define-an-entitys-rating))
- An **inspection view**: the raw signed envelope beside the rendered review, live signature verification in the client, and which nodes currently hold the event
- A **seeded dataset**: entities from public Receita Federal data, plus a generator producing a few hundred properly-signed synthetic reviews across them
- Review replace (edit) and withdraw (logical delete)
- Review responses (company/user replies)
- Basic label-based moderation
- `domain` claim verification end to end (DNS TXT record, self-verifiable, **no verifier service needed at all**) — this is what gives entity edit authority and official company responses a real basis in v1, and `human` via an off-the-shelf CAPTCHA
- Node distributed as something a third party can run without touching source code (e.g. a container image)

## Out of scope for v1

Attachments/IPFS, full DHT, desktop client, mobile app, verified purchase, `cnpj` verification, `email` verification, entity aliases, inventory-based sync reconciliation, social login, social recovery, advanced reputation scoring, dedicated search infrastructure beyond basic full-text, push notifications, indexer federation, anonymous networking, video, token, blockchain.

Several of these were cut deliberately rather than simply never added, and the reasoning is worth keeping:

- **`domain` verification is in scope; `cnpj` and `email` are not.** `domain` is the only self-verifiable claim type — an indexer does a DNS lookup and that's the entire mechanism, no verifier service, no manual review. It's also the one that unlocks tier-1 entity authority and the official-response badge. `cnpj` needs a manual verifier process. `email` needs a verifier service, SMTP, deliverability and templates — real infrastructure with nothing to show for it, when `human` (an off-the-shelf CAPTCHA, roughly an afternoon of work) already separates a fresh key from a slightly less fresh one.
- **Entity aliases are out.** `entity.alias_proposed` stays in the protocol, but nothing in v1 projects it. It carries real complexity in the indexer (grouping two entities and resolving the conflict) against very little that's visible, and it's purely additive — nothing else depends on it.
- **Inventory-based sync reconciliation is out**, replaced by a plain cursor. See [docs/protocol.md](./protocol.md#node-to-node-sync) for why the cursor made it redundant rather than merely deferred.
- **Node pruning is out.** The wire format is specified, but no v1 node prunes; see [docs/protocol.md](./protocol.md#sync-against-a-peer-that-pruned-data).

## Acceptance criteria

The MVP is done when:

- A user can open the client with no signup and create an identity.
- A signed review can be published; the gateway never touches the private key.
- The node validates the signature and rejects invalid events (bad signature, bad ID, bad schema, oversized, unsupported version).
- A valid event is stored on **at least 3 nodes**.
- A node that goes offline can catch up via sync once it's back.
- The indexer processes events and search can find an entity.
- A review can be replaced and withdrawn.
- A moderator can publish a label, and it changes what a client following that moderator sees while changing nothing for a client that doesn't — the label is a projection, not a deletion. *(Below the cut line: the first criterion to drop if the schedule slips.)*
- A third party can start a node without touching its source.
- A brand-new node with no prior known peers can join the network using nothing but the default bootstrap list.
- Taking one node down doesn't stop propagation between the others.
- The index can be dropped and rebuilt purely from node event logs, reproducing the **deterministic** half of the projection exactly — see [architecture.md](./architecture.md#what-must-be-identical-across-indexers-and-what-must-not-be). Re-derived local observations are not expected to match, and the test must not assert that they do: identity age, live `domain` check results, the weights built on them, which entity metadata proposal is canonical, and the revocation cut.
- Two independently-built indexers fed the same event log agree on every item in that deterministic list.
- An event whose reference arrives out of order (a vote before its review) is buffered and accepted once the target lands, rather than dropped.
- A node that has accepted an `identity.key_rotated` rejects a later conflicting one, whatever its `eventId`; a node that receives both at once treats the identity as contested and honors neither successor.
- An event from a different network (`protocol` mismatch) is ignored and never relayed, even though its signature is valid.
- Two independently-built nodes agree on every signature in the ZIP-215 test-vector set, accepting and rejecting exactly the same ones — including the cases where ZIP-215 deliberately *accepts* what a stricter rule set would reject.
- An event whose JSON carries duplicate object keys, or nests deeper than the cap, is rejected before any signature work.
- A review that is edited after being endorsed does not carry its endorsements onto the new version.
- An event with an oversized payload, an unknown top-level envelope field, or a `createdAt` more than 5 minutes in the future is rejected.
- The client can be pointed at a different gateway without code changes.

## Where v1 stands

Checked on 2026-11-24 against this repository, every flow run end to end through [infra/local](../infra/local/README.md): three nodes, the indexer, the gateway behind nginx, and the web client driven in a browser.

| Acceptance criterion | State | Shown by |
|---|---|---|
| Open the client with no signup and create an identity | ✅ | `/app/identity` needs no account; writing does, at this gateway |
| Publish a signed review; the gateway never touches the key | ✅ | the key is non-extractable in IndexedDB; the gateway forwards bytes |
| The node rejects bad signature, ID, schema, size, version | ✅ | `protocol/go` rejection vectors; a tampered event refused (`indexer/cmd/demo tamper`) |
| A valid event on at least 3 nodes | ✅ | the inspection view: "Held by 03 nodes" |
| A node back online catches up through sync | ✅ | node-2 stopped and restarted: it held the event within seconds |
| The indexer processes events and search finds an entity | ✅ | search by name and by category |
| Replace and withdraw a review | ✅ | from the review page, by its author |
| A label changes what a client following its moderator sees, nothing for one that does not | ✅ | `?moderators=`; two browsers, one following the moderator |
| A third party starts a node without touching its source | ⚠️ | a container image builds from the repository; publishing it to a registry is open |
| A brand-new node joins from the default bootstrap list alone | ❌ | the embedded list is empty until a public node exists |
| One node down does not stop propagation | ✅ | publishing with node-1 down reached the other two |
| The index dropped and rebuilt from the logs reproduces the deterministic half | ✅ | `indexer rebuild`: 1712 events, 15 tables identical; and in tests |
| Two independently-built indexers agree on the deterministic list | ⚠️ | this indexer is the only implementation; two instances of it, fed the logs in other orders, agree in its tests (`guarantees_test.go`) |
| A vote before its review is buffered, then accepted | ✅ | the node's orphan tests and fixtures |
| A conflicting later rotation is rejected; both at once contest the identity | ✅ | the node's rotation tests |
| An event from another network is ignored and never relayed | ✅ | the node's network test |
| Two independently-built nodes agree on the ZIP-215 vector set | ⚠️ | the Go node and library pass every vector; a second implementation (`protocol/ts`) is pending |
| Duplicate keys or nesting past the cap are rejected before signature work | ✅ | `protocol/go` envelope tests and vectors |
| An edit after endorsement does not carry the endorsements | ✅ | the review's history: "▲ 1 on this version", the new head at ▲ 0 |
| Oversized payload, unknown envelope field, future `createdAt` rejected | ✅ | rejection vectors |
| The client is pointed at another gateway without code changes | ✅ | `BACKEND_ORIGIN` on the site's nginx: one image, any gateway. Per user at run time is not built ([architecture.md](./architecture.md#substitutability-only-counts-if-the-client-isnt-served-by-the-gateway)) |

Every step of the demonstration below runs: steps 1–4 and 7–8 in the client, step 3's "which nodes hold it" in the inspection view, steps 5 and 6 with `indexer/cmd/demo`, step 9 with `indexer rebuild`, step 10 by restarting the indexer's API with `RANKING=endorsed`. [infra/local/README.md](../infra/local/README.md#the-demonstration) has the commands.

## The demonstration

The acceptance criteria above are a checklist; this is the narrative that makes them legible to someone watching for ten minutes. It's listed here because it changed what's in scope — the inspection view and the seeded dataset exist for this and nothing else, and they're cheap next to the amount of otherwise-invisible work they make visible.

Every step below runs on what's already in scope. Nothing here is an extra feature.

1. **Open the client, create an identity.** No signup, no email, no server round-trip. Show that the private key is generated locally and never appears in any request.
2. **Publish a review.** Show the raw signed envelope next to the rendered review — the canonical bytes, the digest, the signature.
3. **Watch it land on the other two nodes** over Gossipsub, within a second.
4. **Kill one node.** Everything is still readable, and publishing still works. Bring it back and watch it catch up through sync.
5. **Try to publish a tampered event** — flip one byte of the payload, keep the old signature. The node rejects it at step 7 or 8, and never rebroadcasts it.
6. **Try to edit someone else's review with your own key.** Valid signature, valid schema, existing target — and rejected anyway, by the authorship rule ([protocol.md](./protocol.md#who-may-sign-what)). This is the step that shows the design is the project's own and not inherited from a library.
7. **Edit your own review after it was endorsed.** The endorsements don't carry to the new version ([protocol.md](./protocol.md#votes-do-not-follow-an-edit)) — a deliberate anti-abuse decision, made visible.
8. **Verify a domain by publishing a DNS TXT record**, then post a company response that renders with an official badge — with no verifier service anywhere in the loop.
9. **Drop the entire index and rebuild it from the node logs.** Same content, same chains, same votes.
10. **Change the indexer's ranking configuration.** Same events, different order, both correct. This is the project's thesis on screen: the log is the truth, the ranking is an opinion, and anyone can hold a different one — another indexer, run by anyone, is this same choice made by someone else.

Step 10 is the one worth rehearsing. Everything before it demonstrates that the system works; step 10 demonstrates *why it was built this way*, and it's nearly free — one setting, and the API restarted. That a second indexer would agree on the deterministic half is shown by the indexer's tests, not by running one.

## Phases

What each phase contains. For the order to build in, the dependencies between steps, and what blocks what, see [roadmap.md](./roadmap.md).

| Phase | Focus |
|---|---|
| 0 — Spec artifacts | The design is closed; this phase produces artifacts from it. JSON Schema per event type, canonicalization and signing test vectors, **a strict-verification signature test-vector set including the malleability cases**, and fixtures for the rules that are easy to implement subtly differently — chain-head resolution, rotation conflicts, orphan buffering, vote-to-version binding |
| 1 — Node prototype | A node implementing the full protocol: envelope validation (16 KB ceiling, closed envelope, `createdAt` bounds, reordered duplicate check), the orphan buffer, local receive sequence, version-compatibility handling, bootstrap/peer discovery, Gossipsub on the single topic, the two-call sync protocol, local storage — 3 local nodes proving all of it end to end |
| 2 — Gateway + indexer | Publish endpoint, indexer, index, search, rebuild-from-events — with the deterministic/local-observation split ([architecture.md](./architecture.md#what-must-be-identical-across-indexers-and-what-must-not-be)) enforced in tests rather than assumed |
| 3 — Client | Local identity, review form, local signing, search UI, signature verification, static bundle served independently of the gateway, and publishing under a fresh identity as a first-class one-tap option ([identity.md](./identity.md#pseudonymous-is-not-anonymous)) |
| 4a — Demo-critical social | Review replace/withdraw, and review responses. **Not cuttable** — the demonstration turns on editing a review after it was endorsed, and on a company response carrying an official badge |
| 4b — Cuttable social | Profiles, reports, moderation labels and filters. **This is the designated cut line** — if the schedule slips, these can be described as designed-but-not-implemented without weakening the central demonstration |
| 5 — Self-hosting | Distributable node package, node docs, bootstrap list, metrics, status dashboard |
| 6 — Broader decentralization | Multiple gateways/indexers, gateway selection, DHT, entity aliases, `email` and `cnpj` verification, inventory- or Merkle-based sync reconciliation, additional client types, IPFS attachments |

Stack decisions — which language the node is written in, what the gateway/indexer/client are built with — happen when each phase is actually picked up, not here. Phase 0 and 1 are the only ones that matter for closing the base (protocol + node); phases 2 onward are implementation work for whichever gateway/indexer/client eventually gets built, official or not.

## Key risks

| Risk | Mitigation |
|---|---|
| Network is effectively centralized at launch (all nodes run by one operator) | Ship self-hosting early |
| Spam / fake identities | Weighting by claim type rather than a verified/unverified flag ([reputation.md](./reputation.md#eligibility-and-weighting)); per-node admission policy, since gateway rate limits are bypassable by anyone running a node ([identity.md](./identity.md#where-these-controls-actually-sit)); moderation labels |
| Illegal content / PII stored on nodes | **Not adequately mitigated, and the mitigations listed are weaker than the risk.** No attachments in v1 shrinks the surface; labels hide but don't remove; local policies are per-operator. See [Legal exposure](#legal-exposure) |
| Users lose their key, or it's stolen | **Loss:** encrypted export/backup. **Theft:** not solvable for a single-key identity — compromise is terminal, and the docs say so rather than implying otherwise. The root/device key model is the designed answer and the envelope supports it from v1 ([identity.md](./identity.md#root-key-and-device-keys)) |
| A gateway censors an event | Multiple independent gateways, future thicker clients |
| A gateway or node links an identity to an IP address | **Not solved in v1, and stated rather than implied.** Gateways must not log source addresses alongside event content; real network anonymity needs Tor or equivalent and is out of scope. The interface has to say so at publishing time — see [identity.md](./identity.md#and-the-network-layer-knows-more-than-the-protocol-does) |
| Two implementations disagree about whether an event is valid, partitioning the network | ZIP-215 verification, duplicate-key rejection and a nesting cap, all pinned in [protocol.md](./protocol.md#signature-verification-follows-zip-215), plus a shared test-vector set as a Phase 0 deliverable |
| An indexer manipulates results | **Partially mitigated.** Client-side signature verification catches *altered* content but never *omitted* content — an indexer that silently drops results passes every cryptographic check a client can run. Real mitigations are public policies and plural indexers a client can compare; a completeness commitment is an open question, see [architecture.md](./architecture.md#integrity-is-verifiable-completeness-is-not) |
| Low replica count / data availability | Replication policy, community nodes, monitoring |
| Scope creep — protocol + network + client + gateway + moderation is a lot | Small MVP, phased roadmap above, and a designated cut line at Phase 4 |
| **libp2p is the single biggest schedule risk** — it can absorb an entire timeline on its own | The two-call sync protocol keeps libp2p usage to connect, subscribe, publish, and handle one request/response stream, which is manageable. Declared fallback if it still isn't: run v1 over plain WebSocket gossip with a static peer list. The architecture explicitly sanctions this — libp2p is transport, the protocol is meaning ([architecture.md](./architecture.md#why-this-split)) — but it's a plan B, not the starting choice, because a real P2P substrate is part of what the project is claiming |

## Legal exposure

Not solved here, and flagged rather than buried, because it is the risk most likely to end the project and the one the rest of these documents currently answer with a single non-goal line.

A pseudonymous complaints platform about named Brazilian companies runs into three things immutability doesn't get to opt out of:

- **Marco Civil art. 19** conditions an intermediary's immunity on removing content after a court order. The architecture already contains the right answer, and it should be stated instead of implied: **the compliance point is the indexer and the gateway, not the node.** The index is the read path, so removing an event there satisfies an order without rewriting the log — that's the whole reason "moderation is a projection" was worth designing.
- **LGPD** grants a right to erasure that sits in direct tension with a replicated append-only log. Same answer structurally (the data leaves the projection), plus a policy for third-party PII that a reviewer pastes into a review body.
- **Who receives the subpoena.** Whoever holds data linking a pseudonym to a person becomes the natural target of any defamation action, while claiming authority over nothing. **v1 is unusually well placed here**, and by accident rather than design: `email` verification is deferred ([out of scope](#out-of-scope-for-v1)), `domain` is self-verifiable and involves no verifier at all, and `human` is a CAPTCHA that identifies nobody. So v1 holds no linking data — there is nothing to hand over. That changes the moment `email` ships, which is a reason to decide the retention policy before building it rather than after ([docs/verification.md](./verification.md#email--delegated-deferred)).

There's a fourth, quieter one: **third parties who run nodes** (Phase 5) inherit hosting exposure for content they never chose. Shipping a container image without documenting that is passing risk along in silence.

None of this changes the architecture — the compliance surface is already in the right place. What's missing is written policy, and it's a dedicated pass rather than a paragraph.

## Open questions

Still genuinely open:

- ~~Payload schemas per event type~~ — **settled** ([protocol.md](./protocol.md#payloads)). With this the protocol's design is closed; what remains for Phase 0 is producing artifacts from it, not deciding anything.
- **LGPD, Marco Civil, and the illegal-content/PII process** — deferred to a dedicated pass, not because it's minor but because it's the opposite. See [Legal exposure](#legal-exposure).
- Old-version display policy — how much of a replace chain a client shows by default. (The v1 client shows the current version, and the whole history below it on the review's page.)
- Re-check intervals for the delegated claim types — the maximum lifetimes are settled, but `email` and `human` have no cheap re-check, so in practice they just expire ([docs/verification.md](./verification.md#open-questions)).
- Whether an indexer should publish an auditable completeness commitment ([docs/architecture.md](./architecture.md#integrity-is-verifiable-completeness-is-not)).
- ~~Identity export/backup format~~ — **settled for v1**: a backup made when the identity is created, the PKCS #8 key sealed with a password (PBKDF2-SHA-256, 600 000 rounds, AES-GCM bound to the `did:key`), the key kept non-extractable in the browser ([decisions.md](./decisions.md#168-the-identity-is-made-on-purpose-with-its-backup-made-at-that-moment--call)). Which key model v1 ships is settled too: the single key ([decisions.md](./decisions.md#35-v1-ships-the-single-key-model-and-rejects-any-devicekey--call)).
- ~~Which language, framework, and database the gateway, indexer and client use~~ — **settled**: the node in Go with bbolt; the indexer in Go with Postgres; the gateway in Python with FastAPI and Postgres; the client in React with TypeScript ([README](../README.md#repository)).

Settled during the architecture review and no longer open: node retention for v1 (nodes don't prune), minimum replica target (3), alias conflicts (an alias is an ordinary weighted metadata proposal), and the key-recovery strategy (single-key is terminal on compromise; the root/device model is the designed way out). Every decision taken, with its cost and the alternatives rejected, is in [docs/decisions.md](./decisions.md).
