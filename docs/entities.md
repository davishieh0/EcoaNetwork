# Entities

An entity is whatever a review is about — a company, product, service, place, or other subject. Entities differ from reviews in one important way: a review is a personal fact that belongs to whoever signed it, but an entity's metadata (name, category, description) is a shared fact that different identities might describe differently. That difference is what this document covers. For the wire format — identifiers, event types — see [docs/protocol.md](./protocol.md#identifying-entities).

## Creating an entity needs no permission

An entity's existence isn't gated by anyone's approval. The identifier itself — `cnpj:12345678000199`, `domain:example.com`, `gtin:07891234567895` — is self-describing. The first review that references it is enough for the entity to exist in an indexer's projection; no prior event is required, and there is no `entity.created` — an `entity.updated` only ever supplies optional descriptive metadata (a suggested name, category, description), never permission to exist.

Entities without a public identifier (`entity:<hash>`) are the one place duplicates are a real risk — two people describing the same physical place slightly differently generate different hashes. That's handled by `entity.alias_proposed`, which is weighted as an ordinary metadata proposal under the rules below. **No v1 indexer projects aliases** ([docs/mvp.md](./mvp.md#out-of-scope-for-v1)) — the event type stays in the protocol so publishing one is never wasted, but grouping two entities is deferred work, and until then duplicates simply coexist.

## Names are an impersonation surface

An entity's identifier is unforgeable — `cnpj:12345678000199` is that company and no other. Its *name* is a display string that anyone can propose, and display strings are where impersonation lives.

Unicode normalization doesn't help here. NFC settles how a character is composed, not which character it is: `Аmazon` written with a Cyrillic `А` normalizes to itself and renders identically to the Latin one. So an attacker can create a visually identical entity, accumulate reviews under it, and readers can't tell by looking.

This is an indexer concern rather than a protocol one — the events are all valid — and the defenses are the usual ones for confusables:

- Flag or reject names mixing scripts that have no business appearing together, the way registrars and browsers handle IDN homographs.
- Compare a proposed name against existing entities under a confusable-folding (skeleton) mapping, not just exact equality, and surface a near-collision instead of silently creating a twin.
- Prefer showing the identifier alongside the name for anything consequential — an identifier can't be spoofed, and a reader who can see `cnpj:` knows what they're looking at.

The same applies to profile display names, which are subject to no verification at all: a profile calling itself an official company account is exactly as valid as any other, and a client that renders a display name in a way implying verification is doing the impersonation for the attacker.

## The real problem: editing shared metadata

Nothing stops any identity from publishing `entity.updated` for any entity. Two identities can publish contradictory versions of the same company's name, category, or description — both cryptographically valid, both signed correctly. Unlike a review, there's no natural owner of that signature to defer to. Something has to decide which version becomes the canonical projection.

## Edit authority model

Two tiers, reusing mechanisms already defined elsewhere rather than inventing new ones.

### 1. Verified representative — authoritative by default

If the identity publishing `entity.updated` holds a **recognized** `verification.claim_created` proving control of that entity's domain or CNPJ (the "company representative" role from [docs/identity.md](./identity.md#content-roles)), the indexer applies the edit as canonical without further confirmation:

```
entity.updated { entity: "domain:example.com", name: "Example Ltda" }
  from did:key:company
  + verification.claim_created { claimType: "domain", entity: "domain:example.com" }
    — self-signed by the same identity, DNS record checked by this indexer
  → applied as canonical
```

