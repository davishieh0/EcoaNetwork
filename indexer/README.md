# Ecoa indexer

The read side of Ecoa. It follows the event logs of one or more nodes, turns
them into tables that can be searched, and answers an HTTP API the gateway
calls. The design is in [docs/design.md](./docs/design.md), and the order it
is built in is in [docs/roadmap.md](./docs/roadmap.md).

It is configured only by environment variables.

## Run

From this directory:

    cp .env.example .env
    make serve

`make` reads `.env`. The API answers on `HTTP_ADDR`:

    curl http://127.0.0.1:8081/health

## Docker

Build from the repository root, since the module depends on `protocol/go`:

    docker build -f indexer/Dockerfile -t ecoa-indexer .
    docker run -d -p 127.0.0.1:8081:8081 ecoa-indexer

The image runs `serve` by default and answers on port 8081. Another
subcommand goes as the argument: `docker run ecoa-indexer project`.

## Subcommands

One binary, one subcommand per job:

| Subcommand | Does | State |
|------------|------|-------|
| `migrate`  | Applies the database migrations | not built yet |
| `project`  | Follows the nodes and writes the projections | not built yet |
| `serve`    | Answers the HTTP API | `/health` only |
| `rebuild`  | Rebuilds the projections from cursor 0, then switches | not built yet |

## Configuration

| Variable    | Default          | Meaning                           |
|-------------|------------------|-----------------------------------|
| `HTTP_ADDR` | `127.0.0.1:8081` | host:port the HTTP API answers on |

The database and the nodes to follow come with the steps that use them.
