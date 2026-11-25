# Reading: decisions

How the platform shows what the network holds: search, an entity, a review,
and the inspection view. Decided on 2026-11-23, before building it. Each
entry says what to revisit and when.

## What exists before this

- **Writing is real.** The identity is an Ed25519 key created in the browser,
  not extractable, kept in IndexedDB (`logic/protocol/identity.ts`); the
  envelope is signed in the browser (`logic/protocol/envelope.ts`) and
  published through `POST /api/events` (`app/write`). The private key never
  leaves the browser.
- **Reading is a mock.** `app/reviews/Explore.tsx` filters a fixed list from
  the landing's content; `EventInspector` shows a hand-written event in an
  old shape; `VersionStack` is static; `ReviewCard` is typed for the landing.
  No route shows an entity or a review, and nothing verifies a signature.

## Decided

### 1. Signatures are verified in the browser now, with WebCrypto

The inspection view shows whether a signature holds. The frontend already has
what verifying needs: JCS, the signing input, SHA-256 and WebCrypto. What it
adds is reading the public key out of the `did:key` and calling
`crypto.subtle.verify`, after checking that the `eventId` is the hash of the
signing input.

**The cost, accepted:** the protocol verifies under ZIP-215; WebCrypto
verifies under RFC 8032's stricter rules. They agree on every signature an
honest client makes, and disagree only on signatures built on purpose for
the edge cases, which this check could then call invalid although every node
accepts them.

**A browser that cannot check says so.** WebCrypto has Ed25519 from Chrome
137, Firefox 129 and Safari 17; an older one answers "Not checked here",
never "Signature does not hold", which would accuse a valid event. Signing a
review needs Ed25519 too, so such a browser cannot write either.

**Revisit when `protocol/ts` exists:** verify through it, under ZIP-215. The
check lives in one function, `verifyEnvelope` in `logic/protocol/verify.ts`,
so the switch is there and nowhere else.

### 2. Four screens, in this order

| Screen | Route | Shows |
|---|---|---|
| Search (`Explore`, made real) | `/app/explore?q=` | the entities whose name holds the text, with the name that holds |
| Entity | `/app/entities/:entity` | its name and category; its reviews, with rating, body, author, edited, endorsements and disputes; a toggle for what moderation hides (`?labeled=show`) |
| Review | `/app/reviews/:eventId` | the current version, its edit history (`VersionHistory`) and its conversation, with the same toggle; the withdrawn (410) and removed (451) states |
| Inspection | a panel of the review (`EventInspector`, made real) | the raw envelope beside the rendered review, whether its signature holds, and when the network had it |

All of them are public: reading needs no account (`docs/mvp.md`), so they
sit outside `RequireAuth`, as `Explore` already does.

### 3. What comes later, on purpose

The screens are built with what the indexer answers today, and each gains
the rest when it exists. **Revisit as each lands in the indexer
(`indexer/docs/roadmap.md`, "What remains for the MVP"):**

| Missing | Where it shows | Waits on |
|---|---|---|
| The entity's rating summary (distribution and mean) | the entity page | the indexer's step 6 |
| Which nodes hold the event | the inspection view | the indexer's item 1 |
| Moderators chosen by the client | the lists | `?moderators=`, the indexer's item 2 |
| The verified-representative mark, official responses | reviews and responses | claims, the indexer's item 5 |
| Real data in volume | every screen | the generator, the indexer's item 4 |

### 4. Lists load more with a button

Lists page by the cursor the API returns (`next`): a "Load more" button
fetches the page after. Simpler and more predictable than infinite scroll,
and it keeps the position when the user comes back.

**Revisit** if long lists make the button tiresome.

### 5. The interface stays in English

As the rest of the frontend is today.

**Revisit** when the product picks its languages.

## How it is built

- **One requester per backend router**, in `logic/datasource/`: `entities.ts`
  (`searchEntities`, `getEntity`, `listEntityReviews`), `reviews.ts`
  (`getReview`, `listReviewResponses`), `events.ts` (`getEvent`, beside
  `publishEvent`), `profiles.ts` (`getProfiles`), `identities.ts`
  (`getIdentity`). The answers are typed in their `.t.ts`.
- **The envelope arrives as a string and is kept as one.** The requester
  parses the answer, and the envelope stays a string inside it, intact: the
  raw string is what the signature check and the inspection view use; a
  separate parse, in `logic/protocol/`, reads its fields to render them.
- **An author's display name sits beside their `did:key`, never in its
  place.** Nothing verifies a display name (`docs/entities.md`); the page
  fetches the profiles of its authors in one call (`/profiles?ids=`).
- **`components/ecoa/` keeps taking props and never fetching.** Their types
  follow the real data; what does not exist yet (nodes, claims) is optional,
  so the landing's mocks keep working. `ReviewCard` takes real reviews;
  `EventInspector` takes the event to show, and the landing hands it its
  example; `VersionHistory` and `Conversation` are new. `VersionStack` stays
  the landing's drawing: an animated motif, not a list of versions.
