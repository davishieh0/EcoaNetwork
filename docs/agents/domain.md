# Domain Docs

How the engineering skills should consume this repo's domain documentation when
exploring the codebase.

## Before exploring, read these

- **`GLOSSARY.md`** at the repo root, if it exists.
- **`docs/decisions.md`** — this repo's decision log. It is long, written to be
  argued with, and every entry states what was rejected and why. Read the sections
  that touch the area you're working in; entries are numbered by section and stable.
- **`docs/adr/`** — if it exists, read ADRs covering the area. This repo does not
  use a separate ADR directory; the decision log above is where decisions live.

If any of these files don't exist, **proceed silently**. Don't flag their absence;
don't suggest creating them upfront. The `/domain-modeling` skill (reached via
`/grill-with-docs` and `/improve-codebase-architecture`) creates them lazily when
terms or decisions actually get resolved.

## This repo is spec-first

The prose in `docs/` is the source of truth, and it outranks the code. When code and
documents disagree, the documents win **unless** the divergence is recorded in
`docs/decisions.md`.

That inverts the usual default. Before implementing anything under `node/`, read
`docs/protocol.md` — most of what the node does is defined there. An assertion about
a protocol rule belongs in a document, never in an agent's memory or in a comment
that quietly disagrees with the spec.

Docs are also an Obsidian vault: `docs/conceitos/` and `estudos/` are study notes
that use wikilinks with exact headings (`[[protocol#Field rules]]`), not markdown
links. Follow that convention when editing those files.

## File structure

Single-context repo:

```
/
├── GLOSSARY.md              ← created lazily by /domain-modeling
└── docs/
    ├── protocol.md          ← the specification
    ├── architecture.md      ← roles, write path, read path
    ├── decisions.md         ← the decision log; see above
    ├── checklist.md         ← the build plan, numbered section.item
    └── adr/                 ← only if a decision warrants a standalone ADR
```

## Use the glossary's vocabulary

When output names a domain concept — an issue title, a refactor proposal, a
hypothesis, a test name — use the term as defined in `GLOSSARY.md`. Don't drift to
synonyms the glossary explicitly avoids.

Vocabulary drift is a known failure mode in this repo, not a hypothetical one. The
decision log records an instance: three documents used "disputed" for both a
moderation label and a community vote, in a project whose entire design rests on
keeping those two apart. A shared vocabulary that collides on the distinction the
project cares most about costs more than the sum of its parts.

If the concept you need isn't in the glossary yet, that's a signal: either you're
inventing language the project doesn't use (reconsider), or there's a real gap (note
it for `/domain-modeling`).

Prefer the term the documents already use. If a document has already named
something, do not introduce a synonym for it, even a more familiar one — that is
the whole of the "disputed" bug.

## Flag contradictions

If output contradicts an existing decision-log entry, surface it explicitly rather
than silently overriding:

> _Contradicts decisions.md §1.2 (every encoding is pinned), but worth reopening
> because…_

If output contradicts the code, say so too — and don't assume the code is wrong. In
this repo the documents win, but a divergence nobody recorded is a bug in the
documents, not just in the code.