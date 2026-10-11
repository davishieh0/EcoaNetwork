<picture>
  <source media="(prefers-color-scheme: dark)" srcset="brand/banner/ecoa-banner-dark.png">
  <img alt="Ecoa. Every experience leaves an echo." src="brand/banner/ecoa-banner-light.png">
</picture>

# Ecoa

Every experience leaves an echo.

## Docs

- [Overview](docs/overview.md) — what this is and why
- [Concepts](docs/concepts.md) — the ideas behind it, in plain terms
- [Architecture](docs/architecture.md) — component roles and how they talk to each other
- [How It Fits Together](docs/diagrams.md) — the same thing as diagrams, for the flows that are hard to read as prose
- [Protocol](docs/protocol.md) — event format, signing, sync
- [Entities](docs/entities.md) — what's being reviewed, and who can edit it
- [Identity, Trust & Security](docs/identity.md) — keys, verification, threat model
- [Verification](docs/verification.md) — how claims get issued, and why forging one doesn't help
- [Reputation & Ranking](docs/reputation.md) — community voting, weighting, anti-brigading
- [MVP](docs/mvp.md) — v1 scope, acceptance criteria, the demonstration
- [Roadmap](docs/roadmap.md) — what to build, in what order, and why that order
- [Related work](docs/related-work.md) — Nostr, AT Protocol, SSB and ActivityPub, and where this project sits
- [Decision log](docs/decisions.md) — every architectural decision taken in review, with its cost and what was rejected

## Artifacts

- [`schemas/`](schemas/) — JSON Schema for the envelope and all 18 event types, transcribed from the specification
- [`fixtures/`](fixtures/) — behavioural cases for the rules that are easy to implement subtly differently
