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
| `project`  | Follows the nodes and writes the projections | not built yet |
| `serve`    | Answers the HTTP API | `/health` only |
| `rebuild`  | Rebuilds the projections from cursor 0, then switches | not built yet |

## Configuration

| Variable       | Default                          | Meaning                           |
|----------------|----------------------------------|-----------------------------------|
| `DATABASE_URL` | the local `ecoa_indexer`         | Postgres the projections live in  |
| `HTTP_ADDR`    | `127.0.0.1:8081`                 | host:port the HTTP API answers on |

The nodes to follow come with the step that uses them.

## Tests

    make test

The database tests use `ecoa_indexer_test` from `infra/local`, emptied by each
test, or `TEST_DATABASE_URL` when set. Without it they skip, and say why.
