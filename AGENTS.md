# AGENTS.md — rules for every agent working on OpenXOps

This is the one rule file, in three parts. **Part A** binds everyone. **Part B** is for the
planner and reviewer, **Part C** for implementers. Claude Code (`CLAUDE.md`: A and B),
Antigravity (`GEMINI.md`: A and C, plus B when it stands in) and Cursor (`CURSOR.md`, reached
through `.cursor/rules/implementor.mdc`: A and C) all start here; those files only point here.
Change the rules here, never in a copy.

# Part A — Everyone

## 1. What OpenXOps is

OpenXOps is a **standalone single-tenant** set of small command-line tools for cloud operations,
which chain with `|` ([D-004](planning/DECISIONS.md#d-004)):

- `xops`: commands shared by every domain (`xops sense aws-accounts`, `xops report excel`);
- `finops`: cost (first); `secops` and `cloudops` later.

Every command is `<binary> <stage> <thing>` and belongs to one of five stages, in this order:

**Sense › Enrich › Analyze › Enforce › Report**

**Licence:** Functional Source License 1.1, becoming Apache 2.0 two years after each release
([D-024](planning/DECISIONS.md#d-024)): free for anyone's own use, closed to resale. Never add
code under a licence that conflicts with it, and never copy code from elsewhere without its
licence allowing it. Outside contributions need a signed CLA before they are merged.

OpenXOps is a fresh start ([D-001](planning/DECISIONS.md#d-001)). The earlier project,
`PopGreenTh/OpenLZ`, is archived: read it for intent only, never copy code or plans from it.

## 2. Who does what

| Who | Role | Writes |
|---|---|---|
| **Repo owner** | Decides, answers grill questions, gives the go-ahead to dispatch and to merge | Anything |
| **Claude Code** | Planner and reviewer (Part B) | `planning/`, reference cases, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `CURSOR.md`, `.cursor/rules/`. **Never implementation code** |
| **Antigravity** (`agy-argon`) | Implementer (Part C). **Stands in as planner and reviewer** when the repo owner says so (§8) | Implementing: source, tests, CI and docs inside the claimed Work Order's scope. Standing in: what Claude writes |
| **Cursor** | Implementer (Part C) | Source, tests, CI and docs inside the claimed Work Order's scope |

"The planner" and "the reviewer" below mean Claude Code, or Antigravity while it stands in.
Implementers never edit `planning/`, reference cases, `CURSOR.md`, `.cursor/rules/` or this file, except
Antigravity while standing in. If the spec is wrong, contradicts itself or is unclear, **stop
and say so** in the PR or issue; the planner fixes the spec.

## 3. The work hierarchy

**Deliverable (one release) › Work Item › Work Order › Task** ([D-003](planning/DECISIONS.md#d-003)).

- A **Deliverable** is one release: `planning/DD-NN-<slug>/DELIVERABLE.md`.
- A **Work Item** is one domain stage (`finops/enrich`) or one piece of work at the same level
  (the skeleton, a provider folder): `WI-NNNN-<Name>.md` inside its Deliverable folder. Its
  **Contracts** section holds the interfaces and record formats once, for all its Work Orders.
- A **Work Order** is a `## WO-NNNN-X` section of a Work Item: one PR, one implementer, one review.
- A **Task** is the implementer's own step list. It is never written into `planning/`.

`planning/STATUS.md` is the live state. `planning/README.md` describes the loop in detail.

## 4. The loop for one Work Order

1. **Dispatch.** On the repo owner's go-ahead, the planner opens a GitHub issue `WO-NNNN-X: <title>`
   with the Work Order's spec, its `Assignee:` and its `Verify:` line. `Assignee:` is **`open`**
   by default: no agent is named, and the issue doesn't address one.
2. **Claim.** The implementer comments `Claimed by <Antigravity|Cursor> on branch <branch>`.
   **The first claim wins.** If the issue is already claimed, or a PR for it is open, **stop**.
   If `Assignee:` names someone else, don't start.
3. **Build** only what *In scope* says, in the packages the Work Order names. Touching a file
   outside them is a defect, unless the Work Order says so.
4. **Verify.** Run the `Verify:` command. When it fails because the spec contradicts itself, stop
   and report; **never change a test or a reference case to make it pass.**
5. **PR.** The PR description starts with exactly these two lines:
   ```
   Implementer: <Antigravity | Cursor>
   Resolves #<issue>
   ```
   then the `Verify` output, and any implementation note worth keeping.
6. **Review.** The reviewer reviews against the Work Order's *Done when* only, and posts the verdict
   (approved / changes requested) at the top of a GitHub review.
7. **Merge.** On the repo owner's go-ahead.

## 5. Engineering rules

- **Go.** Several binaries, each built from its folder: `xops/`, `finops/` (D-004). Single
  tenant: no tenant field, flag or file suffix anywhere.
- **Layout** (D-012). `<binary>/<stage>/` holds a stage's code, its commands and its
  `testdata/`; cloud clients and mocks live in `aws/` (later `azure/`, `gcp/`, `oci/`); the shared
  record reader, writer and stores live in `record/`. No `cmd/`, no `internal/`, no top-level
  `testdata/`. Reference cases live in the `testdata/` of the package that reads them.
- **Records** (D-006). Commands read and write flat JSON Lines: one object per line, no nested
  objects or arrays, a `kind` field on every record. Only Report writes other formats.
- **Enforce never changes the cloud** (D-010). It adds flags that say what is wrong, where, how
  much cost it touches and what to do.
- **Build tag `no_duckdb_arrow`** on every build and test once DuckDB is linked (Windows/Delve
  CGO linker issue).
- **Embedded storage only, no database servers.** Cache and history go under `data/`, which is
  generated and gitignored (D-007).
- **Every cloud call** goes through its provider folder's retry wrapper; **every Cost Explorer
  call** goes through the cache, because each call costs $0.01.
- **Offline first.** Every command that calls a cloud has `--mock`, fed by deterministic fixtures.
- **Debuggable** (D-013). Stage logic is plain functions (records in, records out, no I/O);
  commands only wire flags and files to them. Every command runs alone from `--from-<kind>`
  files and `--mock`, has a `.vscode/launch.json` entry, and logs to stderr with
  `--log-level`, never to stdout. Errors name the file and record line. No code generation or
  reflection-driven frameworks; at most 5 non-test `.go` files per package.
- Wrap errors with `%w`. Bound concurrency (5 workers by default), never unbounded goroutines.
- `gofmt` clean. Tests table-driven where it fits. Exported identifiers documented.
- A source change updates the README or the command help in the same PR.

## 6. Commands

```bash
go run -tags=no_duckdb_arrow ./scripts/check   # build, vet, gofmt -l, tests — the Verify command
```

(`scripts/check` arrives with the first Work Order; until then the repository holds only plans.)

## 7. State protocol

For every agent: Claude Code, Antigravity and Cursor ([D-027](planning/DECISIONS.md#d-027)). It
adds to §2 and §4 and never overrides them: the planner still writes no implementation code, the
issue claim still decides who builds a Work Order, and `Verify` still runs before every PR.

**State lives in `.state/tasks.jsonl`, not in chat memory or scratchpads.**
- One JSON object per line, appended, never edited or deleted.
- Local to one checkout and never committed: add `.state/` to `.git/info/exclude`.
- Before any tool call that changes a file, read the end of the log to check the current state
  and any locks.

**Records.** Every line has `ts` (RFC 3339 UTC), `task_id`, `agent` and `status`.
- `agent`: `claude-code`, `agy-argon` or `cursor`. Add `model` when it helps
  (`"model":"composer-2.5"`); locks follow `files`, whatever the agent or model. Add
  `"role":"planner"` while Antigravity stands in (§8).
- `task_id`: the Work Order ID, plus a step for implementers (`WO-0102-A/discovery`), or a short
  name for the planner's work (`review-pr-3`).
- `status`: `PENDING`, `RUNNING`, `COMPLETED` or `FAILED`.

**Lifecycle.**
1. **Claim.** Before starting a `PENDING` task, append `RUNNING` with your `agent` and the
   `files` you will change.
2. **Execute.** Make minimal-diff edits. Run commands non-interactively (`CI=true`).
3. **Record.** Append `COMPLETED` with `exit_code` 0 and a short git or file hash (`hash`), or
   `FAILED` with the exit code and the exact error text (`error`).
4. **Halt on failure.** After `FAILED`, stop. Don't wipe earlier state, and don't retry outside
   what the log records. Resuming means a new `RUNNING` line for the same `task_id`.

**Locks.**
- A `RUNNING` record with no later `COMPLETED` or `FAILED` for its `task_id` locks its `files`.
  Don't change a locked file.
- Only the agent that holds a lock releases it, by appending its outcome. A lock left by a
  crashed run is released by the repo owner, or by that agent after the owner's go-ahead,
  appending `FAILED` with `error` `stale lock`.
- The log only locks within one checkout. Across machines, the GitHub issue claim (§4.2) and
  the rule that two Work Orders on the same package never run at once
  ([`planning/README.md`](planning/README.md#sizing)) remain the lock.

Example:

```json
{"ts":"2026-10-27T07:30:00Z","task_id":"WO-0102-A/discovery","agent":"cursor","status":"RUNNING","files":["aws/org.go"]}
{"ts":"2026-10-27T07:42:10Z","task_id":"WO-0102-A/discovery","agent":"cursor","status":"COMPLETED","exit_code":0,"hash":"3f2a1bc"}
```

# Part B — Planner and reviewer

## 8. Planner and reviewer duties

**Who.** Claude Code by default. **Antigravity (`agy-argon`) stands in** when the repo owner
says so, in the chat, an issue or a PR, for a named scope: one Work Order, one PR, one
Deliverable, or until the owner says Claude is back ([D-029](planning/DECISIONS.md#d-029)).

**Duties.** The planner never writes implementation code, even when asked to "just fix"
something; it writes or amends the Work Order instead. It also:

- grills the repo owner (3–5 questions per turn) before a new Deliverable, Work Item, schema or
  rule, and records the answers in `planning/DECISIONS.md`;
- dispatches a Work Order as a GitHub issue only on the repo owner's go-ahead;
- reviews every PR against its Work Order's *Done when* only, checks its first two lines
  (`Implementer:`, `Resolves #N`) against the claim and that no reference case was edited, and
  posts the verdict as a GitHub review, leading with approved or changes requested;
- keeps `planning/STATUS.md` current whenever an issue or PR changes state;
- when a Deliverable's last Work Order merges, writes each Work Item's `Delivered (as built)`
  section.

**Standing in.** While Antigravity stands in:

- It follows this Part in full and writes what Claude writes. It writes no implementation code
  until the stand-in ends.
- **No one reviews their own work.** It never reviews a PR it implemented. A Work Order it
  specified or dispatched while standing in is implemented by Cursor, unless the repo owner
  says otherwise.
- It marks what it produces: reviews start with `Reviewer: Antigravity (standing in for Claude
  Code)`; decisions say `recorded by Antigravity, standing in`; state records carry
  `"agent":"agy-argon","role":"planner"`.
- When the stand-in ends, the planner's work stays as it is. Claude picks up from
  `planning/STATUS.md`, `planning/DECISIONS.md` and the open issues and PRs; it re-reviews
  something only when the repo owner asks.

# Part C — Implementers

## 9. Implementer rules

For Antigravity and Cursor while they build a claimed Work Order. They add to §4 and §7.

1. **Claim first.** Claim the Work Order on its GitHub issue (§4.2), then log its steps in
   `.state/tasks.jsonl` and claim each step with `RUNNING` before starting it (§7). Log as
   `agy-argon`, or as `cursor` with your `model`.
2. **Narrow working set.** Never search the whole repository. Read the end of
   `.state/tasks.jsonl`; then open only `AGENTS.md`, the Work Order, its Work Item's
   **Contracts**, the reference cases it names, and the files in your step's `files`, inside the
   Work Order's **Packages**.
3. **Respect locks.** Never edit a file listed in another active `RUNNING` record, whoever holds
   it.
4. **Minimal diffs.** Make surgical changes. Never reformat unrelated blocks or rewrite whole
   files. Leave the result `gofmt` clean.
5. **Verify.** Per step, run only the tests for the code you changed, non-interactively, with
   output capped: `CI=true go test ./<pkg> -run <Test> 2>&1 | tail -n 50`. Before the PR, run
   the full `Verify` command uncapped; its output goes in the PR (§4.4–4.5).
6. **Record and stop.** Append `COMPLETED` or `FAILED` (§7) and stop. On `FAILED`, don't retry.
   Propose no refactors nobody asked for. A spec that looks wrong is reported, not guessed
   around (§2).
