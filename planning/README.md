# Planning — how OpenXOps is planned, built and reviewed

Read [`AGENTS.md`](../AGENTS.md) first; this file adds the detail behind its §3 and §4.

## Files

| File | What it is | Who writes it |
|---|---|---|
| [`ARCHITECTURE.md`](ARCHITECTURE.md) | Concepts: stages, binaries, record kinds, data flow, layout, cross-cutting rules | Claude |
| [`RELEASES.md`](RELEASES.md) | What ships in which release, and what is still undecided | Claude |
| [`DECISIONS.md`](DECISIONS.md) | Numbered decisions (`D-NNN`) and open questions (`Q-NNN`) | Claude, on the repo owner's answer |
| [`STATUS.md`](STATUS.md) | One row per Work Order: status, assignee, issue, PR, branch | Claude |
| `DD-NN-<slug>/DELIVERABLE.md` | One release: goal, scope, Work Items, order, release criteria | Claude |
| `DD-NN-<slug>/WI-NNNN-<Name>.md` | One Work Item: Outcome, Contracts, Work Orders | Claude |
| `<package>/testdata/cases/` | Reference cases: inputs and expected output, read by tests unchanged | Claude |

## IDs

`DD-NN` a release; `WI-NNNN` a Work Item (first two digits = its release); `WO-NNNN-X` a Work
Order. IDs are never reused or renumbered.

## Work Order template

```markdown
## WO-NNNN-X — <title>

**Assignee:** open (default) | Antigravity | Cursor — name one only when it's really needed
**Depends on:** WO-… | —
**Packages:** the only directories this Work Order may change

**Goal.** One sentence.

**In scope.** Bullets.

**Out of scope.** Bullets, each naming the Work Order that owns it.

**Done when.** Checkable bullets. Review checks these and only these.

**Verify.** `go run -tags=no_duckdb_arrow ./scripts/check` (plus any extra command)
```

## Sizing

One PR a reviewer can read in one sitting: roughly 300–1,200 changed lines, one package area.
Two Work Orders that touch the same package never run at the same time.

## Status values

`not started` (no issue) · `to do` (issue open) · `in progress` (claimed) · `in review` ·
`changes` · `merged` · `withdrawn` (dropped by a decision).

## Grill before spec

Before Claude writes a Deliverable or Work Item, or changes a record format or a rule, it asks
the repo owner 3–5 questions per turn and records the answers as decisions. Questions that block
a Work Order are listed as `Q-NNN`; that Work Order is not dispatched until they are answered.

## Reference cases

Inputs and expected output for a rule, in the `testdata/cases/` of the package that reads them.
Tests read them in place and never edit them. A case that looks wrong is reported; Claude fixes
it and records why.

## Release close-out

When a release's last Work Order merges, Claude adds `## Delivered (as built)` to each Work Item
and the repo owner tags the release.
