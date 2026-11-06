# Protocol

The protocol is the shared contract every implementation (node, client, indexer, or any third-party alternative) must follow. It's not a running service, and it's not tied to any particular language or stack — it's schemas, canonicalization rules, and signature requirements that anyone can implement independently.

## Event envelope

```json
{
  "protocol": "ecoa",
  "version": 1,
  "eventId": "evt_k3f9x2mq7v...",
  "author": "did:key:z6Mk...",
  "deviceKey": null,
  "eventType": "review.created",
  "createdAt": "2026-08-05T17:00:00Z",
  "payload": { },
  "signature": "base64url..."
}
```

### Field rules

| Field | Rule |
|---|---|
| `protocol` | The network identifier: `ecoa` for the main network, `ecoa-<name>` for any other (`ecoa-testnet`, `ecoa-dev`). Inside the signed bytes, so an event can never be replayed into a different protocol *or a different network* — see [Network separation](#network-separation). |
| `version` | Integer envelope version. See [Versioning & compatibility](#versioning--compatibility). |
| `eventId` | `evt_` + the signing digest, base32 lowercase without padding ([RFC 4648](https://www.rfc-editor.org/rfc/rfc4648) §6). Base32 rather than hex or base58: case-insensitive, alphabet-unambiguous, and safe in a URL or a filename without escaping. |
| `author` | A `did:key` for an Ed25519 public key (multibase `z` + multicodec `0xed01`) — **the identity**, and what every "same author" comparison in this document means. Identities compare as exact strings after that canonical encoding. |
| `deviceKey` | `null`, or a `did:key` for the device key that actually produced `signature`. `null` means the identity signs with its own key. **Always present, never omitted** — under JCS a null-valued key and an absent key serialize differently and therefore hash differently, so an implementation that drops the field produces a different `eventId` for identical content. See [Signing key vs. identity](#signing-key-vs-identity). |
| `eventType` | `namespace.verb`, lowercase, from the registry below. Unrecognized values are structurally valid; see [Event types and payload schemas](#event-types-and-payload-schemas--usually-no-version-bump-at-all). |
| `createdAt` | [RFC 3339](https://www.rfc-editor.org/rfc/rfc3339) UTC with a literal `Z`, second precision, no numeric offsets — `2026-08-05T17:00:00Z`. One spelling per instant keeps a timestamp from being a free source of hash variation. Self-attested; see [What `createdAt` can and cannot be used for](#what-createdat-can-and-cannot-be-used-for). |
| `payload` | A JSON object, per-`eventType` schema. |
| `signature` | Base64url without padding. |

### Canonicalization, event ID, and signature

**Canonicalization (v1):** [JSON Canonicalization Scheme](https://www.rfc-editor.org/rfc/rfc8785) (JCS, RFC 8785) — deterministic property ordering and number formatting, with existing implementations in most general-purpose languages, so no implementation has to write its own. DAG-CBOR is a possible future upgrade for tighter content-addressing, and would be an envelope `version` bump.

**The signing bytes** are the JCS serialization of the **entire envelope minus `eventId` and `signature`**. Both exclusions are load-bearing:

- `eventId` can't be inside its own digest.
- `signature` is excluded so the digest is over content only. A useful side effect: Ed25519 signature malleability is harmless here, because a second valid signature over the same content yields the same `eventId` and is caught as a duplicate rather than becoming a second event.

Everything else — `protocol`, `version`, `author`, `eventType`, `createdAt`, `payload` — is inside the signed bytes. **Not just `payload`**: signing the payload alone would let the same review body be reattributed to a different author or a different timestamp while keeping a valid signature and, worse, would give two different authors publishing identical text the same `eventId`.

```
canonical     = JCS({protocol, version, author, deviceKey, eventType, createdAt, payload})
signing_input = "ecoa-event-v1" || 0x00 || canonical
eventId       = "evt_" + base32_lower_nopad(SHA-256(signing_input))
signature     = Ed25519(signing_private_key, signing_input)
```

Verification recomputes `signing_input` from the received envelope, so a node never trusts the sender's `eventId` or its framing — nothing is trusted just because it arrived from a peer.

**The `ecoa-event-v1` prefix is domain separation**, and the `0x00` byte after it prevents any prefix ambiguity. `protocol` and `version` sitting inside the JSON already make cross-protocol replay hard, but only by accident of content. The prefix makes it structural: a signature produced for an Ecoa event can never be valid for anything else the same key might ever be asked to sign — a login challenge, a session token, a future feature nobody has thought of yet. This costs nothing today and cannot be added later without invalidating every signature ever made.

### Signature verification follows ZIP-215

Ed25519 implementations do **not** all accept the same set of signatures. Cofactored versus cofactorless verification, small-order points, and non-canonical encodings are each handled differently by different libraries. In a network where every node independently decides whether an event is valid, that is not a cryptographic weakness — it is a **partition**: a Go node and a Rust node disagreeing about the same bytes, permanently, over an event neither one was wrong to have processed the way it did.

So verification is pinned to **ZIP-215**, which exists for precisely this problem:

- **Cofactored verification** — check `[8]R = [8]([s]B − [k]A)`, not the unbatched equation.
- **The scalar must be canonical** — reject `S ≥ L`.
- **Non-canonical point encodings are accepted** for `A` and `R`, provided they decode to a curve point at all.
- **Small-order points are not rejected.**

The last two look wrong at a glance, and they are the entire point. ZIP-215 is not the strictest available rule set — it is the one under which every conforming implementation agrees, including with the large body of signatures already produced by permissive implementations. Cofactorless verification does the opposite: its result depends on whether a small-order component happens to be present, which is exactly how two honest nodes end up disagreeing about the same bytes.

**Consensus and binding are different goals, and this design needs only the first.** Rejecting small-order keys buys *signature binding* — the guarantee that a signature cannot be mauled into a second valid one. This project already has that from somewhere else: `signature` sits outside the `eventId` digest, so a mauled signature over the same content yields the same `eventId` and is dropped as a duplicate. What is left to protect is agreement, and agreement is what ZIP-215 provides.

`did:key` decoding is a separate question and stays strict: the multibase string must be canonical, because identities are compared as exact strings and two spellings of one key would fracture every authorship check in the protocol. Bytes that decode to no curve point are rejected there too.

### Signing key vs. identity

`author` says *who* an event is from. `deviceKey` says *which key* signed it. When `deviceKey` is `null` — the simple case, and the only one v1 accepts — they are the same key and `signature` verifies against `author` directly.

**In v1 an event with `deviceKey` set is rejected**, with that as the reason, rather than sent to the [orphan buffer](#unresolved-references): no v1 node processes `identity.device_authorized`, so the authorization would never arrive and the event would never resolve ([decisions.md](./decisions.md#35-v1-ships-the-single-key-model-and-rejects-any-devicekey--call)).

Once the root/device model is adopted, `signature` verifies against the **device key**, and the event is valid only if that device key was authorized by `author` and not since revoked ([docs/identity.md](./identity.md#root-key-and-device-keys)). Because that means a signature check can depend on an `identity.device_authorized` event the node may not have yet, a device-signed event whose authorization hasn't arrived will go to the orphan buffer like any other unresolved reference.

**The field exists in v1 even though v1 may never set it**, and that is the entire reason it's here. The envelope is a closed set (above), so adding a field later is a breaking `version` bump that every node must adopt in lockstep. A nullable field costs one `null` per event now and buys the option to adopt the root/device model — the only real answer to a compromised key — without a protocol migration. This is the one place where paying a small cost up front is clearly better than the alternative, because the alternative is not "add it later" but "fork the envelope later".

Note the consequence for `author` comparisons: they are always against the *identity*, never against the signing key. An identity that swaps device keys every month keeps a single unbroken history, and can still edit or withdraw anything it published from any of them.

**Rules that keep the digest reproducible.** JCS canonicalizes *after* parsing, so everything a JSON parser decides before that point is part of the attack surface:

- **Duplicate object keys are rejected.** Given `{"a":1,"a":2}`, some parsers keep the first value, some the last, and some error — and any two that disagree compute different digests for the same bytes. This is the same class of network partition as loose signature verification, arriving through the parser instead of the crypto.
- **Nesting depth is capped at 32, and object key count is capped per schema.** The 16 KB ceiling bounds size, not structure: thousands of nesting levels fit comfortably inside 16 KB, and a recursive canonicalizer meeting them overflows its stack. That's denial of service that costs the attacker nothing and lands *before* any signature is checked, which is exactly where cheap rejections have to live.
- **Unknown top-level envelope fields are rejected.** The envelope is a closed set. Otherwise a node that drops a field it doesn't recognize computes a different digest than one that keeps it, and the two disagree about whether a valid event is valid. Payloads are the opposite — additive by design (see [Versioning](#event-types-and-payload-schemas--usually-no-version-bump-at-all)) — so a payload's unknown fields must be preserved byte-exactly and included in the digest even by a node that can't interpret them.
- **Text is NFC-normalized before signing.** JCS pins JSON structure but not Unicode: the same accented character composed one way and decomposed another serializes to different bytes and hashes differently. Clients normalize every string in the payload to Unicode NFC before signing; nodes do not re-normalize on receipt, since that would alter the signed bytes. A client that skips this produces events that verify correctly but may duplicate visually identical content — a content problem, not a validity one.

**No floating-point numbers in payloads.** JCS does define a deterministic serialization for them, but the failure mode when an implementation gets it subtly wrong is a signature that verifies nowhere. Ratings ([Ratings](#ratings)), counts, and every other numeric field are integers; anything fractional is expressed in a smaller unit or as a string.

## Validation checklist (every node runs this on receipt)

1. **Size.** Reject above the 16 KB protocol ceiling, or above this node's own stricter limit.
2. **Network and version.** `protocol` matches exactly the network this node is configured for; `version` is one it supports. A mismatch is not an error to report — it's an event from a different network, silently ignored and never relayed.
3. **Envelope shape.** Decode, then check the envelope is a closed set — every required field present, **no unrecognized top-level field**, each field well-formed per [Field rules](#field-rules). Includes the `createdAt` bounds rule: reject anything more than 5 minutes in the future (see [What `createdAt` can and cannot be used for](#what-createdat-can-and-cannot-be-used-for)).
4. **Duplicates.** If this `eventId` is already stored, drop silently and stop here.
5. **Payload schema.** Validate the payload against the schema for this `eventType`, including that every entity identifier it carries is already normalized (see [Identifying entities](#identifying-entities)). *Skipped for an unrecognized `eventType`.*
6. **Canonicalize the signing bytes** — the whole envelope minus `eventId` and `signature`, JCS-serialized.
7. **Recompute the digest** over those bytes and compare it to `eventId`.
8. **Verify the signature** against `deviceKey` when it is set, against `author` when it is `null`. Then reject if this node has already observed an `identity.key_revoked` for the signing key (see [When rotation and revocation take effect](#when-rotation-and-revocation-take-effect)).
9. **Resolve references and check authorship** — that `replaces` and similar targets exist, that the author is entitled to act on them ([Who may sign what](#who-may-sign-what)), and, once the root/device model is adopted, that an `identity.device_authorized` from `author` covers `deviceKey` and no `identity.device_revoked` has retired it (v1 rejects a set `deviceKey`, see [Signing key vs. identity](#signing-key-vs-identity)). Anything unresolvable goes to the orphan buffer rather than being rejected ([Unresolved references](#unresolved-references)). *Type-specific, except the `deviceKey` part.*
10. **Node admission policy** — per-author rate, per-identity thresholds, whatever this operator has configured ([docs/identity.md](./identity.md#where-these-controls-actually-sit)). Local, never protocol-level.
11. **Store**, assigning a local receive timestamp and the next monotonic local sequence number.
12. **Notify the indexer.**
13. **Rebroadcast.**

Steps 1–4 and 6–8 are **universal**: they need no knowledge of what the payload means, which is what lets a node relay an event type it has never heard of ([Event types and payload schemas](#event-types-and-payload-schemas--usually-no-version-bump-at-all)). Steps 5 and 9 are largely **type-specific** and are skipped for an unrecognized `eventType` — such an event is stored and relayed as structurally sound and authentically signed, but semantically opaque. Step 10 is **local policy** and binds nobody but the node applying it.

A node must never rebroadcast before finishing validation. Duplicate detection runs early on purpose — checking a stored `eventId` is cheap, while canonicalizing and verifying a signature isn't. A peer replaying an event a node already validated shouldn't be able to burn its CPU by forcing it to redo that work, or get it rebroadcast again as if it were new.

The `eventId` is attacker-controlled until step 7 recomputes it, so step 4 is a lookup against already-validated stored events and never a trust decision: a hit means "already have this, drop it", a miss simply proceeds to full validation. It makes replay cheap to reject; it does nothing about fresh garbage, which still pays for a full canonicalization and signature check.

## Network separation

`protocol` is not decoration and it is not always the literal string `ecoa`. It names **which network** an event belongs to, and it sits inside the signed bytes.

- `ecoa` — the main network.
- `ecoa-dev` — the development network, and the only one that exists today.
- `ecoa-<name>` — anything else: a private deployment, a classroom, a second test network.

A node is configured for exactly one network and rejects everything else at step 2, before any other work.

**Without this, an event signed on a test network is cryptographically valid on the main one.** Same key, same schema, same signature — nothing distinguishes them, so test data could be lifted into production wholesale, and signatures harvested from a throwaway network would be indistinguishable from real ones. Separate gossip topics keep the two from propagating into each other by accident, but they do nothing to stop deliberate copying, because the events really are valid.

This costs nothing: `protocol` was already a constant string inside the signed bytes, so naming the network in it needs no new envelope field. It only had to be decided *before* v1 ships, since the envelope is a closed set and adding a field afterwards is a breaking version bump for the whole network.

## Maximum event size

A single size limit applies to the whole envelope, for every event type — no per-type table for now, since v1 has no attachments (those are CID references, not embedded bytes) and even a long, venting review comfortably fits in a few KB of text.

- **Protocol ceiling: 16 KB per envelope.** This is a hard limit every node must enforce (validation checklist step 1) — no node may accept an event above it, no matter how permissive its own policy is. It exists to bound worst-case gossip traffic across the whole network: a node still pays the bandwidth cost of receiving an oversized event before it can even check and reject it, so without a shared ceiling, permissive nodes could keep relaying huge payloads across the mesh indefinitely.
- **Node policy: each operator may go stricter, never looser.** A node can choose to accept only, say, 4 KB for its own resource reasons — it just won't store or relay anything above its own threshold, the same way it wouldn't for an unrecognized event type. It can never raise the limit above the protocol's 16 KB.

## Peer abuse & flood protection

This is a different problem from API-level rate limiting (see [docs/identity.md](./identity.md#anti-abuse-sybil-resistance), which covers a single *publisher* spamming events). Here the concern is a misbehaving *peer* on the P2P layer — too many connections, too much traffic, repeated garbage.

No bespoke Ecoa mechanism for this — libp2p already covers the P2P layer well:

- **Gossipsub peer scoring** — penalizes peers that send invalid or low-value messages, deprioritizes them, eventually stops relaying through them.
- **Connection manager limits** — caps how many peers a node holds connections with at once, so no single peer (or coordinated set) can exhaust connection slots.

The one Ecoa-specific mitigation is the reordered validation checklist above — cheap duplicate detection before any expensive crypto work is itself an anti-flood measure, since it stops replay spam from costing meaningful CPU.

Deeper P2P-layer attacks (eclipse attacks, targeted peer-set manipulation) are left to implementation-time hardening rather than a design decision here — mitigated in the broad strokes already noted elsewhere (multiple independent bootstrap nodes, no single node treated as authoritative) but not fully specified.

**What an eclipse can and cannot do is worth stating precisely**, because it bounds the damage sharply. An attacker who controls every peer a node talks to can show that node a *subset* of the network — withholding events, delaying them, presenting a partial history. They **cannot** show it anything forged: every event is validated locally against its own signature and digest, and no peer is ever trusted for having said something is valid. So eclipse is a censorship and availability attack, never an integrity one. A node fed a curated view still cannot be made to accept a review nobody wrote, a claim nobody issued, or an edit by the wrong author.

The practical mitigation is the ordinary one — connect to peers from more than one independent bootstrap source — and v1 cannot offer it, since one operator runs everything ([Bootstrap & peer discovery](#bootstrap--peer-discovery)).

## Versioning & compatibility

Two different things get versioned here: the **envelope** (rare, breaking changes) and **event types** (frequent, additive changes). A third axis — the sync wire protocol — is handled separately by libp2p and needs no design of its own.

### Envelope version — the `version` field

Covers the outermost format: canonicalization scheme, hash algorithm, signature algorithm, and the envelope's required fields. It bumps only when one of *those* changes — never for a new event type or a new optional payload field.

Because events are replicated and kept indefinitely, compatibility is asymmetric by direction:

- **Backward — mandatory, forever.** A node must always retain the ability to validate and serve every envelope version it has ever supported. A `version: 1` event published today has to still verify correctly years from now, even after `version: 2` exists — the past is never renegotiated.
- **Forward — not expected.** A node that only understands `version: 1` isn't expected to validate, store as trusted, or relay `version: 2` events — it may not even be able to canonicalize them correctly if the scheme itself changed. It simply doesn't participate in that version until it upgrades, on the operator's own schedule — no attempt at partial processing. A `version` bump is rare and deliberate, and should require explicit, deliberate agreement rather than happening as a side effect of any single implementation's changes.

### Event types and payload schemas — usually no version bump at all

New event types, and new *optional* fields on an existing type's payload, don't touch the envelope `version`. This is the common case, and it degrades gracefully on nodes that haven't updated their type registry yet, by splitting validation into two tiers:

1. **Universal checks** — apply to every event regardless of `eventType`: size limit, envelope shape, hash recomputation against `eventId`, signature verification. None of these require knowing what the payload means.
2. **Type-specific checks** — schema validation of the payload, required references, type-specific anti-abuse rules. These require the node to recognize `eventType`.

A node that passes tier 1 but doesn't recognize the `eventType` still stores and relays the event — it just can't apply type-specific rules or hand it to an indexer meaningfully. This doesn't break "never rebroadcast before validating": what changes is what *validated* means for an unrecognized type — structurally sound and authentically signed, semantically opaque. That's enough to keep the event propagating through nodes that haven't upgraded, without those nodes vouching for content they can't interpret. The network keeps flowing instead of stalling at the slowest-upgraded node.

Removing a field, or changing what an existing field means, is **not allowed** in place — that requires a new event type (e.g. `review.created_v2`), never a silent redefinition of an existing payload shape.

### Sync wire versions

Already handled by libp2p protocol negotiation, not something to design here. Protocol IDs like `/ecoa/sync/1.0.0` let two peers negotiate the highest version they both support when a stream opens; a node just needs to keep old handlers registered during a migration window instead of removing them immediately. See [docs/architecture.md](./architecture.md#node-communication-patterns).

This section covers version mismatches, not retention — a peer that understands a version but has since pruned that data is a separate case, handled in [Sync against a peer that pruned data](#sync-against-a-peer-that-pruned-data) below.

## Event types (v1 candidates)

| Category | Types |
|---|---|
| Identity/profile | `profile.updated`, `identity.key_rotated`, `identity.key_revoked`, `identity.device_authorized`, `identity.device_revoked` |
| Entities | `entity.updated`, `entity.alias_proposed` |
| Reviews | `review.created`, `review.replaced`, `review.withdrawn`, `review.response_created` |
| Moderation | `moderation.label_created`, `moderation.label_revoked`, `moderation.report_created` |
| Reputation | `review.endorsed`, `review.disputed` |
| Verification | `verification.claim_created`, `verification.claim_revoked` |

`verification.claim_created` is generic on purpose: it carries a `claimType` field (`email`, `purchase`, `domain`, `cnpj`, `human`, ...) plus whatever evidence that claim type needs. It is signed by a verifier for delegated claim types and by the subject itself for self-verifiable ones — [docs/verification.md](./verification.md#two-kinds-of-claim-two-different-defenses) covers which is which and why the difference decides what a consumer has to trust. This is how email confirmation, purchase confirmation, and company/domain control all reuse the same event type instead of each needing its own. See [docs/identity.md](./identity.md#claims-not-tiers) for how clients and indexers interpret these claims into roles.

There is deliberately **no `profile.created` or `entity.created`**, and no `entity.alias_confirmed`. A "created" event would be indistinguishable from the first "updated" one — same payload, same author, same effect — and having both invites the question of which outranks the other, which is an ambiguity with no upside. Profiles and entity metadata are last-writer-wins records; the first write is not a special kind of write. `entity.alias_confirmed` goes for a different reason: it implies an authority to confirm, and no such authority exists. An alias is a proposal like any other entity metadata, resolved by the same weighting in [docs/entities.md](./entities.md#2-unverified-edit--a-proposal-not-a-fact) — a verified representative's proposal already carries tier-1 weight, which is exactly what "confirmed" was reaching for.

`review.endorsed` / `review.disputed` are the community voting primitives — a signed opinion on an existing review, separate from moderation labels. See [docs/reputation.md](./reputation.md) for the vote payload, weighting, and how it feeds into ranking.

`moderation.report_created` and `moderation.label_created` are two different acts, not two names for one. A **report** is a request, publishable by anyone: *"I think this is spam."* A **label** is a verdict, published by an identity acting as a moderator, and it's the one clients and indexers actually act on. Reports feed a moderator's queue; labels feed presentation. Neither is privileged by the protocol — a label matters only to a consumer that has chosen to follow the identity that signed it.

Not all of these need to ship in the MVP — see [mvp.md](./mvp.md).

## Who may sign what

A valid signature proves *a* key authored an event. For most event types that's the whole question — anyone may publish a profile, a review, a vote, a report, and the signature settles it. But some events **act on an event that already exists**, and there a valid signature isn't enough: the event must also come from the identity entitled to act on that target.

The rule is universal rather than per-type:

> **An event that replaces, withdraws, or revokes another event must carry the same `author` as its target.**

An event that merely *references* another — a response, a vote, a report, a moderation label — is a new, independent statement by its own author, and carries no such restriction.

| Event | Must be signed by | Why |
|---|---|---|
| `review.replaced` | the author of the event named in `replaces` | editing someone else's statement is forgery |
| `review.withdrawn` | the author of the targeted review | so is deleting it |
| `profile.updated` | the identity the profile describes | a profile is a statement about self |
| `identity.key_rotated` | the key being rotated away from | only the old key can authorize its own succession |
| `identity.key_revoked` | the key being revoked | same reason, without naming an heir |
| `identity.device_authorized` / `identity.device_revoked` | the root identity key | the whole point of a cold root is that a stolen device key cannot authorize another one |
| `moderation.label_revoked` | the author of the label being revoked | a moderator retracts their own verdict, nobody else's |
| `verification.claim_revoked` | the issuer of the original claim | a verifier retracts their own word — see [docs/verification.md](./verification.md#revocation) |
| `review.response_created` | anyone | a reply is a new statement, not a change to the original |
| `review.endorsed` / `review.disputed` | anyone | a vote is the voter's own opinion |
| `moderation.label_created` / `moderation.report_created` | anyone | authority comes from who follows the signer, never from the protocol |
| `entity.updated` / `entity.alias_proposed` | anyone | an entity has no author; authority over its metadata is a separate problem — see [docs/entities.md](./entities.md) |

Two consequences worth stating outright:

- **"Same author" follows the rotation chain.** An identity that rotated its key must still be able to edit and withdraw what it published under the old one, so the comparison is between *identities resolved through `identity.key_rotated`*, not between raw key bytes. An identity has at most one successor ever, and the first rotation a node observes is the one that holds — a later conflicting one is rejected rather than allowed to displace it retroactively. A node that sees both at once treats the identity as contested and honors neither. See [docs/identity.md](./identity.md#key-rotation).
- **The check needs the target.** Like every reference check in step 9, it can only run on a node that actually holds the target event. A node that pruned the target keeps enough to still answer the question: the tombstone retains the target's `author` alongside its `eventId` — see [Sync against a peer that pruned data](#sync-against-a-peer-that-pruned-data).

## What `createdAt` can and cannot be used for

`createdAt` is written and signed by the author. Nothing about a valid signature makes it *true* — an identity can stamp any past date it likes on its first event. This matters because identity age is one of the two inputs to vote weight ([docs/reputation.md](./reputation.md#eligibility-and-weighting)), so a forgeable clock would be a forgeable Sybil defense.

There is no trustworthy global clock available here, and inventing one would mean electing timestamp authorities — a centralization the rest of the design spends its effort avoiding. So the design accepts the limit and splits time into two things that are used for different jobs.

**1. `createdAt` — signed, global, and only trusted about the author's own events.**

- A node **rejects** any event whose `createdAt` is more than **5 minutes in the future** relative to its own clock. This kills post-dating outright, and 5 minutes is loose enough to absorb ordinary unsynchronized-clock drift. Note the dependency it creates: a node whose own clock is badly wrong rejects healthy traffic, so keeping it synchronized is an operational requirement rather than a nicety.
- Backdating is *not* rejected, because a node has no way to disprove it.
- `createdAt` is authoritative for **ordering an author's own events against each other** — which `profile.updated` is current, which link of a replace chain follows which. Lying there only reorders your own content, so there's no incentive and no cross-identity damage. **Ties break on the lower `eventId`**, making the order a deterministic total order that every implementation reproduces identically from the same set of events.
- `createdAt` is **never** authoritative for comparing *different* identities — not for identity age, not for "which claim came first", not for vote recency.

**2. `receivedAt` — unsigned, local, and never replicated as fact.**

Every node records, alongside each stored event, when it first accepted the event and a monotonically increasing local sequence number. This is **local observation**, not protocol truth: it lives in the node's own store, it is not part of the signed bytes, it is not gossiped as authoritative, and two nodes will legitimately hold different values for the same event.

Identity age is then computed as `max(createdAt, earliest local receipt for that identity)`. Backdating buys nothing, because the `max` picks the observation.

**The honest limitation:** a node that just joined and back-filled three years of history has `receivedAt ≈ now` for everyone, so it cannot weight by age at all until it has been running for a while. It has two options, and no third: inherit a starting view from a longer-running indexer it explicitly chooses to trust, or set the age multiplier to a constant and let `claim_weight` carry the whole defense. Taking `receivedAt` hints from sync peers is **not** an option — the hint is forgeable in exactly the direction an attacker wants (looking older than you are).

This is why [docs/reputation.md](./reputation.md#eligibility-and-weighting) puts most of the weight on claims, which have an issuer and replay identically, rather than on age, which does not.

### When rotation and revocation take effect

Self-scoped ordering has one adversarial exception, and it's worth naming because it looks like a contradiction. The rule above says an author's own clock orders their own events, on the reasoning that lying only hurts yourself. That reasoning fails for `identity.key_rotated` and `identity.key_revoked` — because in the scenario those exist for, the author's clock is in the *attacker's* hands. A thief who steals a key and sees it revoked can simply publish new events backdated to before the revocation, and a timeline cut based on `createdAt` would let all of them through.

So **a rotation or a revocation takes effect at the point the node locally observed it**, not at the `createdAt` it claims. Once a node has accepted a revocation for a key, it stops accepting anything new signed by that key, regardless of what date those events carry. Events accepted *before* that observation stay valid; a revocation is a full stop going forward, never a retroactive erasure of history.

The same applies to `identity.device_revoked`: a device key stops being able to sign at the moment the node observes its revocation, not at whatever timestamp the revocation claims. And it applies to rotation in the strongest form — **an accepted rotation is never displaced by one that arrives later**, which is what stops a leaked old key from reaching back past months of accepted history to seize an identity ([docs/identity.md](./identity.md#key-rotation)).

This makes the effective cut a local observation, so two nodes can differ by however long the revocation took to reach each of them. That's the correct trade: the alternative is a cut the attacker gets to place themselves.

### The general rule

> **Ordering within one author is deterministic and uses that author's own clock. Ordering across authors — and any rule an attacker holding the author's key would want to move — is a local observation, and implementations may legitimately disagree.**

Everywhere a document says "the latest" — the current profile, the head of a replace chain, an identity's current vote on a target — it means the first rule. Everywhere it involves comparing identities, it means the second, and the result is explicitly per-indexer rather than a protocol fact. See [docs/architecture.md](./architecture.md#what-must-be-identical-across-indexers-and-what-must-not-be).

## Unresolved references

Gossip does not deliver in causal order. A `review.endorsed` routinely arrives before the `review.created` it targets, and a node that was offline gets history in whatever order its peers hand it over. Step 9 requires the target to resolve, so a naive implementation would reject a perfectly valid event for the sole reason that it arrived a few hundred milliseconds early — and, because rejection means no rebroadcast, would drop it out of the network rather than merely delaying it.

So a failed reference check is **not** the same as an invalid event:

- An event that passes every check through step 8 (structurally sound, hash correct, signature valid) but whose reference does not resolve goes into a bounded **orphan buffer** instead of being rejected. It is not stored as valid, not indexed, and **not rebroadcast**.
- An event whose `eventType` the node doesn't recognize is **not** an orphan. It has no reference check to fail, because reference checks are type-specific ([Event types and payload schemas](#event-types-and-payload-schemas--usually-no-version-bump-at-all)) — it passes the universal checks, gets stored and relayed, and never enters the buffer.
- When a newly accepted event completes a pending reference, the buffered event is re-run from step 9 and, if it passes, stored and rebroadcast normally.
- The buffer is bounded in both entries and age — a suggested v1 default of **1 hour or 10,000 events, whichever comes first** — and evicts oldest-first. Eviction is silent and safe: a genuinely valid orphan comes back through ordinary sync once its target is in place, because the peer that has the target has the child too.
- An event that is still an orphan after eviction is simply gone from this node's view until then. That is a delay, not a loss, and it's the reason the buffer can stay small.
- **The buffer is itself an attack surface**, since filling it with valid-signature events pointing at targets that will never exist is cheap and would evict legitimate orphans. So the bound is applied *per peer*, not just globally: no single peer may occupy more than a small share of the buffer, and a peer producing orphans that never resolve is exactly the behavior Gossipsub peer scoring already penalizes ([Peer abuse & flood protection](#peer-abuse--flood-protection)).

Without this, ordinary out-of-order delivery would be indistinguishable from an attack, and the network would lose valid events at a rate proportional to how fast it is.

## Editing and deleting

Nothing is mutated in place.

- **Edit:** publish a `review.replaced` event whose `replaces` names the **immediately preceding version**, not the original. Successive edits form a chain.
- **Delete:** publish a `review.withdrawn` event targeting any link in the chain. It withdraws the **whole chain**, not one version — otherwise withdrawing would only ever hide one link while the rest stayed readable. Clients and indexers hide a withdrawn chain by default.
- **Limitation:** because events are replicated, there's no way to guarantee physical deletion from every node that ever stored a copy. The UI must make this clear *before* publishing, not after.

### Resolving the head of a chain

`replaces` names a predecessor rather than the original so that a chain is a chain, with no ambiguity about which of five edits is current. Two edits can still name the same predecessor — a fork, usually from one client publishing while offline and unaware of the other. The head is found by walking forward from the original and, at each step, taking the winning successor under the ordering rule from [What `createdAt` can and cannot be used for](#what-createdat-can-and-cannot-be-used-for): higher `createdAt`, ties broken by lower `eventId`. Every link is by the same author (the authorship rule guarantees it), so this is self-scoped ordering and every implementation resolves the same head from the same events.

Implementations cap the walk — **64 links** is the suggested v1 ceiling — so a pathological chain can't turn a page render into unbounded work. Beyond the cap, the projection stops and reports the chain as truncated instead of guessing.

### Votes do not follow an edit

A review can be endorsed, then replaced with different text. Left alone, that is a clean **bait-and-switch**: publish something agreeable, collect endorsements, edit the body, keep the borrowed credibility. The chain makes the swap *detectable* — the original is still there, signed — but detectable isn't prevented, and nobody audits.

So: **a `review.endorsed` / `review.disputed` binds to the exact `eventId` it targeted.** The head's displayed score counts only votes cast on the head. Votes on superseded versions stay visible as part of the chain's history ("14 endorsements on an earlier version") and never contribute to the current version's ranking.

The cost is real and worth naming: fixing a typo resets a review's score, and authors will feel that. It's the deliberate trade — the rule fails toward *unearned score is not displayed* rather than *an attacker keeps credit for text nobody endorsed*, and for a review platform that's the right direction to fail in. It also puts a healthy price on casually rewriting a published review.

This is a **projection rule**, not a wire-format one. The vote event already records which version it targeted, so a future version can soften the policy — carry votes forward with a decay, say — without a protocol change or a migration.

### Labels and responses do follow an edit

Moderation labels and responses attach to the **chain**, not to one version — the exact opposite of votes, and the asymmetry is the point.

- A **label** is a warning. If it bound to a version, clearing a spam label would be as easy as publishing a one-character edit, and moderation would be defeated by the cheapest possible action. Labels therefore carry forward to every later version of the chain. A moderator who thinks an edit fixed the problem revokes the label deliberately (`moderation.label_revoked`), which is a decision by a person rather than a side effect of the author's own edit.
- A **response** is a conversation. A company replying to a complaint is replying to the complaint as a thing, not to one revision of its wording, so responses stay attached across edits. A client showing a response should still make it visible when the text it replied to has since changed — the chain has both versions, so it costs nothing to say so.

The general shape: **credit binds to the version that earned it; warnings and conversation bind to the thing.** Anything the author can gain by editing must not survive the edit; anything the author could escape by editing must.

### Conflicts of interest are visible, and indexers should surface them

An identity holding a recognized `domain` or `cnpj` claim for an entity can also publish reviews of that entity, and vote on other people's. Nothing in the protocol prevents it, and nothing should — a signed event is a signed event, and any rule blocking it would need an authority to enforce it.

But the conflict is *mechanically detectable*: the claim and the review are both public, signed, and reference the same entity. So an indexer can and should mark it — a review of an entity by an identity verified as representing that entity, or a vote from one, is displayed as such rather than counted silently. This is the cheapest possible defense against the most obvious abuse of verification, it requires no new event type, and it's a projection rule, so different indexers can range from a small badge to excluding such votes from the score entirely.

## Payloads

Every payload, and the conventions they all share. `rating` gets its own section below because its *meaning* has to be uniform network-wide; the identifier formats get one for the same reason.

### Conventions

**Nothing in a payload duplicates the envelope.** `author`, `createdAt` and `eventType` are already there. Repeating them creates the question of which one wins when they disagree, and the right answer is to never have the question.

**Optional fields are omitted, never `null`.** The envelope's `deviceKey` is the single exception in the project, and only because the envelope is a closed set.

**Payload schemas validate what they know and permit what they don't.** This is what the versioning rules already require: a new optional field doesn't bump the envelope version, and a node that doesn't recognize it must still preserve it byte-exactly for the digest ([Event types and payload schemas](#event-types-and-payload-schemas--usually-no-version-bump-at-all)).

**Every string is NFC-normalized, trimmed, and free of control characters**, with its own length limit in the tables below. Without per-field limits the 16 KB envelope ceiling is the only bound, which means one field can consume the whole event.

Control characters have one exception, and it is worth naming because a schema has to encode it: a **multi-line** field — `body`, `bio`, `description`, `note` — permits the newline `U+000A` and nothing else. A **single-line** field — `title`, `name`, `displayName`, `label` — permits no control character at all. A review body legitimately contains line breaks; a company name does not.

**Replacement is total, not a patch** — `review.replaced` and `profile.updated` carry the complete new content, and an omitted field is cleared rather than left alone. Patches need merge semantics and an ordering to merge along; total replacement is unambiguous, and the chain already preserves every earlier version.

`entity.updated` is the deliberate exception: it is resolved **per field** ([docs/entities.md](./entities.md#2-unverified-edit--a-proposal-not-a-fact)), so it has to distinguish a field the author actually asserted from one they said nothing about. Under total replacement every proposal would implicitly assert *"and everything else should be empty."*

### Field types

| Notation | Means |
|---|---|
| `evt` | An `evt_`-prefixed event id, well-formed base32 |
| `did` | A `did:key` for an Ed25519 key |
| `id` | An entity identifier, already normalized ([Identifying entities](#identifying-entities)) |
| `str(n)` | UTF-8 string, at most `n` characters after NFC normalization and trimming |
| `?` | Optional — omitted when absent |

### Reviews

| Type | Payload |
|---|---|
| `review.created` | `entity` id, `rating` 1–5, `body` str(5000), `title?` str(120), `lang?` BCP 47 |
| `review.replaced` | `replaces` evt, `rating` 1–5, `body` str(5000), `title?` str(120), `lang?` |
| `review.withdrawn` | `target` evt, `reason?` enum |
| `review.response_created` | `target` evt, `body` str(5000), `lang?` |

```json
{
  "entity": "cnpj:12345678000199",
  "rating": 2,
  "title": "Charged for three months after cancelling",
  "body": "...",
  "lang": "pt-BR"
}
```

**`entity` is absent from `review.replaced`.** A review's subject is fixed by the root of its chain and is immutable — moving a published review to a different company isn't an edit, it's a different act, and one worth making impossible rather than merely discouraged.

**`review.withdrawn.reason` is an enum**, not free text: `mistake`, `resolved`, `duplicate`, `privacy`, `other`. Free text here would be permanent, replicated user content inside the one event whose purpose is to *retract* content — a personal-data surface in exactly the wrong place. An author who wants to explain can publish a response.

**`review.response_created.target` may be a review or another response.** It costs nothing and gives one layer of conversation, which on a complaints platform is the interesting case. Indexers cap nesting depth when rendering.

**`lang`** is a BCP 47 tag and exists for one concrete reason: full-text search stems Portuguese and English differently, and automatic detection is wrong often enough to matter on short text. It's optional, and an indexer that ignores it loses only search quality.

### Entities

| Type | Payload |
|---|---|
| `entity.updated` | `entity` id, `name?` str(120), `category?` enum, `description?` str(1000), `descriptor?` object |
| `entity.alias_proposed` | `entity` id, `alias` id |

**`descriptor` is required when `entity` starts with `entity:`** and forbidden otherwise. It's what makes the derived hash reproducible, and its absence would leave the identifier indistinguishable from a random one ([Identifying entities](#identifying-entities)). Its shape is fixed: `name` str(120), `kind` (`business`, `place`, `product`, `service`, `other`), `country` (ISO 3166-1 alpha-2), `locality?` str(120).

The consequence for clients: creating an entity that has no public identifier means publishing `entity.updated` alongside the first review, not instead of it.

**In `entity.alias_proposed`, the lexicographically smaller identifier goes in `entity`.** Without that rule the same pair produces two different events that nothing can deduplicate.

**`category` is a fixed vocabulary**, flat, with no hierarchy in v1:

`retail`, `food-and-drink`, `financial-services`, `telecom`, `utilities`, `transport`, `travel-and-lodging`, `health`, `education`, `government`, `real-estate`, `professional-services`, `entertainment`, `technology`, `manufacturing`, `nonprofit`, `other`

A free string means everyone invents their own and filtering by category stops working; a full taxonomy is a project of its own. A short controlled list is the middle, and it stays extensible because a new value is additive — an older indexer simply doesn't recognize it.

### Identity

| Type | Payload |
|---|---|
| `profile.updated` | `displayName?` str(60), `bio?` str(500), `lang?` |
| `identity.key_rotated` | `successor` did |
| `identity.key_revoked` | `reason?` enum |
| `identity.device_authorized` | `device` did, `label?` str(40) |
| `identity.device_revoked` | `device` did |

`identity.key_rotated` is one field: the old identity is the `author`, so naming it again would be the duplication the conventions above forbid.

`identity.key_revoked.reason`: `compromised`, `lost`, `retired`, `other`. An empty payload is valid.

**No avatar and no links on a profile.** Attachments are out of scope for v1, and an unverified link field on a public profile is an invitation to phishing that costs nothing to leave out.

`device.label` is optional and worth a warning in the client: device names leak more than people expect, and the event is permanent.

### Moderation and votes

| Type | Payload |
|---|---|
| `moderation.label_created` | `target` evt, `label` enum, `note?` str(280) |
| `moderation.label_revoked` | `target` evt — the label event |
| `moderation.report_created` | `target` evt, `reason` enum, `note?` str(280) |
| `review.endorsed` / `review.disputed` | `target` evt, `reason?` str(280) |

Labels and reports share one vocabulary: `spam`, `off-topic`, `illegal`, `personal-data`, `impersonation`, `harassment`, `other`. It has to be controlled — with free-text labels, two moderator lists are incomparable and "following a moderator" stops meaning anything a client can act on.

**There is no action field.** A label says *what something is*, never *what to do about it*. Hiding, down-ranking, or ignoring is the consumer's decision, which is the same rule the rest of the project follows and the reason moderation can be pluralistic at all.

### Verification

| Type | Payload |
|---|---|
| `verification.claim_created` | `claimType` enum, `subject` did, `entity?` id, `evidenceRef?` str(200), `expiresAt` |
| `verification.claim_revoked` | `target` evt, `reason?` enum |

`claimType`: `domain`, `cnpj`, `email`, `human`, `purchase`. `entity` is present for `domain` and `cnpj` and absent otherwise. `evidenceRef` is present only for self-verifiable claims — for delegated ones the evidence is private by construction, and a locator would do nothing but advertise that it exists.

**`expiresAt` is required**, with a maximum lifetime per type: **one year** for `domain`, since a domain changes hands and control is a lease rather than a property; **two years** for the rest. `verification.claim_revoked.reason`: `superseded`, `no-longer-valid`, `issued-in-error`, `other`.

## Ratings

Payload schemas are otherwise out of scope for this document, but a rating is a number whose *meaning* has to be identical on every implementation, the same way an entity identifier does. A client that gets it backwards produces events that are perfectly valid, propagate normally, and mean the opposite of what they say. So this much is protocol, and the rest deliberately isn't.

**`review.created` carries a required `rating`: an integer from 1 to 5, where 1 is the worst and 5 is the best.**

- **Integer**, because the envelope forbids floats in payloads (see [Canonicalization, event ID, and signature](#canonicalization-event-id-and-signature)). No half-stars.
- **1 to 5.** Beyond roughly seven points people stop discriminating consistently, so extra granularity adds noise rather than signal, and a wider range carries much more cross-cultural anchoring variance — "7 out of 10" travels far less well than "4 stars".
- **The direction is normative**, not a display convention. This is the whole reason the field is specified here at all.
- **Required.** An optional rating would put two structurally different populations of review on the network — rated and unrated — with nothing distinguishing them and no way for a consumer to tell a deliberate abstention from a client that never asked.
- **There is no `scale` field.** It looks future-proof and does the opposite: it invites a client to emit `scale: 10` and leaves the network with two incomparable scales in circulation. If a different scale is ever genuinely needed, the versioning rules already say what to do — a new event type, never a silent redefinition (see [Event types and payload schemas](#event-types-and-payload-schemas--usually-no-version-bump-at-all)).

`rating` appears in `review.created` and nowhere else. A response carries no rating, and neither does a vote.

**Not to be confused with a vote.** A rating is the author's verdict on the *entity*. A `review.endorsed` / `review.disputed` is someone else's verdict on the *review* ([docs/reputation.md](./reputation.md)). A one-star review and a disputed review are unrelated signals, and an implementation that blends them is answering a question nobody asked.

### The protocol does not define an entity's rating

There is no canonical average, and the omission is deliberate. The protocol stores one signed integer per review and stops there.

How those integers become a number on a page — mean, median, distribution, weighted by the author's reputation or not, which reviews count and which are excluded — is a projection, decided by whoever runs an indexer, exactly like ranking and reputation ([docs/architecture.md](./architecture.md#what-must-be-identical-across-indexers-and-what-must-not-be)). Two honest indexers can show different numbers for the same entity from the same events, and that is the design working rather than a bug.

Stating this matters because the absence of a rule is easy to misread as an oversight, and a consumer that assumes an official score will build on something the network never promised.

## Identifying entities

Reviews reference an entity by external identifier when one exists, to avoid duplicate/ambiguous entities like "Joe's Market" vs "Joe's Mini Mart":

| Type | Format | Normalization — mandatory before use |
|---|---|---|
| Brazilian company | `cnpj:<14 digits>` | digits only; punctuation stripped; left-padded to 14 |
| Product | `gtin:<14 digits>` | left-padded to GTIN-14; this is what makes GTIN-8, -12 and -13 for the same product resolve to one identifier |
| Website | `domain:<domain>` | lowercase; IDN as A-label (punycode); no trailing dot; no scheme, port, path, or `www.` prefix |
| Physical place | `place:osm:<n\|w\|r><id>` | OpenStreetMap node/way/relation; lowercase type letter, decimal id, no leading zeros — e.g. `place:osm:n240109189` |
| No public ID available | `entity:<hash>` | see below |

**Identifiers are compared as exact strings, so normalization is part of the protocol and not a client courtesy.** `cnpj:12.345.678/0001-99` and `cnpj:12345678000199` are the same company and must not become two entities, and a GTIN-13 with an implicit leading zero must not split a product's reviews in half. A node validates the shape of an identifier it recognizes and rejects an unnormalized one, rather than silently normalizing it — normalizing on receipt would change the signed bytes and invalidate the signature, so the only place it can happen is in the client, before signing.

`place:` is pinned to OpenStreetMap deliberately. Google and Foursquare place IDs are proprietary, can't be independently resolved without an API key, and would make a core identifier of this protocol depend on a company's terms of service. OSM ids are open data and dereferenceable by anyone. A place with no OSM entry uses `entity:<hash>`.

`entity:<hash>` is not a random client-side id. It is `entity:` + base32 lowercase, unpadded, of `SHA-256(JCS(descriptor))`, where the descriptor is a small fixed object — name, kind, country, locality — with every string trimmed, lowercased, whitespace-collapsed, and NFC-normalized before hashing. A random id would guarantee a fresh duplicate every time anyone described the same corner bakery; a derived one makes identical descriptions converge on the same identifier for free, and leaves only genuinely different descriptions to be reconciled.

**The descriptor has to be published for the derivation to be worth anything.** A hash nobody can reproduce is indistinguishable from a random id, so the first `entity.updated` for an `entity:` identifier carries the descriptor it was derived from, and any consumer can recompute the hash and confirm the identifier wasn't just made up. An `entity:` identifier whose descriptor never appears still works — reviews pointing at it group correctly — it simply can't be verified as derived, and an indexer may treat it with the suspicion any unverifiable identifier deserves.

Reconciling those is what `entity.alias_proposed` is for: it asserts that two identifiers name the same thing, and is weighted exactly like any other entity metadata proposal ([docs/entities.md](./entities.md#2-unverified-edit--a-proposal-not-a-fact)). Indexers decide whether to display the two grouped. **A review's entity reference is never rewritten** — an alias is a projection-time grouping, not a mutation of the events that pointed at either side.

Creating an entity needs no permission — the identifier itself is enough, and `entity.updated` only ever adds optional descriptive metadata. Editing it is the harder problem: see [docs/entities.md](./entities.md) for who has authority over `entity.updated` and how conflicting edits get resolved.

## Bootstrap & peer discovery

Before a node can sync or gossip anything, it needs to find at least one peer already on the network. This matters at the base level, not just as an operational detail: without some shared convention, independently-built implementations (a Go node and a Python node, say) have no way to ever find each other on their own.

- **Address format:** standard libp2p multiaddrs (`/ip4/1.2.3.4/tcp/4001/p2p/<peerId>`) — no new format invented here.
- **Where the list comes from, layered:**
  1. A small default list shipped with the node software, for zero-config bootstrapping — the same idea as seed nodes in Bitcoin or IPFS.
  2. The operator can override or add to it — their own peers, a private network, whatever they want.
  3. *(Later, not required for v1)* a DNS-based seed, so the default list can be refreshed without shipping a new version of the node. This is a discovery convenience, not a trust escalation — every event from a peer found this way is still independently verified like any other, so a compromised DNS entry could only steer a node toward bad peers, never make it accept forged data.
- **After the first connection**, ordinary Gossipsub mesh membership and peer exchange handle finding more peers. No DHT in v1 — consistent with the rest of the MVP scope (see [docs/mvp.md](./mvp.md)).
- **A node that has run before** should try the peers it knew from its last session before falling back to the bootstrap list — the network shouldn't depend on bootstrap nodes staying up forever.
- **Being in the bootstrap list grants no authority.** It's purely a discovery convenience, same as every other node — consistent with no node being authoritative. Relying on bootstrap sources from more than one independent operator avoids this becoming a single point of dependency.
- **In v1 there is only one operator**, so that last point is an aspiration rather than a property. Every bootstrap node is run by the project, and a bad default list would steer every new node at once. What v1 can do is make the list overridable from configuration on day one (layer 2 above) so the fix requires no new software, and treat "a second operator runs a bootstrap node" as the first concrete milestone of decentralization rather than a later nicety — it is cheaper than any other item on that path and it removes the most complete single point of dependency in the design.

## Node-to-node sync

Two mechanisms, not one:

- **Gossipsub** — real-time propagation of new events on a **single topic, `/ecoa/events/1`**, carrying every event type.

  Splitting by category (`/ecoa/moderation/1`, `/ecoa/entities/1`, …) is the obvious-looking optimization and is deliberately rejected for v1. A node subscribed to some topics but not others holds a partial view, which quietly breaks two things the design depends on: an index can no longer be rebuilt completely from one node's log, and reference checks (step 9) fail against events that exist on the network but not on this node — turning ordinary partial subscription into a permanent orphan factory. One topic means every node's log is complete by construction. If per-topic subscription is ever genuinely needed for scale, it can be added as a new topic set alongside this one, with `/ecoa/events/1` remaining the complete firehose.
- **Sync protocol** — direct libp2p streams (Protobuf) for a node that was offline to catch up. Two calls, and deliberately no more than two:
  - `GetEventsSince(cursor)` — stream me everything you accepted after this position. This is catch-up.
  - `GetEvents(ids)` — give me these specific events. This is for filling a known hole, such as the target of an event sitting in the [orphan buffer](#unresolved-references).

**The cursor is the peer's local receive sequence, never `createdAt`.** A cursor over the authors' own timestamps is broken by construction: `createdAt` is self-attested, so an event stamped three years ago and published today would fall behind any cursor a peer had already advanced past, and would never be delivered to anyone syncing incrementally — silently, and forever. Each node assigns every accepted event a monotonic local sequence number ([What `createdAt` can and cannot be used for](#what-createdat-can-and-cannot-be-used-for)), and a cursor is a position in *that* peer's sequence. Cursors are peer-specific and not portable: a node tracks one per peer, and starts from zero with a peer it has never talked to.

**No inventory exchange, and no set reconciliation.** An earlier version of this had the syncing node request an inventory of event IDs, diff it against local storage, and fetch only the difference. Having a per-peer cursor makes that redundant: inventory diffing exists precisely for the case where you *don't* know where you left off. What it actually buys is bandwidth when re-syncing against a peer you've never met, since you re-receive events you already hold from someone else — and duplicate detection (step 4) discards those for the cost of a hash lookup. At any volume this network will see for a long time, that is a few megabytes and no correctness difference, in exchange for a materially simpler protocol with fewer places to disagree.

Merkle-tree or IBLT-based reconciliation is the right answer eventually, and adding it is a new sync protocol version that libp2p already knows how to negotiate ([Sync wire versions](#sync-wire-versions)). It is not the right answer first.

**Sync streams are rate-limited per peer.** `GetEventsSince(0)` asks for the entire history, and a set of peers all asking at once is a cheap way to exhaust a node's disk and bandwidth. Every anti-abuse control described elsewhere sits at the gateway ([docs/identity.md](./identity.md#where-these-controls-actually-sit)) — which is precisely the path a peer doing this never touches. A node caps concurrent sync streams, caps events served per peer per period, and drops back to Gossipsub-only for a peer that keeps asking. Serving history is a courtesy, not an obligation.

Pub/sub alone isn't enough — a node offline for a week needs to explicitly ask "what did I miss," not just listen for new broadcasts.

### Sync against a peer that pruned data

Retention is each node operator's own policy, not a protocol-mandated number — which means a sync request can legitimately land on a peer that no longer has what's being asked for. The sync protocol needs to say something more precise than a bare "not found":

- **A node that prunes an event keeps a tombstone** — the `eventId`, the event's `author`, and when it was pruned; never the content. Cheap to keep, and it does two jobs: it turns "not found" from ambiguous into precise, and it preserves the one field the authorship rule needs (see [Who may sign what](#who-may-sign-what)) so a `review.replaced` or `review.withdrawn` naming a pruned target can still be checked instead of being stuck unverifiable forever.
- **`GetEventsResponse` carries a status per requested ID**, not just a flat list of event bytes: `found` (here it is), `pruned` (I had it, it's gone, try a longer-retention peer), or `unknown` (never seen this ID).
- **Where to look next is convention, not protocol guarantee** — community and full nodes (see [docs/concepts.md](./concepts.md#node)) are the natural next hop after a `pruned` response, since they're expected to retain more, but nothing forces them to.

**Honest limitation:** if every node that ever held an event has pruned it, it's genuinely gone from the P2P layer — replication is not a permanence guarantee. Whatever an indexer already projected from it survives as a projection, but the raw signed event itself may become unrecoverable. This is a direct consequence of retention being a local, per-operator choice rather than a network-wide mandate.

**In v1, nodes do not prune.** The wire protocol above is specified now — tombstones, and `found`/`pruned`/`unknown` on `GetEventsResponse` — precisely so that adding pruning later is an operator policy change and not a protocol migration. But no v1 node exercises it. Two reasons: at v1 volumes the entire event log is small enough that pruning buys nothing, and pruning is the one thing that can make the core promise ("the index is a projection, rebuildable from the log") quietly false. Paying for that trade before there's anything to gain from it would be backwards. What v1 does need is the honest disclosure that `pruned` will eventually be a real answer, which is why the status exists in the response from day one.
