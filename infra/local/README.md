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
| `frontend` | 8080 | The built site; nginx passes `/api` on to the backend named in `BACKEND_ORIGIN`, so the site and the API share one origin, as Vite's proxy does in development |
| `backend` | 8000 (`BACKEND_PORT`) | The gateway API |
| `indexer-serve` | 8081 | The indexer's read API, `/entities/...`, `/reviews/...` |
| `indexer-2-serve` | 8082 | The second index's read API, with `--profile second-index` |
| `indexer-project` | — | Follows the three nodes and writes the projections |
| `node-1`, `node-2`, `node-3` | gRPC 50051, 50052, 50053; libp2p 4001, 4002, 4003 | The network, each node dialing the other two |
| `postgres` | 5432 | The backend's and the indexer's databases |
| `dns` | 5353/udp | The `.test` zone (`configs/dns/test.zone`) the indexer checks domain claims against |
| `backend-migrate`, `indexer-migrate` | — | Apply the migrations, then exit; the services that need them wait for them |

When 8000 is taken, by a backend run outside for instance:

```bash
BACKEND_PORT=8001 docker compose up -d
```

The frontend reaches the backend inside the Docker network, whatever port
it has here.

## Data to look at

Seed the network with a few hundred signed reviews across sixty entities
(synthetic, in the Receita Federal's layout; see `indexer/cmd/seed`):

```bash
(cd ../../indexer && go run ./cmd/seed -node localhost:50051)
```

Running it again publishes nothing new.

## Verifying a domain

Domain claims are checked against `configs/dns/test.zone`. Add the line the
site's Verification screen shows, relative to `test.`, and raise the serial:

```
_ecoa-verify.minhaloja IN TXT "ecoa-identity=did:key:z6Mk…"
```

CoreDNS rereads the file every few seconds; on macOS, Docker can take a
minute to show it a change to a mounted file (its log says `Successfully
reloaded zone`). The indexer tries a failed lookup again every 30 seconds, so
publishing the record before signing the claim is quickest. The backend issues human claims with the key in
`HUMAN_VERIFIER_SEED`, which `configs/indexer/verifiers.json` lists. **That
key is for this machine only**, like the nodes' peer keys.

## The demonstration

Steps 5 and 6, a tampered event and an edit of someone else's review:

```bash
(cd ../../indexer && go run ./cmd/demo tamper && go run ./cmd/demo foreign-edit)
```

Step 9, the index dropped and rebuilt from the nodes' logs:

```bash
docker compose stop indexer-serve indexer-project
docker compose run --rm indexer-project rebuild
docker compose start indexer-project indexer-serve
```

Step 10, a second index over the same logs with another ranking: its own
database and projector, ranking by endorsements, its API on `:8082` beside
the first one's on `:8081`. The same entity's reviews come back in another
order, and each answer names its `ranking`:

```bash
docker compose --profile second-index up -d
curl -s 'localhost:8081/entities/<entity>/reviews'
curl -s 'localhost:8082/entities/<entity>/reviews'
```

Its database, `ecoa_indexer_2`, is created with the others when the
Postgres volume is first made; on an older volume, create it once:

```bash
docker compose exec postgres psql -U postgres -c 'CREATE DATABASE ecoa_indexer_2 OWNER indexer'
```

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
