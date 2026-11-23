# infra/local

The whole project on a developer's machine with Docker Compose: the
database, a network of three nodes, the indexer, the backend and the
frontend.

## Requirements

- Docker with Compose v2.24 or later.
- `backend/.env` filled in, Cognito included (see `backend/.env.example`).
  The backend reads it as it is; this file only overrides what differs inside
  the Docker network. Without it, everything starts but the backend.

## Running it

Everything, building the images from the repository:

```bash
docker compose up -d --build
docker compose ps
```

Then open <http://localhost:8080>.

Only the database, for when the backend and the frontend run outside
containers:

```bash
docker compose up -d postgres
```

## Services

Every port is published on `127.0.0.1`, reachable from this machine only.

| Service | Port | What it is |
|---|---|---|
| `frontend` | 8080 | The built site; nginx passes `/api` on to the backend, so the site and the API share one origin, as Vite's proxy does in development |
| `backend` | 8000 (`BACKEND_PORT`) | The gateway API |
| `indexer-serve` | 8081 | The indexer's read API, `/entities/...`, `/reviews/...` |
| `indexer-project` | — | Follows the three nodes and writes the projections |
| `node-1`, `node-2`, `node-3` | gRPC 50051, 50052, 50053; libp2p 4001, 4002, 4003 | The network, each node dialing the other two |
| `postgres` | 5432 | The backend's and the indexer's databases |
| `backend-migrate`, `indexer-migrate` | — | Apply the migrations, then exit; the services that need them wait for them |

When 8000 is taken, by a backend run outside for instance:

```bash
BACKEND_PORT=8001 docker compose up -d
```

The frontend reaches the backend inside the Docker network, whatever port
it has here.

## The nodes

Each node's peer key is fixed, in `configs/nodes`, so their peer IDs are
known in advance. That is what lets each one bootstrap from the other two,
and lets any other node join this network:

| Node | Peer ID |
|---|---|
| `node-1` | `12D3KooWScQpy5eMWGnq9bTa1w2b2S5zQRtvGc9e6HHW2BRtzPH4` |
| `node-2` | `12D3KooWBDfuNroD7Nu9mXbrYAm1H4B3eHwtzsw2Z7UBAVrQCNs7` |
| `node-3` | `12D3KooWHpQ9ZRanrBMfS62s4vYnvueZDLu4WtcsaCRJCGSGHJvs` |

A node run outside Compose joins with, for instance:

```bash
BOOTSTRAP_ADDRS=/ip4/127.0.0.1/tcp/4001/p2p/12D3KooWScQpy5eMWGnq9bTa1w2b2S5zQRtvGc9e6HHW2BRtzPH4
```

**These keys are for this machine only.** Anyone with the repository has
them; a node anywhere else must never run with one.

## Layout

| Path | What it holds |
|---|---|
| `docker-compose.yaml` | Every service |
| `configs/<service>` | The service's configuration, mounted read-only |
| `configs/indexer/moderators.json` | The moderators whose labels the indexer acts on, empty to begin with |
| `data/<service>` | Each service's data, outside git: `data/postgres`, and each node's log and peer list in `data/node-1`, `data/node-2`, `data/node-3` |

After editing a file in `configs`, restart the service:

```bash
docker compose restart <service>
```

After changing the code of a service, rebuild it:

```bash
docker compose up -d --build <service>
```

## Data

Data survives `docker compose down`. To start a service from scratch:

```bash
docker compose down
rm -rf data/<service>
docker compose up -d
```

`rm -rf data/node-*` empties the nodes' logs; the indexer then reads new logs
from the start, by their new `log_id`.

On Linux, the node image runs as user 65532, which cannot write to a folder
Docker created as root. Create the folders for it once:

```bash
mkdir -p data/node-1 data/node-2 data/node-3
sudo chown 65532 data/node-1 data/node-2 data/node-3
```

Docker Desktop, on macOS and Windows, needs nothing of this.

Some settings are only read when the data folder is empty, such as the
passwords in `.env`. A change there only takes effect after starting over.
