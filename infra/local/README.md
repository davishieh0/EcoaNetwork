# infra/local

The services a developer runs on their own machine with Docker Compose.

## Requirements

- Docker with Compose v2.

## Running it

```bash
docker compose up -d
docker compose ps
```

## Services

Every port is published on `127.0.0.1`, reachable from this machine only.

| Service | Port | Used by |
|---|---|---|
| `postgres` | 5432 | backend |

## Layout

| Path | What it holds |
|---|---|
| `docker-compose.yaml` | Every service |
| `configs/<service>` | The service's configuration, mounted read-only |
| `data/<service>` | The service's data, outside git |

After editing a file in `configs`, restart the service:

```bash
docker compose restart <service>
```

## Data

Data survives `docker compose down`. To start a service from scratch:

```bash
docker compose down
rm -rf data/<service>
docker compose up -d
```

Some settings are only read when the data folder is empty, such as the
passwords in `.env`. A change there only takes effect after starting over.
