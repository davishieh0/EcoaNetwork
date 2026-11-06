# Decision log

Every decision taken during the architecture review of the v1 documents, with what it costs and what was rejected. Written to be argued with: if a decision looks wrong, the reasoning is here to attack rather than buried in a diff.

Three kinds of entry:

- **Gap** — something the documents didn't say, and two implementations would have guessed differently.
- **Bug** — something they said that was wrong or self-contradictory.
- **Call** — a genuine fork in the road where the alternatives were defensible and one was picked.

Everything here is scoped to v1. The bias throughout is toward decisions that are cheap now and expensive later — wire format, envelope shape, anything that would force a lockstep upgrade across the network — and toward leaving genuinely reversible things (weights, thresholds, ranking) as tunable defaults.

A second pass re-read every document against this log to check that each decision was actually reflected in the specs rather than only recorded here. It found stale text in four places and one genuine gap per document on average; those fixes are folded into the sections below rather than listed separately, and the entries marked **§2nd pass** are the ones that came out of it.

---

## 1. Cryptography and the wire format

### 1.1 The signing bytes are the whole envelope, not the payload — **bug**

The spec said `eventId` was `SHA-256` "of the canonical payload", which reads as the `payload` field. Under that reading, two different authors publishing identical review text produce the same `eventId` and collide, and an event can be reattributed to a different author or timestamp while keeping a valid signature.

Signing bytes are now the JCS serialization of the entire envelope **minus `eventId` and `signature`**. `eventId` can't be inside its own digest; `signature` is excluded so the digest covers content only — which has the pleasant side effect of making Ed25519 signature malleability harmless, since a re-signed event has the same `eventId` and is dropped as a duplicate.

### 1.2 Every encoding and format is pinned — **gap**

Unpinned encodings are the classic way two independent implementations fail to interoperate while both believing they follow the spec.

| Thing | Decision | Why this one |
|---|---|---|
| `eventId` encoding | base32, lowercase, unpadded | Case-insensitive, unambiguous alphabet, URL- and filename-safe without escaping |
| `signature` | base64url, unpadded | Compact, URL-safe |
| `createdAt` | RFC 3339 UTC, literal `Z`, second precision | One spelling per instant; no offset variants to hash differently |
| `author` / `deviceKey` | `did:key`, Ed25519, multibase `z` + multicodec `0xed01` | The verification key derives from the identity string, so no separate key field |
| Envelope fields | Closed set; unknown top-level fields rejected | A node that drops an unknown field computes a different digest than one that keeps it, and they'd disagree about validity |
| Payload fields | Open; unknown fields preserved byte-exactly and hashed | Additive payload evolution is the common case and must not need a version bump |
| Unicode | Client NFC-normalizes before signing; nodes never re-normalize | Re-normalizing on receipt would alter the signed bytes |
| Numbers | No floats anywhere in payloads | JCS defines float serialization, but getting it subtly wrong produces a signature that verifies nowhere |

### 1.2b The validation checklist is the implementation spec, so it says exactly what to check — **§2nd pass**

The checklist had drifted behind the decisions around it: it still said "canonicalize payload / recompute hash" after the signing domain moved to the whole envelope (§1.1), and never mentioned the closed-envelope check, identifier normalization, `deviceKey` authorization, or assigning the local sequence.

It's now precise about all of them, and — more importantly — labels which steps are **universal** (1–4, 6–8: no knowledge of the payload needed, which is what lets a node relay an event type it has never heard of), which are **type-specific** (5 and 9, skipped for an unrecognized `eventType`), and which are **local policy** binding nobody but the node applying them (10). An implementer reading only that list should now build something that interoperates.

### 1.3 Entity identifiers are normalized as part of the protocol — **gap**

`cnpj:12.345.678/0001-99` and `cnpj:12345678000199` would have become two entities. A GTIN-13 with an implicit leading zero would have split one product's reviews in half.

Normalization is mandatory and specified per type: CNPJ digits-only padded to 14, GTIN padded to GTIN-14, domains lowercased as punycode A-labels with no `www.`/scheme/port/path. Nodes **reject** an unnormalized identifier rather than fixing it, because normalizing on receipt would change the signed bytes.

`place:` is pinned to OpenStreetMap (`place:osm:n240109189`). **Call:** Google and Foursquare place IDs are proprietary and need an API key to resolve, which would make a core identifier of an open protocol depend on a company's terms of service.

`entity:<hash>` is derived — base32 of `SHA-256(JCS(descriptor))` over a normalized name/kind/country/locality object — rather than random. A random id guarantees a fresh duplicate every time anyone describes the same bakery; a derived one makes identical descriptions converge for free.

**The descriptor has to be published, or the derivation is worthless** (**§2nd pass**). A hash nobody can reproduce is indistinguishable from a random id, so the first `entity.updated` for such an identifier carries the descriptor and any consumer can recompute the hash. An identifier whose descriptor never appears still functions, it just can't be verified as derived — and an indexer may treat it with whatever suspicion an unverifiable identifier deserves.

### 1.4 `deviceKey` is a nullable envelope field in v1 — **call**

This is the one place v1 pays a cost for something it may never use, and the reasoning is worth stating plainly. The envelope is a closed set, so adding a field later is a **breaking version bump the entire network must adopt in lockstep**. The root/device key model (§3.3) is the only real answer to a compromised key, and it needs somewhere to say which key signed an event while `author` stays the identity.

One `null` per event now buys the option to adopt it later without a migration. The alternative was never "add it later" — it was "fork the envelope later".

---

### 1.5 The rating scale — **call**

`review.created` carries a required `rating`: an integer from 1 to 5, 1 worst, 5 best. No `scale` field.

Payload design is otherwise the project's own, and this one field is here for the same reason entity identifiers are (§1.3): its *meaning* has to be identical everywhere, or a client that implements it backwards emits perfectly valid events that mean the opposite of what they say.

- **Integer** — forced by the no-floats rule (§1.2). No half-stars.
- **1–5 rather than 1–10** — past roughly seven points people stop discriminating consistently, so the extra range adds noise rather than signal, and carries far more cross-cultural anchoring variance.
- **Required** — an optional rating puts two structurally different populations of review on the network with nothing distinguishing them, and no way to tell a deliberate abstention from a client that never asked.
- **No `scale` field** — it reads as future-proofing and does the opposite, inviting `scale: 10` and leaving two incomparable scales in circulation. The versioning rules already cover a future scale: a new event type, never a redefinition.

