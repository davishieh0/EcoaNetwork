# Triage Labels

The skills speak in terms of five canonical triage roles. This file maps those roles
to the actual label strings used in this repo's Linear workspace.

This repo's existing `Type` and `Area` label groups are **classification** — what
kind of work an issue is, and which part of the system it touches. Triage is
**state**: who should pick it up. The two are orthogonal, and an issue carries one of
each.

| Role in the skills           | Label in our tracker | Exists | Meaning                                  |
| ---------------------------- | -------------------- | ------ | ---------------------------------------- |
| `needs-triage`               | —                    | no     | Maintainer needs to evaluate this issue  |
| `needs-info`                 | —                    | no     | Waiting on reporter for more information |
| `ready-for-agent`            | `ready-for-agent`    | **yes** | Fully specified, ready for an AFK agent |
| `ready-for-human`            | —                    | no     | Requires human implementation            |
| `wontfix`                    | —                    | no     | Will not be actioned                     |

When a skill mentions a role (e.g. "apply the AFK-ready triage label"), use the
corresponding label string from this table.

## Why only one label exists

`to-spec` and `to-tickets` apply `ready-for-agent` and nothing else — every card they
produce is agent-grabbable by construction. The other four roles are implemented by
`/triage`, which has not been run against this workspace yet.

Creating five labels up front would put four labels on the workspace that nothing
applies. Unused labels are what makes a taxonomy rot: they get ignored, then
contradicted, and eventually the next person adds a fifth way of saying the same
thing.

The other four are created on demand, one call each, when a skill first needs them.
The mapping above is what that call follows.

## Why a bare label and not a `Triage` group

The team's existing groups — `Type` and `Area` — are both `singleSelect`, and a third
group would have been stylistically consistent. A bare workspace label was chosen
instead because it is the smallest possible addition:

- it does not alter how any existing filter behaves
- `singleSelect` **silently drops** a second label rather than erroring, so an agent
  that tries to apply two roles would leave the issue in a state it did not intend,
  with nothing to show for it
- it is reversible in one call

If the workspace later grows enough triage labels to be worth grouping, add a
`Triage` group then — and make it `multiSelect` for the reason above.

## Undo

```
linear.retire_issue_label  id: cf633a1a-9b41-4385-b8c3-a05ff046236f
```

Retiring hides the label from new issues without touching existing ones.
`restore_issue_label` undoes it.