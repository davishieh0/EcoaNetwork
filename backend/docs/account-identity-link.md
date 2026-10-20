# Linking an account to an identity

**Status: not built, and not modeled.** This is a note for when the work
starts: what the link is, what is already decided, what is still open, and
what it would take to build.

## What it is

The platform has two things that are deliberately separate:

- an **account**: email and password in Cognito, plus a row in `users` with a
  handle. It identifies a person to this platform.
- an **identity**: a `did:key` generated on the user's device. It signs
  reviews, and it is a pseudonym.

Today the backend only knows accounts. It has never seen a `did:key`.

The link is the fact "this account controls this identity". It is the only
piece of data in the whole system that ties a pseudonym to a person, which is
why it gets its own decision instead of a column added in passing.

## Why it would exist

The planned use is a signed claim: when someone registers, the platform
publishes an event saying this identity holds an account here. The indexer can
then give more weight to identities carrying that claim.

The protocol already has the mechanism: a delegated
`verification.claim_created`, signed by a verifier, which each indexer chooses
to recognize or not ([docs/verification.md](../../docs/verification.md)). The
platform would be the verifier, and its own indexer would recognize it.

To sign that claim the backend must know, at that moment, which `did:key`
belongs to the account making the request. Whether it remembers afterwards is
the main question below.

## What is already decided

These come from the project docs and constrain the design:

- **The default is verify, sign, discard.** Keeping the link is "a deliberate
  decision that needs its own justification"
  ([docs/decisions.md](../../docs/decisions.md), on what a verifier retains).
  Whoever holds it becomes the natural target of any legal demand for an
  author's identity ([docs/mvp.md](../../docs/mvp.md#legal-exposure)).
- **The public event never carries the email, or a hash of it.** It carries
  the bare assertion and nothing else.
- **The claim binds an identity through a signed challenge.** The client signs
  a nonce with its key, proving it controls the identity it names.
- **One account may use many identities.** A client must offer publishing
  under a fresh identity as a one-tap option
  ([docs/identity.md](../../docs/identity.md#pseudonymous-is-not-anonymous)).
- **Claims expire.** Delegated claims live two years at most
  ([docs/protocol.md](../../docs/protocol.md#verification)).
- **`users.id` is the key other tables point at**, not the Cognito `sub`, so a
  link table needs no change to `users`.

## What is still open

Each answer changes the schema, which is why nothing is modeled yet.

1. **Keep the link or discard it?** Discarding follows the default and leaves
   nothing to leak or hand over. Keeping it needs a reason. Candidates, and
   whether each one really needs storage:
   - *Renewing a claim before it expires*: no. The logged-in client can ask
     again and sign a new challenge.
   - *Revoking the claim when the account is deleted*: yes, unless the client
     is asked to present its identities at deletion time.
   - *Stopping one account from collecting claims for unlimited identities*:
     a counter per account is enough; the identities themselves are not.
2. **Which claim type?** The schema allows `domain`, `cnpj`, `email`, `human`
   and `purchase`. A confirmed Cognito email is what `email` already means, so
   the claim may simply be `email`. A new type ("holds an account here") would
   be a protocol change.
3. **How many identities per account get a claim?** One, a fixed number, or
   any. If every identity of an account is claimed and the links are stored,
   the server rebuilds the linkage that fresh identities exist to avoid.
4. **What happens on key rotation?** `identity.key_rotated` moves an identity
   to a new key. Either the claim follows the succession on its own, or it has
   to be issued again.
5. **Does the user choose?** Linking should be an explicit step that says what
   is being traded, not a side effect of registering.

## What it would be in the database

Three shapes, from least to most data held.

**No table.** The backend checks the challenge, signs the claim, and forgets
the identity. Nothing new in the database. This is the default the docs ask
for.

**A counter.** Enough to cap how many identities an account can claim, without
recording which ones:

```sql
ALTER TABLE users ADD COLUMN claims_issued integer NOT NULL DEFAULT 0;
```

**A link table.** Only if question 1 ends in "keep":

```sql
CREATE TABLE account_identities (
    user_id     uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    did         text        NOT NULL,
    claim_id    text        NOT NULL,   -- the eventId of the claim issued
    expires_at  timestamptz NOT NULL,   -- when that claim stops counting
    created_at  timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT pk_account_identities PRIMARY KEY (user_id, did),
    CONSTRAINT uq_account_identities_did UNIQUE (did)
);
```

`UNIQUE (did)` means an identity belongs to one account. `ON DELETE CASCADE`
means deleting the account deletes its links. If this table exists, it needs a
retention rule written down next to it: how long a row lives, and who can read
it.

## What has to be built

None of this exists today.

**Backend**

- A verifier key: an Ed25519 key the platform signs claims with. Where it is
  stored and how it is rotated. A leaked verifier key lets an attacker mint
  claims at will.
- `did:key` handling in Python: decoding the identity into a public key and
  verifying an Ed25519 signature. No component in this repo does it yet. The
  encoding is settled in [docs/decisions.md](../../docs/decisions.md):
  Ed25519, multibase `z`, multicodec `0xed01`, with ZIP-215 signature rules.
- A challenge endpoint: issues a short-lived nonce to a logged-in account.
- A claim endpoint: receives the identity and the signed nonce, verifies the
  signature, builds the `verification.claim_created` event in canonical form,
  signs it and publishes it.
- Publishing: a way for the backend to hand an event to a node or gateway.
- Revocation: issuing `verification.claim_revoked`, and deciding what triggers
  it.
- Depending on the shape chosen: the migration, the storage class and their
  tests.

**Frontend**

- Generating and storing the identity on the device.
- Signing the challenge.
- The screen that explains the trade and asks for consent.

**Indexer**

- Recognizing the platform's verifier key for the chosen claim type, and the
  weight that recognition gives.

**Docs**

- An entry in [docs/decisions.md](../../docs/decisions.md) with the answers to
  the open questions, written before the code.
- The claim type in the protocol, if a new one is added.

## Before starting

1. Answer the five open questions.
2. Write the decision down.
3. Only then pick one of the three database shapes.