**And the protocol defines nothing about an entity's rating.** No canonical average. How a set of integers becomes a number on a page — mean, distribution, weighted by reputation or not, which reviews excluded — is a projection each indexer decides, like ranking. This is written down explicitly rather than left as an absence, because an unstated omission gets read as an oversight and someone builds on an official score that was never promised.

Worth recording what this decision *rejected* as being outside the protocol layer entirely: weighting the aggregate by identity reputation, showing a distribution instead of a mean, confidence-bound shrinkage so a 5.0 from two reviews doesn't outrank a 4.6 from eight hundred, and excluding an entity's own verified representative from its average. Each is a good idea and none of them is a network rule — they're what a particular indexer chooses, and different indexers should be free to choose differently.

## 2. Time, ordering, and what counts as a fact

### 2.1 `createdAt` is self-attested, and the design stops pretending otherwise — **bug**

Vote weight depended on identity age, and identity age was computed from a timestamp the author signs. Anyone could stamp 2019 on their first event and start at full weight.

There is no trustworthy global clock available and inventing one means electing timestamp authorities. So time was split in two:

- **`createdAt`** — signed, global, rejected if more than **5 minutes** in the future, and never rejected for being in the past because a node cannot disprove it. Authoritative **only for ordering an author's events against each other**, where lying reorders nothing but your own content. Ties break on lower `eventId`, making it a deterministic total order.
- **`receivedAt` plus a monotonic local sequence** — unsigned, local, never gossiped as fact. Identity age is `max(createdAt, earliest local receipt)`, so backdating buys nothing.

**Cost, stated rather than hidden:** a freshly-joined node has `receivedAt ≈ now` for everyone and cannot weight by age at all until it has been running. Its only two options are to inherit a starting view from an indexer it explicitly trusts, or to flatten the age multiplier and let claims carry the defense. Taking `receivedAt` hints from sync peers was **rejected** — the hint is forgeable in exactly the direction an attacker wants.

This is why the weighting rework (§6.1) moves the defense onto claims, which have an issuer and replay identically.

### 2.2 The general ordering rule — **gap**

> Ordering within one author is deterministic and uses that author's own clock. Ordering across authors — and any rule an attacker holding the author's key would want to move — is a local observation.

