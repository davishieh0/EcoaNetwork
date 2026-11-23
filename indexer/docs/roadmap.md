# Read path: roadmap

What to build for reads to work end to end, by service, in order. The design
is in [design.md](./design.md); this says what comes first and what waits on
what.

## Node

Already in v0.4.0: a local sequence and receive time for every stored event,
reading the log from a position (`Store.EventsSince`), and one arrival path
for every door. What is left:

1. ~~**`log_id`**, created at random with the store, in a bucket of its own.~~
   Done: `Store.LogID()`.
2. ~~**Telling open streams when an event is stored**, so they send it at
   once.~~ Done: `Store.Changed()`.
3. ~~**The contract in `node_api.proto`:** `SubscribeEvents`, `Cursor` (with
   the `log_id` the cursor counts in) and `StoredEvent` (with `head`);
   regenerate the code.~~ Done, with `make proto` and `scripts/proto`.
4. ~~**Serving the stream:** the history after the cursor through
   `EventsSince`, then each new event, with a configurable cap on open
   streams, read from the environment like the sync limits.~~ Done:
   `SUBSCRIBE_MAX_STREAMS`, and streams end when the node stops.
5. ~~**Tests:** a stream resumes from the right cursor, sees an event stored
   while it is open, starts over for another `log_id`, and is refused over
   the cap.~~ Done, over an in-memory gRPC connection.

## Protocol (`protocol/go` 0.1.0, done)

What the indexer needed is there: `envelope.Decode`, `identity`, the
`projection` rules (chain head, withdrawal, votes, labels, responses, entity
identifier, metadata), `testevent.Sign` and `fixturetest`. Nothing here
blocks the indexer any more. One thing may be added:

- **Contesting a key from rotations seen on different streams.**
  `identity.Tracker` contests only rotations that come in one batch. The
  indexer can apply the rule itself in SQL; moving it to the library is
  optional.

## Indexer

1. ~~**Skeleton:** `go.mod` with a `replace` to `../protocol/go`, as the node
   has; one binary with `migrate`, `project`, `serve` and `rebuild`,
   configuration from the environment, `slog`, a Makefile.~~ Done: `serve`
   answers `/health`, and the other subcommands say they are not built yet.
   Each setting is added with the step that uses it.
2. ~~**Database:** in `internal/platform/database/`: the `indexer` role and
   the `ecoa_indexer` databases in `infra/local`, `DATABASE_URL`, `goose`
   migrations embedded in the binary, `indexer migrate`.~~ Done, with only
   the base tables (`events`, `event_observations`, `projector_cursors`).
   The tables of each subject come with the step that uses them, and `sqlc`
   with step 3.
3. **HTTP API over rows inserted by hand**, in three parts. This unblocks
   the backend and the frontend before the projector exists.
   - ~~**3.1 Foundation:** the `pgx` pool, `sqlc`, the `event` subject and
     `GET /events/{eventId}`, the answer shapes (`platform/respond`), and
     `/health` checking the database.~~ Done.
   - ~~**3.2 Reviews:** the `reviews` and `review_chains` tables,
     `GET /entities/{entity}/reviews` (paged by cursor) and
     `GET /reviews/{eventId}`.~~ Done. Until the projector fills
     `review_chains`, rows are inserted by hand.
   - ~~**3.3 Entities:** the `entities` table, `GET /entities/{entity}` and
     search. Needs full text and metadata resolution; may come after the
     projector.~~ Done, with `entity.updated` in step 5. Filtering the
     search by category comes with step 6.

   *Waits on nothing.*
4. ~~**Stream client:** `NODE_ADDRS`, a list of nodes, one cursor each,
   reconnecting on its own, everything into one applier.~~ Done:
   `indexer project` stores every event, its observation and the node's
   cursor; the tables of each type come in step 5.
5. **Projector, one event type at a time**, calling the `projection`
   functions of `protocol/go` and tested against `protocol/fixtures/`:
   - ~~`review.created`, `review.replaced` and `review.withdrawn`, with
     `ResolveHead` (`chain-head.json`)~~ Done: a review published on a node
     is listed, an edit moves the head, a withdrawal hides it.
   - ~~`review.endorsed` and `review.disputed`, with `CurrentVote` and
     `HeadScore` (`vote-and-label-binding.json`)~~ Done: each version's
     `endorsements` and `disputes` in the API.
   - ~~`review.response_created`, with `ResponsesOnChain`~~ Done, with
     `GET /reviews/{eventId}/responses` (it was step 5's from the start).
   - ~~`moderation.label_created` and `moderation.label_revoked`, with
     `LabelsOnChain`~~ Done, with the moderators file and what each label
     does in the API (it was step 6's).
   - ~~`entity.updated`, checking the identifier with `DeriveEntityID`
     (`entity-metadata.json`)~~ Done: proposals stored, resolved per field
     with `ResolveFields` when the API answers.
   - ~~`profile.updated`~~ Done, with `GET /profiles/{did}`,
     `GET /profiles?ids=` and labels on profiles.
   - ~~key rotations and revocations, and contested identities
     (`identity-succession.json`)~~ Done: votes, profiles, label
     revocation and the moderators file follow the identity, and
     `GET /identities/{did}`.
   - unknown types: skipped, counted, the cursor still advances

   *Can be built and tested now, on events from the generator (step 8);
   only reading them from a real node waits on step 4.*
6. **Observed data:** `received_at` (the earliest any node reports), the
   entity summary (distribution and mean, over the reviews that count), the
   metadata that holds for each entity kept in a table so the search can
   filter by category.
7. **Rebuild:** into new tables from cursor 0, switching once they have
   caught up with the nodes.
8. **Test event generator:** signed synthetic reviews with
   `testevent.Sign`, which are also the MVP's seeded dataset. *Waits on
   nothing; worth building early, since the projector's tests use it.*
9. **The guarantees:** a rebuild reproduces the deterministic tables, two
   instances agree, applying twice changes nothing, a crash mid-way loses
   and doubles nothing.

## Backend (gateway)

1. **Indexer client:** `INDEXER_URL`, and an `indexer_unavailable_error` for
   when it does not answer.
2. **Read routes** under `/api`: entity search, an entity, its reviews, one
   review with its chain, one raw event. The envelope passes through
   untouched. *Waits on the indexer, step 3.2.*
3. **Route tests**, with the indexer stubbed.

## Frontend

1. **`logic/datasource/`:** one function per read route, with the types of
   its answers.
2. **Screens:**
   - entity search
   - an entity: its rating distribution and its reviews
   - a review, with its edit history and responses
   - the inspection view: the raw envelope beside the rendered review

   *Waits on the backend, step 2.*
3. **Verifying the signature of everything received.** *Waits on
   `protocol/ts`.*

## What runs in parallel

- **Now, waiting on nobody:** every node step, and indexer steps 1 to 3,
  5, 6 and 8. `protocol/go` being out unblocked the projector and the
  generator.
- **Once the indexer's API is up** (indexer step 3): the backend, then the
  frontend.
- **The node serves the stream now**, so the stream client (indexer step 4)
  and the guarantees against a real node (step 9) wait on nothing either.
