# Reputation & Ranking

This covers the community voting layer: how reviews get a quality signal from other users, and how that signal changes what people see by default — without ever touching the original event.

## The vote event

Voting reuses two event types already defined in [docs/protocol.md](./protocol.md):

- `review.endorsed` — "this matches my experience too" (upvote)
- `review.disputed` — "this is inaccurate or misleading" (downvote)

```json
{
  "target": "evt_k3f9x2mq7v",
  "reason": "optional short text, at most 280 characters"
}
```

The event type is in the envelope, not the payload — nothing in a payload ever repeats an envelope field ([docs/protocol.md](./protocol.md#conventions)).

Signed and published like any other event. `reason` is optional and, when present, is treated as user content — no HTML, subject to the same content rules as a review (see [docs/identity.md](./identity.md#privacy)).

`target` is any `eventId`, but only two kinds of target mean anything in v1: a review, and a `review.response_created`. A vote on anything else is a structurally valid event that indexers ignore by default — worth allowing rather than rejecting, since a future event type may want votes and the wire format shouldn't have to change for it, and worth ignoring rather than counting, since a score attached to something with no display surface is just a place for spam to accumulate.

### Idempotency rule

An indexer counts only the **latest** vote from a given identity on a given target — not the sum of every vote event that identity ever published for it. Casting the same vote 500 times, or flipping back and forth, doesn't move the score beyond that one identity's current weight. This is a projection rule, not a protocol mechanism — nothing new needs to be added to the wire format for it.

"Latest" here means the ordering rule from [docs/protocol.md](./protocol.md#what-createdat-can-and-cannot-be-used-for): higher `createdAt`, ties broken by lower `eventId`. That's the voter's own self-attested clock, which is safe precisely because it's self-scoped — backdating your own vote only reorders it against your own earlier votes on the same target, and the outcome is one vote either way. Every indexer resolves the same current vote from the same events.

### A vote binds to the version it was cast on

`review.endorsed` and `review.disputed` target a specific `eventId`, and that binding survives an edit: endorsements collected on one version of a review do not transfer to a version the endorsers never read. That rule, and the bait-and-switch it exists to prevent, are described in [docs/protocol.md](./protocol.md#votes-do-not-follow-an-edit). The consequence here is that a review's displayed score is always the score of the text currently on screen.

## Voting vs. moderation

These are deliberately two different systems, even though both attach signals to an event:

| | Moderation label | Community vote |
|---|---|---|
| Question it answers | Is this spam / illegal / off-policy? | Do I find this accurate / useful? |
| Who can publish it | Anyone, but only labels from moderators a client/indexer chooses to follow are applied | Anyone, weighted by identity strength |
| Effect | Can hide content behind a label by default | Shifts ranking/sort order only |
| Governance | Pluralistic — different moderator lists per client | Pluralistic — different scoring formulas per indexer |

Mixing them would let mass downvoting act as informal censorship. Keeping them separate means a heavily-disputed review can still rank low without ever being labeled or hidden as policy-violating.

## Eligibility and weighting

Consistent with the rest of the project: nobody is blocked from voting, they're just weighted less until they've proven something. This reuses the claims model from [docs/identity.md](./identity.md#claims-not-tiers) directly — no separate "registered user" concept needed.

```
weight(identity) = claim_weight(identity) × age_multiplier(identity)
```

`claim_weight` is the weight of the **strongest** claim the identity holds that this indexer recognizes — the strongest, not the sum, since adding up cheap claims would recreate exactly the problem the ladder exists to avoid. Suggested v1 defaults:

| Recognized claim | `claim_weight` | Why that number |
|---|---|---|
| none | 0.1 | counts, but barely moves the score |
| `human` | 0.3 | a CAPTCHA is worth roughly a tenth of a cent at scale — a step above a fresh key, and nothing more |
| `email` | 0.6 | a distinct inbox is a real if modest cost per identity |
| `domain` / `cnpj` | 1.0 | control of a domain, or a verified company, is expensive to forge at scale |

`age_multiplier` grows slowly (e.g. logarithmic in days since the indexer's *earliest observation* of that identity, not since its self-declared first post — see [docs/protocol.md](./protocol.md#what-createdat-can-and-cannot-be-used-for)) and caps at a ceiling, so established identities get more say without any single one dominating a review's score.

An indexer that has only just started has no meaningful observations and must not pretend otherwise: it either flattens `age_multiplier` to a constant, letting `claim_weight` carry the whole formula, or adopts a starting view from a longer-running indexer it explicitly chooses to trust. Silently treating every identity as brand new would hand a fresh indexer's rankings to whoever votes first.

**Why not a single "has a claim" flag.** An earlier version of this promoted any identity holding *any* verified claim from 0.1 straight to 1. That prices the entire ladder at the cost of its cheapest rung: `human` is issued by a CAPTCHA at the gateway (see [docs/verification.md](./verification.md)), solved CAPTCHAs go for around a dollar per thousand, and a binary rule therefore sells a thousand full-weight identities for a dollar. Weighting per claim type keeps cheap signals cheap.

It also makes clear which claim is the interesting one for a review platform specifically: `purchase` (deferred past v1) is the only claim type that ties a voter to the actual transaction being reviewed, rather than merely establishing that some cost was paid somewhere.

The formula lives entirely in the indexer — like reputation and moderation, different indexers can tune or replace it without forking the protocol.

**v1, as built: no weighting yet.** The indexer counts the latest vote of each identity on each exact version (`version_votes`), so every vote weighs the same. The claims it recognizes are already at hand — they mark authors and decide entity authority — so the formula above is the next step, not a redesign.

## What the score actually changes

Only presentation, never the record:

- No event is hidden, edited, or deleted because of its vote score.
- The score affects **default sort order** in an indexer's results, and can drive a UI signal (e.g. "community disputes this — see why") without hiding the text.
- Anyone can still read every review, sorted however they want, on a client/indexer that chooses to expose that.

A review the community rejects fades in the default view — it doesn't disappear from the network. That's the same non-deletion guarantee the rest of the protocol makes, applied to ranking instead of to content.

**v1, as built:** an indexer's ranking is a setting. `RANKING=newest` lists the newest original first; `endorsed` lists the head with the best score first — its endorsements less its disputes, unweighted — then newest. Each answer names the ranking it used, and the client says it ("as this index ranks them"). The demonstration's step 10 switches it on the running index ([decisions.md §16.6](./decisions.md#166-ranking-is-a-setting-and-the-demonstration-changes-it--call)).

## Anti-brigading (later, not v1)

Weighting by claim + age handles the naive case — a burst of keys created today — without any extra machinery. It's worth being clear-eyed that this is the *easy* half: paid vote manipulation isn't an exotic late-stage threat for a review platform, it's the core adversary and the business model of the fraud industry that already exists around every incumbent. What buys the deferral is that the naive case is the common one, and that the damage from the sophisticated case is bounded by ranking rather than deletion — a brigaded review sinks in the default sort, it doesn't vanish.

Coordinated attacks using older or verified identities are a harder, indexer-side problem: detecting bursts of votes on the same target within a short window from identities that otherwise look unrelated, and discounting or holding those votes pending review. This belongs in the same "Later" bucket as the other anti-abuse measures in [docs/identity.md](./identity.md#anti-abuse-sybil-resistance) — not needed to ship a first version.

## Open questions

- Exact shape of the age-multiplier curve and its cap.
- Nothing about how age is *established* — that's settled as `max(createdAt, earliest local receipt)` in [docs/protocol.md](./protocol.md#what-createdat-can-and-cannot-be-used-for). What stays open is how much it should be worth, given that age can be cultivated in advance (create keys, publish something cheap, wait). That's the reason `claim_weight` and not `age_multiplier` carries most of the defense.
- Whether old votes should decay over time (a dispute from years ago on a since-improved entity shouldn't necessarily weigh the same as a recent one).
- Whether reputation/ranking should be scoped per entity category (a harsh voter in one product category isn't necessarily a signal in another) — raised in early brainstorming, not decided.
- Whether a `reason` on a dispute should itself be votable/moderatable, since it's user content too.
