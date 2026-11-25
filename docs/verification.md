# Verification: How Claims Actually Get Issued

[docs/identity.md](./identity.md#claims-not-tiers) establishes that "verified" is a stack of independent `verification.claim_created` events, not a single flag. This document covers the part that was still hand-waved: how each claim type actually gets its evidence, and why the whole thing holds together even though anyone can publish anything.

## Two kinds of claim, two different defenses

**Self-verifiable** — the evidence is public data anyone can independently check (a DNS record). No third party needs to be trusted; the claim is self-signed by the subject, and a consumer either checks it or doesn't.

**Delegated** — the evidence lives behind a private channel (an inbox, a document) that only whoever ran the check actually saw. The claim is signed by that verifier, and a consumer has to decide whether they trust *that identity's* word for it.

| Claim type | Kind | Signed by |
|---|---|---|
| `domain` | Self-verifiable | The subject itself |
| `cnpj` | Delegated (bootstraps from a `domain` claim) | A verifier |
| `email` | Delegated — deferred past v1 (see [docs/mvp.md](./mvp.md#out-of-scope-for-v1)) | A verifier |
| `human` | Delegated (weak signal) | A verifier |
| `purchase` | Delegated — deferred past v1 (see [docs/mvp.md](./mvp.md)) | A verifier or, later, a merchant |

`author` and `subject` are different fields and the relationship between them is what separates the two kinds. In a **self-verifiable** claim they must be equal — a self-signed claim *about somebody else* asserts nothing and is discarded at projection. In a **delegated** claim they differ by construction: the verifier signs, the subject is named. Note what follows from that — **nobody's consent is required to publish a claim about them.** Anyone can sign an event asserting anything about any identity. That is exactly why recognition, not publication, is where authority lives.

## Why forging a claim doesn't help

Nothing stops anyone from publishing a `verification.claim_created` directly to any node, self-signed, claiming anything about anyone — the protocol is permissionless, and everything a node validates ([the checklist](./protocol.md#validation-checklist-every-node-runs-this-on-receipt)) is about form and authorship, never about truthfulness. That's not a gap, it's the same trust model already used for moderation: **a claim has no authority by itself; authority only exists in whether a consumer chooses to recognize the identity that signed it.**

- For a **delegated** claim, an indexer only gives it weight if `author` is on that indexer's list of verifiers it recognizes for that `claimType` — exactly like a "moderators I follow" list. A self-signed `email` claim from a random key is worthless because nobody trusts that key as an email verifier. It exists on the network; it just doesn't count for anything.
- For a **self-verifiable** claim, trust isn't delegated at all — a forged `domain` claim without a matching DNS record gets discarded by anyone who bothers to check, regardless of who signed it.

The one honest caveat: nothing prevents someone from running a rogue client or indexer that *chooses* to trust garbage verifiers and shows fake "verified" badges to unsuspecting users. The protocol can't fully solve that — it's the same trust problem as installing a malicious browser extension. It lives in which client/indexer someone chooses to use, not in a guarantee the network itself can make.

**What a compromised verifier key actually buys**, since v1 has exactly one verifier and it's worth knowing the blast radius: an attacker holding it can mint `human` claims at will, which is a `claim_weight` of 0.3 per identity ([docs/reputation.md](./reputation.md#eligibility-and-weighting)) — cheaper than farming CAPTCHAs, and bounded by the same ceiling. It buys **nothing** against `domain`, because a self-verifiable claim has no issuer to impersonate: forging one still requires control of the DNS record, and every consumer checks that independently. That asymmetry is the point of preferring self-verifiable claim types wherever one exists, and it's why deferring [`email`](#email--delegated-deferred) also shrinks the consequences of losing this key.

## Issuance mechanics per claim type

### `domain` — self-verifiable

1. The user publishes a TXT record at `_ecoa-verify.<domain>` whose value is exactly `ecoa-identity=<their did:key>`. Multiple records are allowed, so a domain can vouch for more than one identity.
2. The user publishes a self-signed `verification.claim_created`:
   ```json
   {
     "claimType": "domain",
     "subject": "did:key:user",
     "entity": "domain:example.com",
     "evidenceRef": "dns-txt:_ecoa-verify.example.com",
     "expiresAt": "2027-08-30T00:00:00Z"
   }
   ```
   `subject` equals the envelope's `author` here, because `domain` is self-verifiable — see [the payload spec](./protocol.md#verification) for the full field list.
3. Any indexer or client looks up that record and confirms the claim's `subject` appears in it. No verifier identity is involved at any point.

**No challenge nonce, deliberately.** A nonce is the right shape for a one-time challenge, and this is the opposite: the record has to stay in place and stay re-checkable, because the claim is live state re-verified on a TTL (below). Control of the domain *is* the evidence, and the record has to prove one thing only — that whoever controls this domain vouches for this identity. A nonce would add a step, add a way to get the implementation subtly wrong, and prove nothing extra.

A file at `https://<domain>/.well-known/ecoa-verify` containing the same `ecoa-identity=<did:key>` line is an equivalent alternative for anyone who can't edit DNS. It's strictly weaker — it depends on TLS and on the web server rather than on the zone — so an indexer that supports both should prefer the DNS record when both exist.

**A `domain` claim is live state, not a replayable fact.** The event asserts *"there was matching evidence when I published this"*; what a consumer acts on is its own lookup, right now. A domain expires, gets sold, or has its TXT record removed, and the same event that verified last month stops verifying today — with no revocation event anywhere, because nothing about the event changed.

Three consequences, all deliberate:

- **A failed lookup downgrades, it doesn't revoke.** The claim event stays valid and stored; what changes is the indexer's *observation* of it. Revocation ([below](#revocation)) is a statement by the issuer; a stale DNS record is not.
- **The result is a local observation**, the same category as `receivedAt` in [docs/protocol.md](./protocol.md#what-createdat-can-and-cannot-be-used-for): unsigned, per-indexer, re-derived rather than replayed, and legitimately different between two honest indexers.
- **Claims carry `expiresAt`**, and an indexer re-checks on a TTL rather than trusting a check it made once. Suggested v1 defaults: `expiresAt` no more than **one year** out, re-checked at least every **30 days**. Without an expiry, a claim published once would assert domain control forever on the strength of a record that stopped existing years ago.

If replay determinism ever becomes necessary, the way to get it is a signed *observation* — some party publishing "I saw this TXT record at time T" — so a consumer that wants reproducibility replays attestations while one that wants freshness keeps querying DNS. That's a strictly additive event type, so it stays out of v1.

#### The lookup itself is a trust dependency

Plain DNS is unauthenticated. An indexer resolving `_ecoa-verify.example.com` through a hostile resolver, or with an attacker on the path, can be shown a record that was never published — and it will then treat a forged claim as verified, badge and all. The claim model's promise that "anyone can check it themselves" is only as good as the channel they check it over.

- **Resolve over DNS-over-HTTPS** to a resolver the operator chose deliberately, rather than whatever the host's default resolver happens to be.
- **Prefer DNSSEC-validated answers** where the zone is signed, and treat an unsigned answer as the weaker evidence it is.
- **The `.well-known` fallback inherits the web PKI** instead — a different trust root with different failure modes, which is part of why it's marked as strictly weaker.

An indexer's DNS view is therefore one of its trust dependencies, in the same category as its verifier list. Two indexers can honestly disagree about whether a domain claim verifies, which the [determinism boundary](./architecture.md#what-must-be-identical-across-indexers-and-what-must-not-be) already accounts for.

**v1, as built** (`indexer/internal/claim`): `indexer project` looks up only self-signed claims on a `domain:` entity, over DNS-over-HTTPS to a resolver set in `DOH_URL` (Cloudflare's by default), or over plain DNS to a server set in `DNS_SERVER` — how the local network checks its `.test` zone. It records whether the resolver validated DNSSEC, and does not yet require it. A pass holds 30 days before the next lookup; a failure is looked up again after 30 seconds, so a record published after the claim counts within a minute. The lookup is observed state (`claim_checks`), never replayed by a rebuild. The `.well-known` fallback is not built.

#### An expiring domain transfers authority

Domain control is a lease, not a property. When `example.com` lapses and someone else registers it, the new owner can publish the TXT record, self-sign a claim, and become the verified representative of `domain:example.com` — with tier-1 authority to rewrite the entity's metadata and to post official-looking responses under every historical review of that company.

This is inherent to proving control of a domain and is not solvable here; ACME and the whole web PKI live with the same property. Expiry and re-checking (above) retire the *old* claim, but nothing stops the *new* one, because the new one is true: that identity really does control the domain now.

The mitigation is presentational, and it belongs to whoever runs an indexer:

- Show **since when** a representative has been verified, not merely that they are. (v1 does: `verified_since` resets when a lookup fails, and the entity page shows it.)
- Treat a **change of verified representative** for an entity as a visible event rather than a silent swap, especially when the previous claim lapsed rather than being revoked. (Not built in v1.)
- Remember that a bad edit is never destructive: every previous value is retained, and an indexer can fall back to it ([docs/entities.md](./entities.md#verified-representative-edits-are-authoritative-not-unaccountable)).

### `cnpj` — delegated, bootstraps from `domain`

There's no public, self-checkable way to prove "I control this CNPJ" the way DNS proves domain control, which is why this claim type is deferred past v1. When it ships, the shape is: the company first proves its `domain` claim (above), then a verifier manually confirms that domain plausibly belongs to that CNPJ — public registry lookup, incorporation documents, whatever's available — and signs:

```json
{
  "claimType": "cnpj",
  "subject": "did:key:company",
  "entity": "cnpj:12345678000199",
  "expiresAt": "2028-08-30T00:00:00Z"
}
```

`evidenceRef` is **omitted**, not null — the evidence is a private document nobody else can check, so a locator would only advertise that it exists.

### `email` — delegated, deferred

Out of scope for v1: it needs a verifier service, mail sending, deliverability and templates — real infrastructure, and `human` already separates a fresh key from a slightly less fresh one for a fraction of the effort. Recorded here because the mechanics are settled and the deferral is about cost, not about doubt.

1. Client sends the address to a verifier service over ordinary HTTPS (outside the P2P/event flow entirely).
2. Verifier emails a one-time code; the user's client signs a challenge nonce with their key at the same time, binding *this* identity to *that* inbox.
3. Verifier issues the claim once satisfied:
   ```json
   { "claimType": "email", "subject": "did:key:user", "expiresAt": "2028-08-30T00:00:00Z" }
   ```

**Privacy note:** the event never carries the email address or a hash of it — that would be permanent, replicated PII (see [docs/identity.md](./identity.md#privacy)). The public event carries the bare assertion and nothing else.

Whether the verifier retains the address at all is a policy choice with a consequence worth naming: whatever it keeps is the only thing on the whole system linking a pseudonym to a person, which makes the verifier the natural target of any legal demand for an author's identity ([docs/mvp.md](./mvp.md#legal-exposure)). The safest default is to keep nothing after issuance — verify, sign, discard — and to treat any retention as a deliberate decision that needs its own justification.

### `human` — delegated, weak signal

A CAPTCHA (or similar low-friction challenge) at first publication, issued by the gateway/API acting as verifier. Cheap to ship, gives a small weight bump above a completely fresh key without requiring an email.

**v1, as built:** the challenge is a proof of work (ALTCHA) the gateway hands out and the browser solves, so no third party sees the user — a hosted CAPTCHA would ([decisions.md §16.4](./decisions.md#164-claims-the-indexer-checks-domains-itself-the-gateway-issues-human-claims-behind-altcha--call)). The user asks for it on the Verification screen rather than at first publication. Each challenge is accepted once; the gateway then signs a `human` claim about the asking `did:key` with its verifier key (`HUMAN_VERIFIER_SEED`), valid two years, and publishes it. It counts at an indexer that lists that key in `VERIFIERS_FILE`. Since v1 weighs nothing yet, a `human` claim shows as a mark beside the author and changes no score.

### `purchase` — delegated, deferred

Out of scope for v1 (see [docs/mvp.md](./mvp.md)). Sketch for later: the claim targets a specific review event (`targetReview`, not just `subject`), evidence is a manually-reviewed receipt/document upload, and eventually a merchant could run its own verifier identity and issue these claims directly for real transactions — no protocol change needed, just a new verifier a client can choose to trust.

## Where this runs

The actual checks (sending an email, looking up DNS, reviewing a document) are **not a node responsibility** — the node only ever validates signature and schema for any event, including claims. Issuance is a separate, ordinary service (part of the API, or a dedicated verifier service) that talks to the user's client over normal HTTPS, and at the end just publishes a signed event like any other.

In v1 the gateway issues `human` claims and the indexer checks `domain` claims; the two flows are drawn in [diagrams.md §15](./diagrams/diagrams.md#15-claims-as-v1-issues-and-recognizes-them).

## Revocation

`verification.claim_revoked` is signed by whoever issued the original claim — a verifier can retract its own word. The subject can't force a third party's claim about them to be revoked, but they're also never required to reference or rely on a claim they don't want.

## Open questions

- Re-check *intervals* for the delegated claim types. Maximum lifetimes are settled — one year for `domain`, two for the rest ([docs/protocol.md](./protocol.md#verification)) — and `domain` re-checks at least every 30 days. `email` and `human` have no cheap re-check available, so in practice they simply expire; whether anything better is worth building is open.
- Whether a public, queryable CNPJ↔domain government API exists that could turn `cnpj` into a self-verifiable claim like `domain` already is.
- Whether `evidenceRef` ever needs to carry more than it does now. Settled for v1: it's a scheme-prefixed locator naming *where* the evidence lives, never the evidence itself — `dns-txt:<name>` or `https://<url>` for self-verifiable claims, and `null` for every delegated claim, where the evidence is private by construction and a locator would only leak that it exists.