- **The new failures get words** in `app/forms/messages.ts`:
  `review_withdrawn_error`, `removed_by_moderation_error`,
  `indexer_unavailable_error`, `entity_not_found_error`, and the rest.
- **After publishing, the write screen links to the review's page**, handing
  it the envelope just signed. Until the indexer projects it (a 404), the
  page shows that envelope, marked, and reads again every 1.5 s, ten times,
  so nobody waits on the index.
- **The entity page's "Write a review" hands the entity over**, and the
  composer starts with its kind and identifier filled in.
- **Display names are matched by the key that signed the profile and by the
  identity**, so after a key rotation a review signed by an older key may
  show no name until `/profiles?ids=` says which asked key each profile
  answers.

## Acting, verifying, and what the index adds

Built after the read screens, on 2026-11-24:

- **On a review**, its author edits it (a `review.replaced`, checked as a
  new review is, without the entity) or withdraws it (a `review.withdrawn`
  with a reason from the protocol's list); anyone else endorses or disputes
  the current version; everyone signed in responds, to the review or to a
  response. Each is signed in the browser (`app/reads/publish.ts`), and the
  page reads again until the index shows it (`expect` in `useReviewData`),
  keeping what is on screen meanwhile. Acting asks for a complete account,
  as writing does; whether the review is yours is this browser's key, read
  without making one (`peekIdentity`).
- **Verification** (`/app/claims`): the human claim through an ALTCHA proof
  of work the gateway checks and signs, and the domain claim, signed by the
  browser once the screen has shown the TXT record to publish. The badges
  follow: *Verified representative* and *Human* beside an author, *Official
  response* on the representative's answers, the representative on the
  entity page.
- **The summary** of an entity's ratings (`RatingSummary`), and the
  ranking each index names: the entity page says which. The search filters
  by category.
- **The gateway is set at run time**: the site's nginx reads
  `BACKEND_ORIGIN`, so one build serves any gateway.

## Moderation and identity

Built on 2026-11-24, for the MVP's stages 4 and 5:

- **Whose labels a reader follows** (`/app/moderation`): the index's list,
  or their own, kept in this browser (`logic/moderation`); every read sends
  it as `?moderators=` when it is their own, so the same label hides a review
  for a reader who follows its moderator and nothing for one who does not.
  **Moderator mode** shows *Label…* on reviews and responses
  (`moderation.label_created`, the protocol's vocabulary and a note) and
  *Revoke* on one's own labels.
- **The public profile** (in the account): display name and bio, a
  `profile.updated` signed in the browser.
- **The identity is made on purpose** (`/app/identity`, and in place on the
  write and verification screens): with a password, the key is exportable
  for an instant, sealed into a backup file (PBKDF2-SHA-256, 600 000
  rounds; AES-GCM bound to the did:key), and kept non-extractable; without
  one, it is non-extractable from the start. Another browser restores it
  from the file and the password, checking the key against the did:key.
  Identities made before this have no backup.
- **A fresh identity per review**, one tap at publishing, kept in this
  browser so its author can edit or withdraw that review, never backed up;
  and, at the same moment, the note that the gateway and the node see the
  author's IP address.

## Where it lives

```
logic/datasource/   entities.ts, reviews.ts, profiles.ts, events.ts (getEvent),
                    signed.t.ts (Signed, Page), route() in http.ts
logic/protocol/     signed.ts (readEnvelope, reviewTextOf), verify.ts,
                    the decoders in encoding.ts
app/reads/          explore/, entity/, review/, and what they share:
                    usePaged, useRead, useAuthors, useInspection, cards.ts
                    (index answers → component props), words.ts, LoadMore,
                    Placeholder, Status, ShowLabeled, stagger.ts; ReadsLayout
                    holds the Backdrop, so it carries on between the screens
components/ecoa/    NetworkField: the read screens' backdrop, peers joined by
                    edges with events travelling between them, in a band that
                    fades out downward
components/ecoa/    ReviewCard, EventInspector, VersionHistory, Conversation
```

## Testing

The frontend has no test framework (vitest was left out on purpose). Each
screen is checked by `eslint`, `tsc` and the build, and against the real
backend and indexer in `infra/local`. **Revisit** when the read screens grow
logic worth testing on its own: `verifyEnvelope`, the decoders and
`cards.ts` are the first candidates.

Checked on 2026-11-23, driving Chrome 158 through `infra/local`: search,
empty and too-long searches; an entity with 25 reviews over two pages, names
beside keys, an unknown and a malformed id; a review with two versions, an
earlier version's link, a conversation with a reply to a reply, a withdrawn
review, a malformed and an unknown id; the inspector on reviews, versions and
responses, valid, and invalid with the body, the signature or the author
changed, or a signature by another key under a correct eventId; signing in,
writing from an entity page, publishing, and reading it back, with the index
made to lag; labels by a followed moderator on a review and a response,
hidden, then shown marked; every screen at 390 px with no horizontal
scroll.
