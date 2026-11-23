# ecoa-backend

Ecoa platform backend API.

## Getting started

```bash
cp .env.example .env
uv sync
scripts/dev
```

| Script | What it does |
|---|---|
| `scripts/dev` | Dev server with hot reload, on port 8000 |
| `scripts/test` | Ruff, mypy over `src` and `tests`, then pytest |
| `scripts/build` | Builds the Docker image as `ecoa-backend:<version>-<commit>` |
| `scripts/init` | Production entry point, run by the image |

Routes live under `/api`. The interactive docs are at
`http://localhost:8000/api/docs`, and are switched off when
`ENVIRONMENT=production`.

`scripts/build` refuses to run on a dirty tree, so every tag maps to a real
commit.

## Reads

The read routes (`/api/entities`, `/api/reviews`, `/api/events/{id}`,
`/api/profiles`, `/api/identities`) need no account and pass the indexer's
answer on as it is: its status, and its body as bytes, never parsed, so every
signed envelope in it reaches the client as it was signed. A failure the
indexer explains (400, 404, 410, 451) passes on with its own slug; an indexer
that does not answer is `503 indexer_unavailable_error`. Each route passes on
only the parameters it takes; the indexer validates them. `INDEXER_URL` points
at it.

## Stack

Python 3.12+ · FastAPI · Pydantic 2 · pydantic-settings · Uvicorn · uv ·
Ruff · mypy · pytest