**Recognized** is doing real work in that sentence, and it means exactly what [docs/verification.md](./verification.md#two-kinds-of-claim-two-different-defenses) already says it means: a *delegated* claim type like `cnpj` counts only when signed by a verifier on this indexer's verifier list, and a *self-verifiable* type like `domain` counts only when the indexer actually checked the evidence itself. A self-signed `cnpj` claim is not a tier-1 credential — anyone can publish one, and it grants nothing. Without this filter, tier 1 would be forgeable by any key willing to assert anything about itself, which is precisely what the claim model exists to prevent.

**A claim grants authority over the identifier it names, and nothing else.** A `domain` claim for `domain:example.com` makes its holder the verified representative of *that* entity. It does not reach `cnpj:12345678000199`, even when both plainly refer to the same company, because nothing in a DNS record says which CNPJ is behind it — asserting otherwise would let anyone holding any domain claim seize any company.

This is the practical shape of tier 1 in v1, and it's narrower than it first looks. `cnpj` verification is deferred ([docs/mvp.md](./mvp.md#out-of-scope-for-v1)), so v1 tier-1 authority exists **only for `domain:` entities**, while many reviews will reference `cnpj:` ones. Three things keep that workable rather than broken:

- Where a company has a website, `domain:` is the better identifier for a client to steer toward anyway — it's the one a person recognizes and the only one with a self-serve verification path.
- `cnpj:` entities are seeded from public Receita Federal data (below), so they carry correct names without needing an owner to claim them.
- A `domain`-verified representative can publish `entity.alias_proposed` linking their domain to a CNPJ. That proposal carries their weight, which is high, but **not** tier-1 authority over the CNPJ side — it's an ordinary weighted proposal, resolved like everything else in tier 2. Strong evidence, not a grant.

The honest v1 limitation that remains: reviews about one company can land on both `domain:` and `cnpj:` entities and stay split until an alias earns enough weight to group them. That's a real cost of not shipping `cnpj` verification, and it's a better cost than shipping a verification path that could be forged.

### 2. Unverified edit — a proposal, not a fact

An `entity.updated` from an identity with no recognized claim is still stored — it's a valid signed event, nothing is rejected — but it doesn't automatically become the canonical projection. It's a proposal, and proposals are resolved **per field**, not per event.

For each metadata field independently (name, category, description, …), the canonical value is the one set by the proposal with the highest weight. Two inputs, in order:

1. **Votes on the proposal**, if any — `review.endorsed` / `review.disputed` applied to the `entity.updated` event instead of to a review, weighted exactly as in [docs/reputation.md](./reputation.md#eligibility-and-weighting).
2. **The author's own identity weight**, when there are no votes — which is the overwhelmingly common case, and the reason this can't be a voting mechanism alone.

Ties break on the earlier proposal, then on the lower `eventId`.

That makes the *rule* deterministic, but not the outcome: weight depends on `claim_weight` and observed age, both of which are local observations ([docs/architecture.md](./architecture.md#what-must-be-identical-across-indexers-and-what-must-not-be)). Two honest indexers apply the same rule to the same proposals and can still land on different canonical values. That's expected rather than broken — it's the same reason two indexers can rank reviews differently — but it does mean canonical entity metadata is a per-indexer projection, not a network fact, and nothing downstream should treat it as one.

Resolving per field rather than per event matters: a proposal that fixes a wrong category shouldn't have to also re-assert a name that was already right, and a good correction to one field shouldn't drag a bad value in another along with it.

**Why author weight and not just votes.** Nobody votes on a company's category. Ranking proposals purely by votes means ranking by zero against zero, and the tiebreak — whatever it silently ended up being in the implementation — becomes the real rule. Almost always that's last-write-wins, which hands every entity on the platform to whoever edited most recently. Falling back to the author's weight reuses machinery that already exists and gives the sensible answer with no participation at all: a fresh anonymous key (weight 0.1) does not overwrite a value set by an established identity, while an anonymous correction to an obviously wrong name still wins against nothing and can still be voted up past an incumbent.

**Incumbency is sticky.** A challenger must exceed the current canonical value's weight by a margin — **20%** is the suggested v1 default — rather than merely tie or edge past it. Without hysteresis, two near-equal proposals make an entity's name flip back and forth every time weights drift, which reads as vandalism even when nobody intended any.

**Seed the entities instead of arbitrating them.** The most effective thing an indexer can do here isn't a better conflict rule — it's making conflicts rare. Brazilian company data is public and bulk-downloadable from the Receita Federal, so an indexer can seed `cnpj:` entities with correct names and categories before anyone edits anything. Seeded values enter as ordinary tier-2 proposals authored by the indexer operator's own identity, competing under the same rules as everyone else's rather than as a privileged back door. Most entities then arrive already correct, and the edit-conflict path becomes the exception it should be instead of the default path every entity travels.

**v1, as built:** `indexer/cmd/seed` reads the Receita Federal's CSV layout (`Estabelecimentos` and `Empresas`, active establishments only) and publishes each entity's name, category (mapped from its CNAE) and a short description as an `entity.updated` signed by an operator key, through a node like anyone's event. The repository carries a synthetic sample in that layout; the real monthly dump is passed by path ([decisions.md §16.12](./decisions.md#1612-the-seeded-dataset-is-synthetic-in-the-receitas-layout--call)). Until weights exist, the first proposal of each field holds, so seeded values win by arriving first, which is exactly what seeding is for.

Worth keeping the stakes in proportion: entity metadata is a company's name and category, not the review text. It deserves a rule that is deterministic and hard to vandalize. It does not deserve to be over-built.

## Claim conflicts

Two identities can both publish a `verification.claim_created` claiming control of the same `cnpj:`/`domain:` — the protocol doesn't prevent it, since a claim is a signed assertion and not a grant. Most such pairs are not conflicts at all: only *recognized* claims (above) can conflict, so a self-signed claim competing with a verifier-issued one is simply ignored, not a tie.

A genuine conflict — two recognized claims on the same entity — is rare and usually mundane: a stale claim nobody revoked, a verifier that made a mistake, or two verifiers that disagree. **Demoting the entity to tier 2 whenever one appears would be a cheap attack**, since obtaining any competing recognized claim would strip a legitimate representative of canonical control. So conflicts resolve by precedence, not by tie:

1. **Self-verifiable beats delegated.** A `domain` claim the indexer checked itself outranks a `cnpj` claim it took on a verifier's word.
2. **Otherwise the higher-ranked verifier wins.** An indexer's verifier list is ordered, not a set — the same way a client's moderator list is a preference, not a flat membership test.
3. **Same tier, same verifier: the earlier claim holds.** A newer claim never silently displaces an incumbent; the incumbent has to be revoked.
4. **Only when precedence genuinely cannot separate them** does the entity fall back to tier 2, with edits from both sides competing as proposals until a claim is revoked or one side gains a stronger form of evidence.

**v1, as built** (`projection.ResolveFields`): one recognized holder is the representative; among several, exactly one self-verifiable holder wins (rule 1); anything else falls back to tier 2. Rule 3 is not applied yet, so two identities a domain verifies at once leave the entity with no representative until one record or claim goes — the safe failure, since it grants nothing.

## Official responses

The same claim that grants tier-1 edit authority is what makes a `review.response_created` *official*. A response is publishable by anyone — it's a new statement by its own author, not a change to the review — so "official" is not a permission, it's a display fact: this response came from an identity holding a recognized claim for the entity being reviewed.

Two things follow. A client that shows an "official response" badge is asserting something about a claim it recognized, so it must apply the same recognition rules as tier 1 — a self-signed `cnpj` claim buys a company nothing here either. And an unbadged response from the company's marketing intern is not blocked, hidden, or invalid; it's simply an ordinary reply from an ordinary identity, which is the correct outcome.

**v1, as built:** the indexer marks a response `official` when its author holds a recognized `domain` claim on the entity the review is about, and the client badges it "Official response". The same claim keeps that identity's own reviews of the entity out of the rating summary — shown, marked, not counted.

## Verified-representative edits are authoritative, not unaccountable

Tier 1 applies a verified representative's edit as canonical without waiting for confirmation. That's a deliberate concentration of power in one key, and a compromised company key vandalizing an entity page is a realistic outcome. The resolution doesn't need new machinery, only the discipline to not throw away what already exists:

- **Every proposal is retained**, tier-1 edits included. Canonical is a pointer into a history, never a destructive overwrite — which is the same rule as everywhere else in this project, applied to entity metadata.
- **Tier-1 edits are not exempt from moderation.** A `moderation.label_created` applies to an `entity.updated` exactly as it would to a review, so an indexer following that moderator can decline to project a vandalized edit while the underlying event stays on the network.
- **An indexer can fall back to the previous canonical value**, since it still has it. That is a policy action by the indexer operator, deliberate and visible, not an automatic rollback rule — automating it would just hand the same power to whoever could trigger the automation.

The remaining exposure is honest and unavoidable: between the bad edit and someone noticing, the vandalized value is what readers see. Verification buys speed at the cost of that window; tier 2 buys review at the cost of everything being slow. Different indexers can sit at different points on that trade, which is the usual answer here.

## Open questions

- Whether entity-edit proposals decay the same way votes might — see [docs/reputation.md](./reputation.md#open-questions).
- Whether the flat 17-value category vocabulary ([docs/protocol.md](./protocol.md#entities)) survives contact with real entities, or wants a hierarchy. Flat is the v1 answer because a new value is additive and a hierarchy is not.
- Whether the 20% incumbency margin and the descriptor fields for `entity:<hash>` survive contact with real data — both are v1 defaults chosen to be adjustable, not to be right on the first try.
