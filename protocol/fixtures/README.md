# Fixtures

Behavioural cases for the rules that are easy to implement subtly differently. Each file covers
one rule, points at where that rule is specified, and each case states **what breaks if an
implementation gets it wrong**.

These are not cryptographic vectors. Signing, digests and canonicalization are covered separately
by the canonicalization and ZIP-215 signature test vectors. Here the identifiers are symbolic and short — `e1`,
`alice`, `mod` — because a failing case has to be readable, and a 52-character hash is not.

```
fixture.schema.json          the format, validated along with everything else
chain-head.json              resolving the head of an edit chain
vote-and-label-binding.json  votes bind to a version, labels and responses bind to the chain
authorship.json              who may replace, withdraw and revoke what
identity-succession.json     succession by first-observed, revocation by local observation
orphan-buffer.json           a reference that has not resolved yet
entity-metadata.json         per-field resolution, with weight supplied as input
```

## Arrival order is data, not a detail

`arrival` is the order in which the node **observed** the events, and for two rules it is the
primary input:

- **Identity succession** resolves by first-observed. That is exactly what the earlier rule —
  lowest `eventId` globally — got wrong, allowing a retroactive hijack for roughly two attempts
  of grinding.
- **Revocation** takes effect at local observation, never at the claimed `createdAt`, because in
  the scenario revocation exists for, the author's clock is in the attacker's hands.

For **chain head** the opposite holds, and one case exists purely to prove it. If the head changes
with arrival order, the projection has stopped being deterministic.

## Weight is an input, never computed

In `entity-metadata.json` each case declares `identities` with a weight per author. That is
deliberate: weight is a local observation, so the **rule** is deterministic and the **outcome** is
per-indexer. Two honest indexers can reach different canonical values from the same log, and that
is the design working — not a case to be "fixed".

## What these cases protect

Each family exists because getting it wrong has a concrete consequence:

| File | If you get it wrong |
|---|---|
| `chain-head` | Two nodes show different versions of the same review |
| `vote-and-label-binding` | Bait-and-switch becomes free, or clearing a spam label costs one character of editing |
| `authorship` | Any identity can edit or delete any other identity's review |
| `identity-succession` | A leaked old key seizes the identity and its whole history |
| `orphan-buffer` | The network loses valid events at a rate proportional to how fast it is |
| `entity-metadata` | Last-writer-wins, and every entity belongs to whoever edited most recently |

## Verification

Fixtures are validated against `fixture.schema.json`, and beyond that: ids are unique per case,
every reference named in `expect` exists in `arrival`, and every author has a declared weight
whenever the case declares weights.
