# Identity, Trust & Security

## Identity model

On first use, the client generates an Ed25519 keypair locally. The public key *is* the identity:

```
did:key:z6Mkabc123...
```

No name, email, phone, or national ID is required. Publishing = signing an event with the private key; anyone can verify it with the public key. The network is pseudonymous by default.

A user profile is itself just another signed event (`profile.updated`), not a row in someone's user table.

## Web account vs. crypto identity

The site can offer an optional conventional account (email/password, passkey, social login) — but it's a convenience layer, not the source of authority:

```
Web account  →  associated with  →  Cryptographic identity
```

Its job is syncing preferences, holding an encrypted backup blob on the user's behalf, and enabling multi-device notifications. If the site disappears, the cryptographic identity is still valid on any other compatible client.

Note the precise limit of "aiding recovery": an account can *store* something the user already encrypted, and it can never reconstruct a key it was never given. A user who loses their key and their backup cannot be restored by any amount of account access, and no support process can change that. (v1 doesn't ship this account layer at all — see [docs/mvp.md](./mvp.md#out-of-scope-for-v1).)

## Roles

There's no central `role` field anywhere in Ecoa Network — no admin panel that flips a user to "moderator" or "verified." Every role is the result of a client or indexer **interpreting a set of signed events** about an identity. Same split used for reputation and moderation elsewhere: the protocol stores facts, every consumer decides how to interpret them.

There are two independent axes — **content roles** (who's publishing/interacting) and **infrastructure roles** (who's running the network). The same person can hold both.

### Content roles

| Role | Requires | Can do | Who vouches for it |
|---|---|---|---|
| Visitor | Nothing — no keypair | Read, search, browse public profiles | N/A — not even a protocol concept, just a UI state |
| Pseudonymous identity | An Ed25519 keypair (+ optional `profile.updated`) | Publish reviews, respond, report | The signature alone |
| Verified identity | One or more **recognized** `verification.claim_created` events — issued by a verifier the consumer trusts, or self-verifiable and actually checked | Same as pseudonymous, with weight in reputation and voting scaled to the strongest claim held | The consumer's own recognition — see *Claims, not tiers* below |
| Company representative | A recognized claim proving control of the entity's own identifier. In v1 that means `domain`; `cnpj` is deferred | Post official responses, update public entity info for **that identifier** | Same recognition mechanism, applied to entity control — see [docs/entities.md](./entities.md#1-verified-representative--authoritative-by-default) |
| Moderator | No protocol requirement — any key can publish `moderation.label_created` | Label events (spam, off-policy, illegal, etc. — deliberately *not* "disputed", which is a community vote and a different system) | The client/indexer's own "moderators I follow" list — trust is asserted by the consumer, not granted by the protocol |

New identities aren't blocked from any of this by default — they just start with less weight in reputation and voting until they hold a claim worth something. That's the Sybil mitigation: friction through weighting, not gatekeeping. Note *which* input carries it — a claim, which has an issuer and replays identically, rather than accumulated history, which rests on a self-attested clock ([docs/protocol.md](./protocol.md#what-createdat-can-and-cannot-be-used-for)).

### Claims, not tiers

"Verified" isn't a single flag. Each fact is its own claim — email confirmed, purchase confirmed, domain/CNPJ control confirmed, human-ness confirmed — expressed as a `verification.claim_created` event (see [docs/protocol.md](./protocol.md#event-types-v1-candidates)) carrying a `claimType` field. Depending on the type, it's signed either by a verifier or by the subject itself; [docs/verification.md](./verification.md#two-kinds-of-claim-two-different-defenses) covers which, and why the difference decides what a consumer has to trust. A claim can be revoked independently (`verification.claim_revoked`) without touching any other claim held by the same identity.

This matters for two reasons:

- **No single "verified" gate to game.** Compromising or faking one claim type doesn't inherit the others.
- **Verifiers are pluggable, like moderators.** Day one the project is the only *delegated* verifier — `email` and `human` claims are signed by an identity it controls. `domain` is the exception and always will be: it's self-verifiable, so the subject signs its own claim and anyone can check the DNS record without trusting a verifier at all. Nothing in the format prevents a second, independent verifier from showing up later; consumers just decide whose claims they recognize.

This also removes the need for a separate "registered user" concept in the voting system. Eligibility isn't a gate — everyone can vote — it's a weight, and it scales with the strongest claim an identity holds rather than with a yes/no flag. See [docs/reputation.md](./reputation.md#eligibility-and-weighting) for the numbers and for why a binary version of this would have been a cheap thing to farm.

For how each claim type actually gets issued — and why publishing a fake one doesn't get you anywhere — see [docs/verification.md](./verification.md).

### Infrastructure roles

Not about authorship — about who runs which piece of software. Visible through configuration and network topology, not signed claims:

- **Node operator** — runs a node implementing the protocol; has its own libp2p peer identity, separate from any content-author identity.
- **Bootstrap operator** — a node additionally configured as a known entry point for peer discovery. Not a separate role, just a config flag.
- **Indexer operator** — runs the event consumer and index, with its own inclusion/moderation policy.
- **Gateway/API operator** — runs the HTTP layer in front of a node for light clients.
- **Verifier** — issues `verification.claim_created` events for one or more claim types, under its own identity. Listed here because it's infrastructure rather than authorship, but note it's the one infrastructure role that *does* own a content-signing key. In v1 the gateway operator is also the verifier (it issues the `human` claim from the CAPTCHA it already runs), which is convenient and worth being explicit about rather than letting it blur: they are two roles that happen to share an operator, and a client's decision to trust a verifier should never be inherited from its choice of gateway.

See [docs/architecture.md](./architecture.md#multiple-operators) for how these compose in practice.

## Key storage (client-dependent)

Where and how the private key is stored depends entirely on which client is holding it — a browser, a mobile app, a desktop app, and a CLI all have different platform primitives for this, and that choice hasn't been made yet (see [docs/mvp.md](./mvp.md)). What's universal, regardless of platform:

- The key is generated and stays on the user's device. It is never sent to a gateway, API, or any server.
- Prefer whatever the platform's non-exportable key primitive is, if it has one — so application code can request a signature but never extract the raw private key bytes.
- Never log the key, never put it in a general-purpose storage location shared with other app data without encrypting it first.

As an illustration of what this looks like for one possible client — a browser-based web frontend — the platform's Web Crypto API can generate a non-extractable `CryptoKey` (`extractable: false`) and hold it in IndexedDB, origin-isolated by the same-origin policy. This is one concrete option, not a commitment to building a web frontend a particular way.

### What "non-extractable" does *not* protect against

This part *is* universal, independent of platform: if an attacker gets arbitrary code running in the client's context (XSS in a web app, a compromised dependency, a compromised script/CDN), that code can still *request* signatures through the app's own signing function — it just can't steal the raw key. The real boundary is **which code is allowed to ask for a signature**, not just where the key lives.

Mitigations, platform-agnostic in principle even though the exact mechanism varies by client:

- Show the user exactly what they're signing, and require explicit confirmation before every signature.
- Keep the signing code path's dependencies minimal — fewer places for a supply-chain compromise to hide.
- Never render review/user content as raw markup — treat it as plain text, whatever the platform's equivalent of avoiding `dangerouslySetInnerHTML` is. **Every string that reaches a client from the network is user content**, without exception: review bodies and the `reason` on a vote, but equally profile display names, entity names, categories and descriptions, moderation label text, and anything an indexer seeded from a public dataset. An entity name renders in a page heading, which makes it the highest-value injection target in the whole system and the one most likely to be treated as trustworthy by mistake.
- For a web client specifically: a strict CSP (`script-src 'self'`, no inline scripts, no third-party analytics/widgets on pages that touch identity) meaningfully shrinks the attack surface.
- Optional, longer-term, also web-specific: isolate signing into its own subdomain with a much smaller attack surface than the main app.

## Recovery and multi-device

A non-extractable key can't be exported, which leaves two ways to survive a lost or stolen device:

1. **Multiple keys under one identity** — each device holds its own keypair, authorized under the same identity by signed events, so no private key ever crosses a network. This is the root/device model specified below, and it's the one that also answers key *theft* rather than only key *loss*.
2. **Encrypted backup** — an exportable seed, always encrypted before it leaves the device (password or recovery key). Simpler to reason about, and it covers loss, but it covers nothing about compromise: whoever gets the seed gets the identity. It also shifts the responsibility, and the attack surface, onto the user.

These aren't really alternatives — a root/device identity still wants its root key backed up. The section below specifies the first; the second is a client concern with no protocol surface.

### Key rotation

Rotation is itself a signed event, letting an identity move to a new key while keeping its history:

```json
{
  "successor": "did:key:new"
}
```

One field. The identity being rotated away from is the envelope's `author`, and the event type is the envelope's `eventType`, so naming either again would be duplication ([docs/protocol.md](./protocol.md#conventions)).

Signed by the **old** key — nothing else could authorize its own succession. Three rules make that safe to rely on:

**One successor, ever, and the first one observed wins.** An identity has at most one valid successor for its entire life. A node that has already accepted a rotation **rejects any later conflicting one outright**, whatever `eventId` it carries. Successors chain transitively; a rotation naming an identity already in the chain is a cycle and is rejected.

This is the same rule already applied to revocation ([docs/protocol.md](./protocol.md#when-rotation-and-revocation-take-effect)), and for the same reason: the general ordering rule says that anything an attacker holding the author's key would want to move is a local observation, and a rotation is exactly such a thing.

**Why not the lower `eventId`.** An earlier version resolved conflicts by taking the lower `eventId`, on the reasoning that it was a deterministic tiebreak with no clock in it. It was — and it was also a retroactive identity hijack:

1. Alice rotates `key_old → key_new` in January. The whole network accepts it.
2. Alice publishes for six months under `key_new`.
3. In July, Mallory obtains an old backup of `key_old` — a decommissioned laptop, a leaked archive.
4. Mallory publishes `key_old → key_mallory`, varying `createdAt` until the `eventId` falls below January's.
5. Under a global lowest-`eventId` rule, Mallory's rotation becomes canonical and Alice's becomes invalid — retroactively. Six months of Alice's events are now signed by a key with no relationship to the identity, and Mallory holds it.

The grinding is free: to land an `eventId` below a given random 256-bit value takes about two attempts on average. It's arithmetic, not work. And note the shape of the flaw — a rule with no clock in it turned out to have no *history* in it either, which let a stale key reach back past six months of accepted fact.

**When a node genuinely can't order them.** A node back-filling history receives both rotations at once and has no "first observed". Rather than pick, it **fails closed: the identity is contested and neither successor is canonical.** History under the original key stays readable and verifiable; no new event is accepted under either successor until a human resolves it out of band.

Mallory can therefore still *freeze* Alice's identity, which is bad. Mallory cannot *become* Alice, which would be catastrophic. Given a stolen key, those are the only two options on offer, and choosing the survivable one is the whole point.

**The cost, stated plainly:** two nodes can hold different views of a contested identity depending on what each saw first. That's the same trade revocation already makes, and it's the correct one — the alternative hands the attacker the power to place the cut themselves. It is also one more argument for the root/device model below: a cold root is far less likely to leak in the first place.

**Revocation without a successor.** `identity.key_revoked`, also signed by the key itself, says *"this key is compromised, trust nothing signed by it from here on"* and names no heir. For the legitimate owner it's the difference between watching a thief inherit their history and publicly killing the identity, leaving a signed tombstone anyone can see, and starting again.

An attacker holding the key can publish it too, and it's worth being honest that *"they gain nothing"* is only true of an attacker who wants the identity. An attacker who wants the identity **silenced** gains exactly what they came for: revocation is an irreversible denial of service against its own subject, with no un-revoke. On a platform whose whole point is publishing things companies would rather suppress, that is a motive worth taking seriously rather than waving past. It is an accepted risk, mitigated only by not letting the key leak — which is, again, the argument for a cold root.

**Without a recovery path, compromise is terminal.** That's the plain truth for a single-key identity and the docs should not dress it up: whoever holds the key *is* the identity. There is no appeal, because there is no authority to appeal to — that's the same property that stops anyone from seizing an identity they don't control.

### Root key and device keys

The way out of "compromise is terminal" is to stop having one key do two jobs. A **root identity key** stays cold and signs only authorization, rotation, and revocation; **device keys** stay hot and sign ordinary content. `did:key:root` is then the identity; a compromised device key is revoked by the root, and the history survives.

This subsumes the per-device-identity and encrypted-backup options above into one mechanism instead of two. It needs two event types (`identity.device_authorized`, `identity.device_revoked`) and one envelope field, `deviceKey`, which names the key that actually produced a signature while `author` stays the identity ([docs/protocol.md](./protocol.md#signing-key-vs-identity)). A content event signed by a device key is valid only if that key was authorized, and not since revoked, by the root.

**The envelope field ships in v1 regardless**, set to `null` by every single-key identity. That's the part that can't be deferred: the envelope is a closed set, so introducing `deviceKey` later would be a breaking version bump for the entire network, while shipping it nullable now costs one `null` per event. The *behavior* is optional in v1; the *room for it* is not.

A useful side effect of `author` remaining the identity: swapping device keys never fragments a history, and never breaks the authorship rule that lets an identity edit or withdraw what it published from a device it no longer has.

**In v1 the shape is fixed and the behavior is the single key.** An identity is one key (simple, and terminal on compromise); a node rejects any event with `deviceKey` set, and the root with device keys (safer, two keys to manage) is for later ([decisions.md](./decisions.md#35-v1-ships-the-single-key-model-and-rejects-any-devicekey--call)). Settling the shape now rather than later is not premature: retrofitting it would change how *every* event is verified, which the versioning rules classify as a breaking envelope bump — the one category of change this project has committed to making rare and deliberate. What v1 must not do is ship a single-key model that assumes it will always be the only model.

## Trust boundary: who validates what

```
Client                    Gateway                  Node
├─ builds event           ├─ checks size/shape     ├─ recomputes eventId
├─ shows exact content    ├─ rate limits           ├─ validates schema
├─ signs locally          └─ forwards              ├─ verifies signature
└─ never sends the                                 ├─ resolves references
   private key                                     ├─ applies admission policy
                                                   ├─ assigns local sequence
                                                   └─ stores and replicates
```

The client's own signature check is for UX/transparency. The node's check is the one that actually protects the network — nothing downstream trusts a claim like `"signatureValid": true` coming from the client.

## Privacy

- No real name, CPF/national ID, phone, or email required to publish.
- Before publishing, the UI should warn that events, once replicated, may persist on third-party nodes indefinitely.
- The product should actively discourage users from posting personal data (IDs, addresses, bank info, medical info, credentials) in review content or attachments.

### Pseudonymous is not anonymous

`author` is stable across everything an identity ever signs, and every event is public and permanently replicated. So one identity's entire history — every company they complained about, every endorsement, the timing of all of it — is trivially linkable by anyone, forever. For most reviews that's fine and is exactly the point: a history is what makes an identity worth weighting.

For the case this platform explicitly wants to serve — an employee describing their employer, a customer describing a company with lawyers — it's a real hazard, and one that gets worse the longer someone uses a single identity well. A few complaints about employers in the same small city, cross-referenced against a public LinkedIn, is not a hard deanonymization.

The protocol already permits the mitigation: **an identity is free, and nothing requires a person to use only one.** A throwaway identity per sensitive review is fully supported. The cost is precise and unavoidable — a fresh key has `claim_weight` 0.1 and no age, so it publishes with minimal weight, which is the same property that makes Sybil flooding pointless. **Weight and unlinkability are the same trade-off seen from two sides, and no configuration escapes it.**

Two things follow for any client. It should make publishing under a fresh identity a visible, one-tap option rather than something a user has to understand the protocol to discover; and it should say plainly, at that moment, what is being traded. What a client must *not* do is silently reuse the default identity for everything and let a user discover the linkage after the fact.

### And the network layer knows more than the protocol does

Everything above is about linkability *between events*. There is a second, larger exposure that no amount of key hygiene touches: **publishing reveals an IP address.**

- The **gateway** sees the source IP alongside the signed event it is forwarding. That is enough to link an identity to a network location, and in v1 the project runs the only gateway.
- A **node** sees the IP of whichever peer handed it an event, and a node reachable directly sees the publisher's.
- Publishing under a fresh identity from the same session and address, minutes after using the main one, correlates the two immediately — which quietly defeats the mitigation recommended just above.

This is the most dangerous promise in the project to leave unqualified, because it fails exactly where the docs claim to be most useful: an employee describing their employer is protected from *readers* by pseudonymity and not at all from anyone holding the gateway's logs.

What v1 can honestly do:

- **A gateway must not log source addresses alongside event content.** Rate limiting needs a counter, not an audit trail — and a counter can be keyed on a short-lived salted hash that is useless afterwards.
- **Say so in the interface**, at the moment of publishing, in the same breath as the warning about permanence. A user who knows the limit can bring their own mitigation; a user told they are anonymous cannot.
- **Real network anonymity is out of scope.** Tor or a comparable transport is the only genuine answer, and pretending otherwise would be worse than admitting it. A client that supports a proxy at all should make it reachable rather than hidden.
- **Fresh identity, fresh session.** A client offering throwaway identities should say that changing the key does not change the address, and encourage the two to be separated.

## Anti-abuse (Sybil resistance)

A valid signature only proves *a* key signed something — it says nothing about whether that key is trustworthy. Mitigations, roughly in order of when they'd land:

**Early:** reputation weighted primarily by the recognized claims an identity holds and secondarily by observed age (see *Roles* above), API rate limiting, size/frequency limits, CAPTCHA on public gateways, duplicate-content detection, local peer blocking.

**Later:** lightweight proof-of-work for high-volume posting, community invites, external attestations, social-graph-based reputation, coordinated-campaign detection.

Reputation itself is **not** a protocol-level truth — the protocol only stores raw facts: which events an identity published, which claims it holds, which endorsements and disputes it drew. Note what is *absent* from that list: **identity age is not a protocol fact.** Timestamps are self-attested, so age is a local observation each node makes rather than something the network can attest to ([docs/protocol.md](./protocol.md#what-createdat-can-and-cannot-be-used-for)). Each indexer computes its own reputation score from the facts plus its own observations, so different indexers can apply different anti-abuse philosophies without forking the network.

### Where these controls actually sit

Almost everything in the "Early" list lives at the gateway — and **the gateway is optional**. A node speaks libp2p directly, so anyone willing to run one publishes without ever passing a rate limiter or a CAPTCHA. Gossipsub peer scoring (see [docs/protocol.md](./protocol.md#peer-abuse--flood-protection)) punishes a peer that sends *invalid* traffic; it does nothing about a peer flooding perfectly valid, perfectly signed garbage from ten thousand fresh keys.

So gateway-side controls raise the cost of casual abuse and are worth having, but they are not what defends the network. Two other things carry that weight:

- **Per-node admission policy.** Validity is protocol-level; *acceptance* is local. A node may cap events per author per period, cap how much it takes from identities with no recognized claim, or refuse anything below a local threshold — and simply not store or relay what exceeds it, exactly the latitude it already has for stricter size limits and unrecognized event types. Nothing propagates through a node that won't relay it, so a policy adopted by well-connected nodes shapes the network without any protocol change.
- **Weighting at read time.** Flooding costs an attacker almost nothing, but it also buys almost nothing: an indexer ranking by weighted reputation shows a million zero-weight events to nobody. Spam that reaches storage but never reaches a reader is a storage-cost problem, not a visibility problem — a real cost to node operators, and a much smaller one than it looks from the publish side.

Proof-of-work stays in the "Later" bucket, but it's the only control listed anywhere here that would make *publishing itself* cost something at the node layer. It's what to reach for if per-node policy proves insufficient.
