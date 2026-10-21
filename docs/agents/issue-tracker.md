# Issue tracker: Linear

Issues and specs for this repo live in Linear, reached through Linear's hosted MCP
server. There is no CLI in the loop — the MCP tools are the interface.

## Setup

The server is registered in the global opencode config:

```
opencode mcp add linear --url https://mcp.linear.app/mcp --global
```

Authentication is a browser OAuth flow and has to be run by the developer:

```
opencode mcp auth linear
opencode mcp list      # confirm it reads "connected", not "needs authentication"
```

Tokens land in `~/.local/share/opencode/mcp-auth.json`.

**Before relying on any tool name below, run `opencode mcp list` and check what the
server actually exposes.** opencode registers MCP tools under the server name with a
single underscore (`linear_*`), which is not the `mcp__linear__*` convention Claude
Code uses. Confirm the exact identifiers rather than guessing — a wrong tool name
fails as "unknown tool", not as a missing server, and the two are easy to confuse.

## Conventions

Operations are expressed by intent. Each maps to a tool on the Linear server:

- **Create an issue** — title plus body. Use a heredoc for the multi-line body.
- **Read an issue** — by identifier, returning title, body, labels, state, relations.
- **List issues** — filter by state, label, team, project, or assignee.
- **Comment on an issue** — appends to the issue's thread.
- **Apply / remove a label** — by name.
- **Assign / claim** — set the assignee to the acting developer.
- **Close** — set state to completed or canceled, with a closing comment.
- **Set relations** — Linear's native blocking relation is the canonical
  representation of a dependency edge, mirroring the GitHub setup's use of native
  issue dependencies. Where it is unavailable, put a `Blocked by: <identifier>`
  line at the top of the child body.

Publish in dependency order, blockers first, so that each ticket's blocking edges can
reference real identifiers rather than placeholders.

**Never close or modify the parent issue** when working from a breakdown. Sub-tickets
are independent work items; the parent is the record of why they exist.

## Interim local state

Until the Linear MCP is authenticated, ticket breakdowns are written as markdown
files under `.scratch/<feature-slug>/issues/`, one file per ticket, numbered from
`01` in dependency order (blockers first), each carrying its own `Blocked by` list.

This is a temporary fallback, not the destination. When the MCP is connected,
publish those same cards as Linear issues and retire the local files. The card
content does not change between the two — only where it lives and whether blocking
edges are native relations or prose.

## When a skill says "publish to the issue tracker"

Create a Linear issue.

## When a skill says "fetch the relevant ticket"

Read it as above. If the MCP is not authenticated yet, read the matching file under
`.scratch/<feature-slug>/issues/` instead.

## Note on the GitHub repositories

This repo has two remotes and GitHub issues are **not** the tracker here:

- `main` → `eduardokairalla1/EcoaNetwork` (upstream)
- `origin` → `davishieh0/EcoaNetwork` (fork)

The fork has GitHub issues disabled. Upstream issues are enabled but unused, and are
not where work for this repo is tracked. `gh` remains useful for PRs; it is not the
place this project's specs live.