Every "the latest" in these documents (current profile, chain head, an identity's current vote) previously had no defined ordering under a forgeable clock. Now it has exactly one, stated once.

### 2.3 Rotation and revocation take effect on local observation — **gap**

The adversarial exception to §2.2. Self-scoped ordering assumes lying only hurts yourself, which is false precisely when a thief holds the key: they'd backdate events to before the revocation and walk right through a `createdAt`-based cut.

So a revocation or rotation takes effect **when the node observed it**, not at its claimed timestamp. Events accepted before that stay valid — a revocation is a full stop going forward, never retroactive erasure. **Cost:** the cut is a local observation, so nodes differ by propagation delay. Correct trade, since the alternative lets the attacker place the cut.

### 2.4 Sync cursors run over local sequence, never `createdAt` — **bug**

A cursor over author timestamps loses events silently and permanently: an event stamped three years ago and published today falls behind any cursor a peer already passed, and is never delivered to anyone syncing incrementally. Both node↔node sync and node→indexer polling now use the serving node's local receive sequence. Cursors are therefore peer-specific and not portable.

---

## 3. Identity and key compromise

### 3.1 One successor, ever — and the first one observed is the one that holds — **gap, then bug**

Two `identity.key_rotated` events signed by the same key naming different successors were both valid, which forks an identity in two with different parts of the network disagreeing about who someone is. An identity now has **at most one** valid successor for its whole life; chains are transitive, cycles rejected.

The *resolution* rule was wrong on the first attempt and is corrected in §11.1: it was lowest-`eventId`-globally, and it is now first-observed-wins, with an unorderable conflict failing closed. See there for the attack.

### 3.2 `identity.key_revoked` — **gap**

Revocation with no heir: *"this key is compromised, trust nothing signed by it from here on."* An attacker can publish it too but gains nothing by burning an identity they just stole. For the real owner it's the difference between watching a thief inherit their history and publicly killing the identity.

### 3.3 Root key and device keys: designed now, optional in v1 — **call**

A cold root signing only authorization/rotation/revocation, hot device keys signing content. This subsumes "per-device identity" and "encrypted backup" into one mechanism instead of two, and it's the only real answer to compromise.

**Behavior optional in v1, shape fixed in v1** (see §1.4). The plain statement that replaces the previous hand-waving: **without a recovery path, compromise is terminal.** Whoever holds the key is the identity, and there is no authority to appeal to — the same property that stops anyone seizing an identity they don't control.

Whether v1 ships the behavior or only the shape is settled in §3.5.

### 3.4 Authorship over derived events — **gap** (rule supplied by the project)

An event that **replaces, withdraws, or revokes** another must carry the same `author` as its target. An event that merely **references** another — response, vote, report, label — is a new independent statement and carries no restriction.

Without this, any identity could edit or delete any other identity's review and every node would accept it: valid signature, valid schema, target exists.

Two consequences: "same author" resolves through the rotation chain (so rotating a key doesn't orphan your own history), and pruning tombstones now retain the target's `author` so the check survives on a node that pruned the target.

### 3.5 v1 ships the single-key model and rejects any `deviceKey` — **call**

The question left open in §3.3 was how much of the root/device model a v1 node enforces. The answer is none of it: a node accepts only `deviceKey: null`, verifies the signature directly against `author`, and rejects an event with `deviceKey` set, with that as the stated reason. The envelope field stays, nullable, exactly as §1.4 fixed it, so the model can arrive later without a version bump.

**Rejected: sending a device-signed event to the orphan buffer.** It is the obvious reading of the old text, where a device event whose authorization has not arrived waits like any unresolved reference. In v1 the authorization will never arrive, because no node processes `identity.device_authorized`. The event would sit in the buffer until it is evicted and resolve into nothing, and anyone could fill that buffer with events built to wait for it.

**Rejected: implementing both models now.** Device-signed events add a second verification path (against `deviceKey`, then the authorization from the root), device revocation taking effect on observation, and the orphan case above. That is the part of validation where two independent nodes most easily disagree about validity (§11.3), added ahead of any need.

**The cost is the one §3.3 already named:** without a recovery path, compromise is terminal. A stolen key can edit and withdraw that identity's history and revoke the identity, and nothing in v1 undoes it. The second cost is that rejecting `deviceKey` is a validity rule. The day device-signed events become valid, a node that has not upgraded will keep rejecting them, so the change has to be adopted by the network together, as any change to validity is.

---

## 4. Propagation

### 4.1 One gossip topic, not several — **call**

The docs proposed `/ecoa/events/1`, `/ecoa/moderation/1`, `/ecoa/entities/1`. Rejected in favor of a single `/ecoa/events/1`.

Per-category topics look like a scaling win and are actually a correctness bug: a node subscribed to some topics holds a partial log, so its index can't be rebuilt completely and its reference checks fail against events that exist on the network but not locally — turning ordinary partial subscription into a permanent orphan factory. One topic makes every node's log complete by construction. If per-topic subscription is ever needed, it can be added alongside, with the firehose remaining complete.

### 4.2 Orphan buffer for out-of-order references — **gap**

Gossip doesn't deliver causally. A vote routinely arrives before the review it targets. Under strict step-9 rejection, that valid event is dropped **and not rebroadcast** — so ordinary network timing would be indistinguishable from an attack, and the network would lose valid events at a rate proportional to how fast it is.

Events that pass every check through signature verification but whose reference doesn't resolve go to a bounded buffer (**1 hour or 10,000 events**, oldest-first eviction), are re-run when the target arrives, and are not rebroadcast until they resolve. Eviction is safe: a genuine orphan returns through ordinary sync, because the peer holding the target holds the child too.

Two details that matter: unknown-`eventType` events are **not** orphans (reference checks are type-specific, so they have nothing to fail and propagate normally), and the buffer bound is **per peer**, since filling it with references that will never resolve is otherwise a cheap way to evict everyone else's legitimate orphans.

### 4.3 v1 nodes do not prune — **call**

Pruning is the one thing that can make the project's central claim ("the index is a projection, rebuildable from the log") quietly false, and at v1 volumes it buys nothing.

The wire protocol for it — tombstones, `found`/`pruned`/`unknown` on `GetEventsResponse` — is specified now anyway, so enabling pruning later is an operator policy change rather than a protocol migration. This also closes the "node storage/retention policy" and "minimum replica target" open questions for v1 (retain everything; 3 replicas, matching the acceptance criteria).

---

## 5. Reviews, edits, and votes

### 5.1 Replace chains — **gap**

`replaces` names the **immediately preceding version**, not the original, so a chain is unambiguous. Forks (two edits naming the same predecessor) resolve by §2.2 self-scoped ordering. Walks are capped at **64 links** and report truncation rather than guessing. `review.withdrawn` targets any link and withdraws the **whole chain** — otherwise withdrawal would hide one version while the rest stayed readable.

### 5.2 Votes bind to the version they were cast on — **call**

The attack: publish something agreeable, collect endorsements, replace the text, keep the borrowed credibility. The chain makes it detectable, but nobody audits.

A vote binds to the exact `eventId` it targeted. The head's score counts only votes on the head; earlier votes stay visible as chain history and never contribute to current ranking.

**Cost, and it's real:** fixing a typo resets a review's score. Accepted deliberately — the rule fails toward *unearned score is not displayed* rather than *an attacker keeps credit for text nobody endorsed*, and putting a price on casually rewriting a published review is a feature on a review platform. This is a projection rule, and the wire format already records which version a vote targeted, so a future version can soften it (carry-forward with decay) with no protocol change.

### 5.3 Labels and responses **do** follow an edit — **call**

The deliberate asymmetry with §5.2. If a label bound to a version, clearing a spam label would cost one character of editing, and moderation would be defeated by the cheapest available action. Responses stay attached because a company is replying to the complaint, not to one revision of its wording.

> Credit binds to the version that earned it; warnings and conversation bind to the thing. Anything the author gains by editing must not survive the edit; anything the author could escape by editing must.

### 5.4 Conflicts of interest are surfaced, not blocked — **gap**

A verified company representative can review and vote on their own entity. Nothing should prevent it — a signed event is a signed event, and blocking it needs an authority to enforce. But it's mechanically detectable, since the claim and the review are both public and name the same entity, so indexers mark it. Cheapest possible defense against the most obvious abuse of verification, no new event type.

---

## 6. Reputation

### 6.1 Weight per claim type, not a verified/unverified flag — **bug**

The old rule promoted any identity holding any claim from 0.1 to 1.0. The cheapest claim is `human`, issued by a gateway CAPTCHA; solved CAPTCHAs cost about a dollar per thousand. The binary rule therefore sold a thousand full-weight identities for a dollar.

| Recognized claim | Weight |
|---|---|
| none | 0.1 |
| `human` | 0.3 |
| `email` | 0.6 |
| `domain` / `cnpj` | 1.0 |

`claim_weight` is the **strongest** claim held, never the sum — summing cheap claims recreates the same problem. Also recorded: `purchase` (post-v1) is the only claim type that ties a voter to the transaction actually being reviewed.

### 6.2 "Latest vote" is defined, and votes have meaningful targets — **gap**

Latest = higher `createdAt`, ties on lower `eventId`, which is safe because it's self-scoped (§2.2). `target` may be any `eventId`, but only reviews and responses count in v1 — allowed rather than rejected so a future event type needn't change the wire format, ignored rather than counted so scores don't accumulate somewhere with no display surface.

### 6.3 Paid brigading is named as a day-one adversary — **honesty fix**

Still deferred past v1, but the docs no longer imply it's an exotic late-stage threat. What buys the deferral is stated instead: the naive case is the common one, and damage is bounded by ranking rather than deletion.

---

## 7. Entities

### 7.1 Tier 1 requires a *recognized* claim — **bug**

`entities.md` granted authority to anyone holding a claim; `verification.md` said delegated claims count only from recognized verifiers. Read literally, a self-signed `cnpj` claim gave canonical control of any company. Tier 1 now requires recognition: delegated types from a verifier on the indexer's list, self-verifiable types actually checked.

### 7.2 Claim conflicts resolve by precedence, not by tie — **bug**

Falling back to tier 2 on any conflict made demotion a cheap attack: obtain any competing claim and strip a legitimate representative of control. Now ordered — self-verifiable beats delegated; higher-ranked verifier wins; same verifier, earlier claim holds; tier 2 only when precedence genuinely cannot separate them.

### 7.3 A claim grants authority over the identifier it names, and nothing else — **bug in my own earlier reasoning**

Bringing `domain` verification into v1 (§8.2) does not give anyone authority over `cnpj:` entities, because nothing in a DNS record says which CNPJ is behind it. So **v1 tier-1 authority covers `domain:` entities only**, while many reviews reference `cnpj:` ones.

Three things keep this workable: `domain:` is the better identifier to steer clients toward anyway; `cnpj:` entities are seeded from public data (§7.5); and a domain-verified representative can propose an alias to a CNPJ, carrying high weight but **not** tier-1 authority over the CNPJ side.

**Honest residual cost:** reviews about one company can land on both identifiers and stay split until an alias earns weight. A better cost than shipping a forgeable verification path.

### 7.4 Tier 2 resolves per field, by author weight, with hysteresis — **gap**

Nobody votes on a company's category, so ranking proposals by votes meant ranking zero against zero and letting the implementation's accidental tiebreak become the real rule — almost always last-write-wins, which hands every entity to whoever edited most recently.

Per **field**, the canonical value is the highest-weighted proposal setting it; with no votes, weight falls back to the **author's identity weight** (already defined for voting). Ties break on earlier, then lower `eventId`. A challenger must beat the incumbent by **20%**, or near-equal proposals make an entity's name flip every time weights drift.

### 7.5 Seed entities from public data instead of arbitrating them — **call**

Receita Federal CNPJ data is public and bulk-downloadable. Seeding makes most entities arrive already correct and turns the edit-conflict path into the exception rather than the road every entity travels. Seeded values enter as ordinary tier-2 proposals authored by the operator's identity — competing under the same rules, not as a privileged back door.

### 7.6 Verified-representative edits are authoritative but not unaccountable — **gap** (was an open question)

Every proposal is retained including tier-1 ones; canonical is a pointer into history, never a destructive overwrite. Labels apply to `entity.updated` exactly as to a review. An indexer can fall back to the previous canonical value as a deliberate policy action — **not** an automatic rollback rule, which would just hand the same power to whoever could trigger the automation.

Residual exposure, stated: between a bad edit and someone noticing, the vandalized value is what readers see. Verification buys speed at that cost; tier 2 buys review at the cost of being slow.

### 7.7 Official responses need no new mechanism — **gap**

"Official" is a display fact, not a permission: the response came from an identity holding a recognized claim for the entity. Anyone may still respond; they just don't get the badge.

### 7.8 Three event types deleted — **call**

`profile.created`, `entity.created`, and `entity.alias_confirmed` are gone.

The `*.created` pair would have been indistinguishable from the first `*.updated` — same payload, same author, same effect — while inviting the question of which outranks the other. Profiles and entity metadata are last-writer-wins records; the first write is not a special kind of write. `entity.alias_confirmed` implied an authority to confirm that doesn't exist; an alias is an ordinary weighted proposal, and a verified representative's proposal already carries tier-1 weight, which is what "confirmed" was reaching for.

---

## 8. Verification

### 8.1 A `domain` claim is live state, not a replayable fact — **bug**

DNS is checked live, so an index rebuilt three years later gets a different answer — which quietly falsified "the index is rebuildable from the logs."

A failed lookup **downgrades** the claim, it doesn't revoke it (revocation is a statement by the issuer; a stale DNS record isn't). The result is a **local observation** in the same category as `receivedAt`. Claims carry `expiresAt`, capped at **one year**, re-checked at least every **30 days**.

Rejected for v1: signed observations ("I saw this TXT at time T") to restore replay determinism. Strictly additive, so it can arrive later if determinism ever becomes necessary.

### 8.1b What actually goes in the DNS record — **§2nd pass**

The issuance steps said the client "generates a nonce and signs it" and then publishes "that value" — which left an implementer to guess whether the TXT record holds the nonce, the signature, or something else. Two clients would have guessed differently and neither could verify the other.

Pinned: the record is at `_ecoa-verify.<domain>` and its value is exactly `ecoa-identity=<did:key>`. Multiple records are allowed, so one domain can vouch for several identities.

**The nonce is dropped, deliberately.** A nonce is the right shape for a one-time challenge; this is the opposite. The record must persist and stay re-checkable because the claim is live state on a 30-day TTL (§8.1). Control of the domain *is* the evidence, and the record needs to prove exactly one thing — this domain vouches for this identity. A nonce would add a step and a way to get it subtly wrong while proving nothing more.

A `.well-known/ecoa-verify` file carrying the same line is an accepted fallback for anyone who can't edit DNS, explicitly marked as weaker (it rests on TLS and a web server rather than on the zone), with DNS preferred when both exist.

### 8.1c `evidenceRef` is a locator, never evidence — **§2nd pass**

Was an open question with a placeholder. Settled: a scheme-prefixed string saying *where* the evidence lives (`dns-txt:<name>`, `https://<url>`) for self-verifiable claims, and **`null` for every delegated claim** — where the evidence is private by construction, and a locator would only advertise that it exists.

### 8.1d The email verifier's default is to keep nothing — **§2nd pass**

The privacy note said the verifier keeps the address "if it needs it for something like account recovery" — which sat badly against the fact that no account can recover a key (§3.3), and glossed over what retention actually creates: the single piece of data on the whole system linking a pseudonym to a person, which makes the verifier the natural target of any legal demand for an author's identity.

Default is now verify, sign, discard. Retention is a deliberate decision that needs its own justification, not a convenience assumed by default.

### 8.2 `domain` verification is in v1 scope; `cnpj` stays out — **call**

Without it, v1 has no tier-1 entity authority at all and no way to mark a company response as genuinely from the company. It's also the cheapest claim to build: a DNS lookup, no verifier service, no manual review. `cnpj` needs a manual verifier process and `domain` covers any company with a website.

### 8.3 `author` vs `subject`, and consent — **gap**

Self-verifiable claims require `author == subject`; a self-signed claim *about someone else* asserts nothing and is discarded at projection. Delegated claims differ by construction.

Stated outright because it surprises people: **nobody's consent is required to publish a claim about them.** Anyone can sign an event asserting anything about any identity. That's exactly why authority lives in recognition, not in publication.

---

## 9. Architecture and honesty

### 9.1 The determinism boundary is written down — **gap**

"The index is a projection, rebuildable from the log" is only true of part of the index, and being vague about which part is the difference between a testable property and a slogan.

**Must be identical across honest indexers:** which events exist and their content, structural validity, chain heads and withdrawal state, which votes and responses attach to which version, which entity metadata proposals exist, all within-author ordering.

**May legitimately differ:** `receivedAt` and identity age, whether a `domain` claim currently verifies, reputation and vote weight, recognized verifiers and moderators, ranking — plus two the second pass moved across (**§2nd pass**): *which* entity metadata proposal is canonical, since with no votes it resolves on the author's weight and weight is an observation; and the revocation cut, since whether an event signed just after an `identity.key_revoked` is accepted depends on whether this node had seen the revocation yet.

The test for which list something belongs in turned out to be sharper than "is the rule shared": the rules are written down and identical either way. It's **"can it be recomputed from the events and nothing else"** — and both items that moved have a deterministic rule fed by a non-deterministic input.

A disagreement in the first list is a bug; a disagreement in the second is the design working. Moving anything from the second list to the first requires electing an authority over clocks, DNS, or ranking. The MVP's rebuild criterion was narrowed to the first list, and a second criterion added: two independently-built indexers fed the same log must agree on all of it.

### 9.2 Integrity is verifiable, completeness is not — **honesty fix**

The risk table claimed client-side signature verification mitigates a manipulating indexer. It doesn't: a client can only check what it was handed, so **omission is undetectable**. The mitigation is plural indexers a client can compare; a completeness commitment (signed event counts, a Merkle root) is recorded as an open question.

Related requirement: gateway reads return **whole signed envelopes**, not flattened rows. A client handed a prettified projection has nothing left to verify, which would turn client-side verification into decoration.

### 9.3 The client is a static bundle; the gateway is data-only — **call**

"You can point the client at another gateway" is close to meaningless if the same operator ships the JavaScript. Whoever serves the code can serve code that exfiltrates the key, and no CSP or non-extractable key survives that.

So: the client ships as a static, independently hostable bundle with no server-rendered or gateway-injected code, the gateway never serves application code, and gateway choice is client-side runtime configuration. This makes the MVP criterion "pointable at a different gateway without code changes" a security property rather than a convenience.

### 9.4 Anti-abuse controls sit at the gateway, which is optional — **honesty fix**

Rate limits and CAPTCHAs are bypassed by anyone willing to run a node and gossip directly, and Gossipsub peer scoring punishes *invalid* traffic, not valid spam. The real weight sits on per-node admission policy (acceptance is local even though validity is protocol-level) and on read-time weighting (a million zero-weight events are shown to nobody). Proof-of-work stays deferred but is named as the only control that would make publishing itself cost something.

### 9.5 The censorship claim was softened to what's actually true — **honesty fix**

No one can silently **edit or delete** — that's cryptographic and holds. No one can **hide** is a much weaker claim: a reader sees what their indexer returned, and on day one there is exactly one indexer, run by the project. The overview now says so and points at progressive decentralization, instead of promising a property the launch configuration doesn't have.

### 9.6 Pseudonymous is not anonymous — **gap**

`author` is stable across everything an identity ever signs, all of it public and permanent, so one identity's whole history is trivially linkable forever. For the employee-describing-their-employer case this platform explicitly wants to serve, that's a real hazard that worsens the longer someone uses one identity well.

The protocol already permits the mitigation — identities are free and nothing requires using only one — and the cost is exact: a fresh key has weight 0.1 and no age. **Weight and unlinkability are the same trade-off seen from two sides.** Clients must make "publish under a fresh identity" a visible one-tap option and say what's being traded, rather than silently reusing the default identity and letting users discover the linkage afterwards.

### 9.7 Legal exposure is named, not solved — **flagged, deferred by the project**

Deliberately not designed here, but no longer answered by a single non-goal line. Marco Civil art. 19, LGPD erasure, and who receives a subpoena are recorded, along with the observation that **the architecture already puts the compliance point in the right place — the indexer and gateway, not the node** — and that third parties running nodes inherit hosting exposure that shipping a container image without documentation would pass along in silence.

---

### 9.8 "Disputed" means a vote, never a label — **§2nd pass**

Three documents used "disputed" as an example moderation label while `review.disputed` is a community vote — the two systems the design goes out of its way to keep separate (§5.3). Label examples are now spam / off-policy / illegal, and "disputed" is reserved for the vote. Small, but a shared vocabulary that collides on the one distinction the design most wants understood is worse than no vocabulary.

Also corrected in the same pass: `identity.md` still listed identity age among "raw facts the protocol stores", which §2.1 had made false, and still described per-device identity as an alternative to the root/device model rather than as the thing that model specifies.

## 10. Scoping the v1 down to something buildable

A late pass re-read the scope as a *schedule* rather than as a design, asking which items cost the most and demonstrate the least. Four cuts came out of it, and two additions.

### 10.1 Sync loses the inventory exchange — **call**

The spec had the syncing node request an inventory of event IDs, diff it locally, and fetch the difference. The per-peer cursor decision (§2.4) had already made this redundant without anyone noticing: inventory diffing exists for the case where you don't know where you left off, and a cursor is exactly knowing where you left off.

Sync is now two calls — `GetEventsSince(cursor)` to catch up, `GetEvents(ids)` to fill a known hole such as an orphan's target. What the inventory bought was bandwidth when re-syncing against a peer you've never met, since you re-receive events you already hold from someone else; duplicate detection discards those for a hash lookup each. At any volume this network will see for years, that's a few megabytes against a materially simpler protocol with fewer places for two implementations to disagree. Merkle or IBLT reconciliation is the eventual answer and is a negotiated sync-protocol version, not a redesign.

### 10.2 `email` verification and entity aliases leave v1 — **call**

`email` needs a verifier service, SMTP, deliverability and templates: real infrastructure with nothing visible to show for it. `human` (an off-the-shelf CAPTCHA) already separates a fresh key from a slightly less fresh one, and `domain` — the one claim type needing *no verifier service at all*, just a DNS lookup — carries tier-1 entity authority and the official-response badge. One self-verifiable and one delegated claim type is enough to demonstrate that the claim model works.

Entity aliases stay in the protocol but no v1 indexer projects them. Grouping two entities carries real indexer complexity against almost nothing visible, and it's purely additive.

### 10.3 An inspection view and a seeded dataset enter v1 — **call**

The only two additions, and both exist to make existing work legible rather than to add capability.

Most of the effort in this project is invisible: canonicalization, signature verification, replication, chain resolution. An **inspection view** — raw signed envelope beside the rendered review, live verification in the client, which nodes hold the event — converts that into something a person can point at.

A **seeded dataset** (entities from public Receita Federal data, plus a generator producing properly-signed synthetic reviews) is the difference between a demonstration that looks like a system and one that looks like a toy. Both are cheap next to what they make visible.

### 10.4 Phase 4 is the declared cut line, and libp2p is the declared risk — **call**

Profiles, responses, reports and moderation labels can be described as designed-but-not-implemented without weakening the central demonstration. Everything before Phase 4 cannot.

libp2p is separately named as the single biggest schedule risk, with a pre-decided fallback (plain WebSocket gossip over a static peer list) that the architecture already sanctions, since libp2p is transport and the protocol is meaning. It's recorded as a plan B rather than a starting choice — deciding it in advance is worth more than leaving it to be discovered under pressure.

## 11. Security review

A threat-model pass over the finished design, looking for the way in rather than for internal contradictions. It found one flaw that allowed identity takeover, two that could partition the network, one missing separation between networks, and one privacy claim the design could not actually keep.

### 11.1 Retroactive identity hijack via a ground `eventId` — **bug, the serious one**

Conflicting rotations resolved on the **lower `eventId`, globally and forever** (§3.1). The attack:

1. Alice rotates `key_old → key_new` in January; the network accepts it.
2. Alice publishes for six months under `key_new`.
3. In July, Mallory recovers a backup of `key_old` — a decommissioned laptop, a leaked archive.
4. Mallory publishes `key_old → key_mallory`, varying `createdAt` until the `eventId` falls below January's.
5. Mallory's rotation becomes canonical *retroactively*. Alice's becomes invalid, six months of her events are signed by a key with no relationship to the identity, and Mallory holds it.

**The grinding is free** — landing an `eventId` below a given random 256-bit value takes about two attempts on average. It's arithmetic, not work.

Worse, the rule contradicted the document's own general principle, which already said that *anything an attacker holding the author's key would want to move is a local observation*. A rotation is precisely such a thing, and the rotation section was the one place that principle wasn't applied.

**Now: first observed wins.** A node that has accepted a rotation rejects any later conflicting one outright. A node back-filling history that receives both at once cannot order them, and **fails closed** — the identity is contested, neither successor is canonical, history under the original key stays readable and verifiable.

**The cost:** a stolen key can still *freeze* an identity, and two nodes can hold different views of a contested one. Given a leaked key those are the only options available, and freezing is survivable where transfer is not. The general lesson is worth keeping: a rule with no clock in it also had no *history* in it, which let a stale key reach back past months of accepted fact.

### 11.2 No network identifier: test events were valid in production — **gap**

`protocol` and `version` sat inside the signed bytes and prevented cross-*protocol* replay, but nothing separated a test network from the real one. An event signed on `ecoa-dev` was cryptographically valid on the main network — same key, same schema, same signature — so test data could be lifted into production wholesale and throwaway-network signatures were indistinguishable from real ones. Separate gossip topics stop accidental propagation and do nothing about deliberate copying, because the events genuinely are valid.

`protocol` now names the network: `ecoa`, or `ecoa-<name>` for anything else, checked at step 2 before any other work. **Cost: zero** — it was already a constant string inside the signed bytes. It only had to be decided before v1 shipped, since the envelope is a closed set.

### 11.3 Two ways for honest nodes to disagree about validity — **gap**

Both would partition the network rather than compromise a key, which is the failure mode that only shows up in production and is hardest to diagnose.

**Ed25519 verification isn't uniform across libraries.** Cofactored versus cofactorless verification, small-order points, non-canonical encodings — each handled differently by different implementations. Pinned to a single rule set, with a signature test-vector set promoted to a Phase 0 deliverable. (The rule set chosen here was *wrong* on the first attempt and is corrected in §13.1 — it asked for cofactorless verification, which is the one choice that guarantees disagreement.)

**JSON parsing happens before canonicalization.** Duplicate object keys are resolved differently by different parsers — first wins, last wins, or error — so two implementations compute different digests for identical bytes. Duplicate keys are now rejected outright. Nesting depth is capped at 32 in the same breath, since the 16 KB ceiling bounds size but not structure, and thousands of levels fit inside it — enough to overflow a recursive canonicalizer's stack *before* any signature is checked, which is the cheapest possible denial of service.

### 11.4 Signatures had no domain separation — **gap**

Ed25519 signed the raw canonical bytes. `protocol` and `version` inside the JSON made cross-context replay hard, but only by accident of content. Signing input is now prefixed with `"ecoa-event-v1" || 0x00`, so a signature made for an Ecoa event can never be valid for anything else the same key is ever asked to sign — a login challenge, a session token, a feature nobody has designed yet. Free today, impossible later without invalidating every signature ever made.

### 11.5 The privacy claim ignored the network layer — **honesty fix, and the one users would feel**

"Pseudonymous is not anonymous" covered linkability *between events* and never mentioned that **publishing reveals an IP address**. The gateway sees the source address next to the signed event it forwards, and in v1 the project runs the only gateway. Publishing under a fresh identity from the same session and address, minutes after using the main one, correlates them immediately — quietly defeating the mitigation the same document recommends.

This failed exactly where the docs claim to be most useful: an employee describing their employer was protected from readers and not at all from whoever holds the gateway logs.

Now stated, with what v1 can honestly do: gateways must not log source addresses alongside event content (rate limiting needs a counter, not an audit trail); the interface must say so at publishing time; real network anonymity needs Tor or equivalent and is out of scope; and a client offering throwaway identities must say that a new key is not a new address.

### 11.6 Fixed alongside, all smaller

- **Sync had no resource limit.** `GetEventsSince(0)` from many peers is a cheap way to exhaust a node — and it's exactly the path that never touches the gateway where every other rate limit lives. Sync streams are now capped per peer.
- **`deviceKey` had to be pinned as required-but-nullable.** Under JCS a null-valued key and an absent key hash differently, so an implementation that omits it computes a different `eventId` for identical content.
- **DNS lookups are a trust dependency.** A hostile resolver can forge a `domain` verification. Now: DNS-over-HTTPS to a deliberately chosen resolver, DNSSEC preferred where the zone is signed.
- **An expiring domain transfers authority.** Whoever registers a lapsed domain becomes the verified representative of that entity, with tier-1 edit rights and an official-response badge over every historical review. Inherent to domain control and unsolvable here — the mitigation is presentational: show *since when* a representative has been verified, and treat a change of representative as a visible event rather than a silent swap.
- **Unicode confusables in entity and profile names.** NFC settles composition, not identity: a Cyrillic `А` renders identically to a Latin one. Indexers should fold names to a confusable skeleton before checking for collisions, flag mixed-script names, and show the identifier alongside the name for anything consequential.
- **Every string from the network is user content** — profile names, entity names and categories, label text, seeded data — not only review bodies. An entity name renders in a page heading, which makes it the highest-value injection target and the one most likely to be mistaken for trustworthy.
- **Derived fields are unverifiable by construction.** Scores, applied labels and resolved chain heads carry no signature. A client that cares must fetch envelopes and recompute — possible for everything on the deterministic side of the boundary, impossible in principle for anything on the observation side.
- **The 5-minute clock rule makes NTP an operational requirement.** A node with a badly wrong clock rejects healthy traffic.

### 11.7 Accepted, and named rather than waved past

- **Revocation is an irreversible denial of service against its own subject.** "An attacker gains nothing by burning a stolen identity" is only true of an attacker who wants the identity. One who wants it *silenced* gets exactly what they came for — a motive worth taking seriously on a platform built for publishing what companies would rather suppress.
- **Eclipse attacks are real but bounded, and the bound is sharp.** An attacker controlling every peer a node talks to can show it a *subset* — withholding, delaying, curating. They cannot show it anything *forged*, because validation is local and no peer is ever trusted for asserting validity. Eclipse is a censorship and availability attack, never an integrity one.
- **A compromised verifier key mints `human` claims**, worth 0.3 weight each — cheaper than farming CAPTCHAs, bounded by the same ceiling. It buys nothing against `domain`, which has no issuer to impersonate. That asymmetry is the argument for preferring self-verifiable claim types, and another reason deferring `email` was the right call.

## 12. Payloads

The last open design item, and the one that closes the protocol. Only the conventions and the choices that could reasonably have gone the other way are recorded here; the field-by-field spec is in [protocol.md](./protocol.md#payloads).

### 12.1 Conventions — **gap**

- **Nothing in a payload duplicates the envelope.** `author`, `createdAt` and `eventType` live there. Duplication creates the question of which wins when they disagree, and the right answer is not to have the question. Three existing examples in the documents violated this and were corrected: the vote example carried `type`, and the rotation example carried both `oldIdentity` (the `author`) and a `type`.
- **Optional means omitted, never `null`.** The envelope's `deviceKey` remains the project's only required-nullable field, and only because the envelope is a closed set.
- **Schemas validate what they know and permit what they don't** — required by the versioning rules, which say a new optional field must survive a node that can't interpret it.
- **Per-field length limits**, because otherwise the 16 KB envelope ceiling is the only bound and a single field can consume the whole event.

### 12.2 Replacement is total, except for entities — **call**

`review.replaced` and `profile.updated` carry complete content; an omitted field is cleared. Patches need merge semantics and an ordering to merge along, and total replacement is unambiguous while the chain already preserves every earlier version.

`entity.updated` is the deliberate exception, resolved **per field**, because tier-2 resolution has to distinguish a field the author asserted from one they said nothing about. Under total replacement every proposal would implicitly assert *"and everything else should be empty"* — which would make a correction to one field silently blank the others.

### 12.3 A review's subject is immutable — **call**

`entity` appears in `review.created` and **not** in `review.replaced`. Moving a published review to a different company isn't an edit; it's a different act, and worth making structurally impossible rather than merely discouraged. Vote-to-version binding (§5.2) already prevents the obvious profit from it, but the rule costs nothing and closes the door properly.

### 12.4 Free text is used sparingly, and never where it does harm — **call**

`review.withdrawn.reason` is an **enum** (`mistake`, `resolved`, `duplicate`, `privacy`, `other`). Free text there would be permanent, replicated user content inside the one event whose entire purpose is to retract content — a personal-data surface in precisely the wrong place. An author who wants to explain publishes a response.

Moderation labels and reports share one controlled vocabulary (`spam`, `off-topic`, `illegal`, `personal-data`, `impersonation`, `harassment`, `other`) because free-text labels make two moderator lists incomparable, and "following a moderator" then means nothing a client can act on. `note` stays free text, since a moderator explaining a call is the case where prose earns its place.

**Labels carry no action field.** A label says what something *is*, never what to do about it. That the consumer decides is what makes moderation pluralistic at all.

### 12.5 Category is a fixed flat vocabulary — **call**

Seventeen values, no hierarchy. A free string means everyone invents their own and category filtering stops working; a real taxonomy is a project of its own. The flat list stays extensible because a new value is additive and an older indexer simply doesn't recognize it — which is not true of introducing a hierarchy later, so flat is the choice that keeps the option open.

### 12.6 Smaller calls worth recording

- **`entity.alias_proposed` puts the lexicographically smaller identifier in `entity`.** Without it the same pair yields two distinct events nothing can deduplicate.
- **`descriptor` is mandatory for `entity:<hash>` identifiers** and forbidden elsewhere — it's what makes the derived hash reproducible (§1.3). The consequence for clients is that creating an entity with no public identifier means publishing `entity.updated` alongside the first review.
- **`expiresAt` is required on every claim**, capped at one year for `domain` (control of a domain is a lease, not a property) and two years for the rest. This closes an open question rather than preserving it.
- **`review.response_created.target` may be a review or another response**, giving one layer of conversation for free — the interesting case on a complaints platform.
- **No avatar and no links on a profile.** Attachments are out of v1 scope, and an unverified link field on a public profile is a phishing surface that costs nothing to omit.
- **`lang` is optional and exists for search quality.** Portuguese and English stem differently and automatic detection is wrong often enough to matter on short text. An indexer that ignores it loses only relevance.
- **`device.label` is optional and warned about.** Device names leak more than people expect, and the event is permanent.

### 10.5 Sync serving limits have defaults — **call**

`protocol.md` requires a node to cap concurrent sync streams and events served per peer, and names no numbers. The node reads three variables: `SYNC_MAX_STREAMS` (32), `SYNC_MAX_STREAMS_PER_PEER` (2) and `SYNC_EVENTS_PER_PEER` (4096 per minute, a fixed window).

4096 events is sixteen full pages: enough for a peer to catch up a log of v1 size in one visit, little enough that `GetEventsSince(0)` on repeat cannot drain a node. A stream over any limit is reset and the connection stays open, so the peer is still reachable by Gossipsub.

## 13. Stack for the node

The first stack decision, taken when the node phase was picked up rather than in advance. It brought one correction with it.

### 13.1 Verification is ZIP-215, not "as strict as possible" — **bug**

§11.3 pinned Ed25519 verification to three rules: reject non-canonical `S`, reject small-order points, and verify **cofactorlessly**. Researching Go libraries to implement it surfaced that two of the three were wrong, and wrong in the direction that causes the exact failure the rule existed to prevent.

**ZIP-215 is cofactored.** Cofactorless verification is the choice whose result depends on whether a small-order component happens to be present — the textbook way two honest implementations end up disagreeing about the same bytes. Asking for it while citing ZIP-215 as the justification was a straightforward error.

The rules, corrected:

| | Old (wrong) | ZIP-215 |
|---|---|---|
| Equation | cofactorless | **cofactored**, `[8]R = [8]([s]B − [k]A)` |
| Scalar `S` | reject non-canonical | reject non-canonical — unchanged |
| Point encodings | reject non-canonical | **accepted**, if they decode at all |
| Small-order points | reject | **not rejected** |

**The underlying mistake was conflating two goals.** Rejecting everything questionable buys *signature binding*: no second valid signature over the same message. Making every implementation agree buys *consensus*. They are different properties with different rule sets, and the section asked for the first while arguing for the second.

This project needs only consensus — and it already has binding from elsewhere, because `signature` is outside the `eventId` digest (§1.1), so a mauled signature produces the same `eventId` and is discarded as a duplicate. A decision taken for one reason turned out to cover a second.

`did:key` decoding stays strict, and that is not in tension: canonical multibase is about identities comparing as exact strings, not about which signatures verify.

### 13.2 The node is written in Go — **call**

`go-libp2p` is the reference implementation of libp2p, and Gossipsub plus request/response streams is well-trodden ground there. Since libp2p was the largest schedule risk on the board ([docs/roadmap.md](./roadmap.md)), picking the language with the most mature binding is the cheapest available reduction of it.

Two library choices follow, and both are consequential enough to record:

**`github.com/hdevalence/ed25519consensus` for signature verification, not the standard library.** Go's `crypto/ed25519` uses the unbatched (cofactorless) equation, does not check canonicality of `A`, and enforces canonicality of `R` only as a side effect of comparing bytes — so it does not implement §13.1 and cannot be made to. There is precedent for the risk being real rather than theoretical: Go's IBM z/Architecture backend has diverged from its own software implementation, meaning two machines running the same standard library disagreed about a signature. `ed25519consensus` is a fork of the standard library that implements ZIP-215, and is the de-facto choice for consensus-critical Go.

**`github.com/gowebpki/jcs` for canonicalization**, a fork of the RFC 8785 author's reference implementation. It carries its own parser rather than using `encoding/json`, which fits: it operates on raw bytes, so unknown payload fields survive canonicalization without a struct round-trip losing them. It also **rejects duplicate object keys natively**, which is half of §11.3's parsing requirement for free.

It does **not** limit nesting depth, and it recurses — so the stack-overflow half of that requirement stays the project's own work, exactly as the specification anticipated.

**One implementation trap worth recording before it bites.** Verifying a received event means removing `eventId` and `signature` before canonicalizing. Doing that by round-tripping through `map[string]interface{}` turns every JSON number into a `float64` and silently loses precision on large integers — in a protocol that forbids floats and relies on integers. Use `json.Decoder` with `UseNumber()`, or edit at the raw-JSON level.

The gateway, indexer and client remain unchosen; they speak HTTP and SQL, not libp2p, and nothing about them is blocked by this.

## 14. Related work

### 14.1 The contribution claim was too wide, and the research narrowed it — **honesty fix**

Researching the neighbouring protocols was scheduled early precisely so it could change the design rather than decorate it, and it did — by removing something from the contribution column.

**Moderation as a separate, subscribable stream of labels is not new here.** AT Protocol has [stackable moderation](https://bsky.social/about/blog/03-12-2024-stackable-moderation): independent labelers publish labels, users subscribe to the ones they trust, and the underlying content is never deleted. That is the same design as this project's, arrived at independently. Prior framing in these documents implied it was a contribution; it is not, and the claim is withdrawn.

Converging on it independently is still worth something — as evidence the design is right, not as evidence of originality.

**What the research confirmed instead.** Nostr's relays [deliberately do not replicate to each other](https://nostr.how/en/the-protocol) — no gossip layer, no relay mesh, by design. That makes durability a client-side habit rather than a network property, and it is a real difference from this project, where inter-node replication is most of the cost of the network layer.

**What survived as contribution**, stated narrowly:

- **Authority over data nobody owns.** Every neighbouring protocol assumes each record has an owner. An entity's name has none, which is a problem a social network never has.
- **Weighted claims as the Sybil defence**, per claim type, with the honest note that age cannot carry it.
- **Vote-to-version binding**, which only matters when editable content carries a score.
- **A published determinism boundary.** AT Protocol has the same architecture; we could not find the equivalent written contract.

**The lesson worth keeping:** the wide claim would have survived until someone in a defence asked about Ozone. Doing this before writing code, rather than after, is what the roadmap ordering was for.

## Still open, and owned by the project

- **Payload schemas per event type.** The largest remaining gap before Phase 0 closes. The rating scale is settled (§1.5) because its meaning has to be uniform across the network; the rest of each payload is ordinary schema work and nothing above depends on how it lands.
- **LGPD, Marco Civil, and the illegal-content/PII process.** A dedicated pass, not a paragraph.
- Per-period publish limits and per-node admission defaults.
- Max review character length beyond the 16 KB byte ceiling.
- `expiresAt` for `email` and `human` claims (settled for `domain`).
- Whether an indexer publishes an auditable completeness commitment.
- Vote decay over time, and per-category reputation scoping.

## Things deliberately not built

Recorded so they don't get re-litigated as oversights: timestamp authorities, global consensus on ordering, automatic rollback of entity edits, per-category gossip topics, blocking conflicts of interest rather than surfacing them, protocol-level reputation, and any mechanism requiring a node to be authoritative over anything.
