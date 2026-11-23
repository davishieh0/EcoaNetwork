# Ecoa indexer

The read side of Ecoa. It follows the event logs of one or more nodes, turns
them into tables that can be searched, and answers an HTTP API the gateway
calls. The design is in [docs/design.md](./docs/design.md), and the order it
is built in is in [docs/roadmap.md](./docs/roadmap.md).

It is configured only by environment variables.

## Run

Postgres comes from `infra/local`:

    cd ../infra/local && docker compose up -d

Its init scripts create the `indexer` role and the `ecoa_indexer` databases,
but only on a data folder that is empty. On a Postgres that already ran,
create them once by hand:

    cd ../infra/local
    docker compose exec -e INDEXER_DB_PASSWORD=goodpassword \
        -e POSTGRES_USER=postgres postgres \
        sh /docker-entrypoint-initdb.d/02-indexer.sh

Then, from this directory:

    cp .env.example .env
    make migrate
    make serve

`make` reads `.env`. The API answers on `HTTP_ADDR`:

    curl http://127.0.0.1:8081/health

## Docker

Build from the repository root, since the module depends on `protocol/go`:

    docker build -f indexer/Dockerfile -t ecoa-indexer .
    docker run -d -p 127.0.0.1:8081:8081 ecoa-indexer

The image runs `serve` by default and answers on port 8081. Another
subcommand goes as the argument, and the migrations are inside the image:

    docker run --rm \
        -e DATABASE_URL='postgres://indexer:goodpassword@host.docker.internal:5432/ecoa_indexer?sslmode=disable' \
        ecoa-indexer migrate

## Subcommands

One binary, one subcommand per job:

| Subcommand | Does | State |
|------------|------|-------|
| `migrate`  | Applies the database migrations | built |
| `project`  | Follows the nodes and writes the projections | every event type of v1 the indexer understands |
| `serve`    | Answers the HTTP API | events and reviews |
| `rebuild`  | Rebuilds the projections from cursor 0, then switches | not built yet |

## Configuration

| Variable       | Default                          | Meaning                           |
|----------------|----------------------------------|-----------------------------------|
| `DATABASE_URL` | the local `ecoa_indexer`         | Postgres the projections live in  |
| `NODE_ADDRS`   | `127.0.0.1:50051`                | gRPC addresses of the nodes `project` follows, comma-separated |
| `HTTP_ADDR`    | `127.0.0.1:8081`                 | host:port the HTTP API answers on |
| `MODERATORS_FILE` | none, so nobody is followed   | JSON file of the moderators whose labels count, read when `serve` starts |


## API

| Route | Answers |
|-------|---------|
| `GET /health` | 200 when the database answers, 503 when it does not |
| `GET /events/{eventId}` | the signed envelope, as a string, with `derived` and `observed` |
| `GET /entities?q=` | entities with a name that holds `q`, ignoring case and accents, newest first, paged; every entity without `q` |
| `GET /entities/{entity}` | the value of each field (name, category, description) and the proposal that set it; 404 when nothing named it |
| `GET /profiles/{did}` | the identity's current profile, as a signed item; 404 when it published none |
| `GET /profiles?ids=` | the current profiles of up to 100 identities, in the order asked, for the authors of a page in one call |
| `GET /identities/{did}` | the identity a key belongs to: its keys in order, the current one, whether it is contested, its revocations |
| `GET /entities/{entity}/reviews` | the current version of each review, newest first; `?limit=` (1 to 100, 20 by default) and `?after=` (the `next` of the previous page) |
| `GET /reviews/{eventId}` | one version with every version of its chain; 410 when the author withdrew it |
| `GET /reviews/{eventId}/responses` | the conversation on a review, in order, paged like the listing; each says what it answers and whether the text changed since |

A failure is `{"error": "<slug>", "message": "..."}`, as the backend answers.

## Moderation

Every label is stored, whoever signed it; the ones that count are those of
the moderators in `MODERATORS_FILE`:

    {"moderators": ["did:key:z6Mk…"]}

`illegal` and `personal-data` leave every list and answer 451 by id;
`spam`, `harassment` and `impersonation` leave lists unless `?labeled=show`;
`off-topic` and `other` are shown. The labels that count come in each item's
`observed.labels`. A file that is not valid stops `serve`. Change it and
restart `serve`; nothing needs rebuilding.

## Following the nodes

With a node running (see `node/README.md`):

    make migrate
    ./bin/indexer project

`project` reads each node's log from where it stopped and keeps up with it;
it reconnects on its own when a node goes away. Run `serve` beside it to
answer the API.

The client code for the node's gRPC is generated from
`protobuf/node_api.proto`:

    make proto

It needs `protoc`, and `protoc-gen-go` v1.36.12 and `protoc-gen-go-grpc`
v1.6.2 from `go install`, as the node does.

## Queries

Each subject writes its SQL in `internal/<subject>/queries.sql`, and `sqlc`
generates typed Go from it into `internal/<subject>/sqlc/`, which is never
edited by hand. After changing a query or a migration:

    go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
    make sqlc

## Tests

    make test

The database tests use `ecoa_indexer_test` from `infra/local`, or
`TEST_DATABASE_URL` when set. Each test works in a schema of its own, dropped
when it ends, so packages tested at the same time never meet. Without the
database they skip, and say why.
