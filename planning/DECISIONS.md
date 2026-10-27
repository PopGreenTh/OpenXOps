# Decisions

Newest last. A decision is changed only by a later decision that says so. D-001 to D-012 were
agreed by the repo owner on 2026-10-27, in the session that closed OpenLZ.

## D-001

**OpenXOps is a fresh start.**

- A standalone, single-tenant tool for cloud operations, open source (changed to
  source-available by D-023).
- Nothing is carried over from OpenLZ: no code, no plans, no reference cases. OpenLZ is archived
  and read for intent only.
- Go, embedded storage only (DuckDB allowed), no database servers.

## D-002

**Four co-creators with fixed roles.** The repo owner decides and approves; Claude Code plans and
reviews and never writes implementation code; Antigravity and Cursor implement. Work Orders are
dispatched as GitHub issues, `Assignee: open` by default, and claimed by comment (AGENTS.md §4).

## D-003

**Planning hierarchy.** Deliverable (one release) › Work Item (one domain stage, or one piece of
work at the same level) › Work Order (one PR) › Task (the implementer's own, never written down).
Decisions stay in their own file.

## D-004

**Several binaries; commands are `<binary> <stage> <thing>`.**

- `xops` holds commands shared by every domain, such as `xops sense aws-accounts` and
  `xops report …`. `finops` is the first domain binary; `secops` and `cloudops` come later.
- The five stages run **Sense › Enrich › Analyze › Enforce › Report**. Spelling: `analyze`.

| Stage | Job |
|---|---|
| Sense | Read a cloud (or `--mock`) and emit raw records |
| Enrich | Add taxonomy context: who owns the cost, how shared cost splits |
| Analyze | Compare and grade over time |
| Enforce | Flag what breaks a rule, with the action to take |
| Report | Turn records into tables people read |

## D-005

**The cloud is part of the thing's name.** `aws-accounts`, `aws-cost`, `aws-focus`; later
`azure-…`, `gcp-…`, `oci-…`. Each command has its own cloud's flags (`--profile`, …) and
`--lz CODE` narrows a run to one landing zone. Every cloud's command emits the same record
kinds, so every stage after Sense works for any cloud.

## D-006

**Records and inputs.**

- Records are flat JSON Lines: one object per line, no nested objects or arrays, a `kind` on
  every record. Only Report writes other formats.
- `--from-<kind> FILE` reads a record file (`--from-cost cost.jsonl`). Records can also come
  through the pipe; a command knows them by `kind`. The same kind from both the pipe and a flag
  is an error.
- `--from-history FILE` is the same command's own earlier output, used as its baseline.
- `--taxonomy FILE` names the taxonomy YAML.

## D-007

**History and cache.**

- Local only, under `data/` (gitignored). Runs happen by hand and by cron.
- Cost Explorer answers are cached: a closed month is fetched once, the open month is reused for
  12 hours. Each call costs $0.01, so every call goes through the cache.
- Cost history reaches back at least 24 months (monthly), which needs multi-year data turned on
  in AWS Billing.
- Restated by default (today's taxonomy for every period); `--as-reported` uses the taxonomy and
  organization saved for each period, so Enrich saves a copy of each taxonomy it uses.

## D-008

**Reading AWS cost.**

- **Cost Explorer, two levels** (CE groups by at most two dimensions per call): one call by
  account × `Project` tag, then one call per account by service × `App` tag. Monthly defaults to
  `UnblendedCost`, daily to `AmortizedCost`; `--metric` overrides.
- **FOCUS** for everything Cost Explorer can't give in those two calls (region, `Feature` and
  other tags): `finops sense aws-focus` reads AWS's FOCUS export, and `finops sense focus-cost`
  turns FOCUS rows into the same `cost` records that `map` and later stages read. Confirmed by
  D-014.
- `Feature` comes from a resource tag, like `App` and `Project`.

## D-009

**The taxonomy mapping wins over tags.** An account's Project, App and Environment come from the
taxonomy mapping (YAML in git). A `Project` tag that disagrees is reported, not used.

## D-010

**Enrich, Analyze and Enforce commands.**

- **Enrich:** `map` (who owns each account) and `share` (how a shared account's cost splits) are
  separate commands.
- **Analyze:** `trend` (how cost moved against its baseline), `changes` (what moved in the
  mapping) and `impact`, which reads both, applies a floor and a factor, grades levels and marks
  each new, ongoing or resolved against `--from-history`.
- **Two hierarchies** for `trend` and `impact`, chosen with `--hierarchy cost|project`:
  cost = OU › account › region › service; project = Project › App › Environment › Feature.
  `--floor-factor FILE` holds the floor and factor for each hierarchy, the most specific node
  winning.
- **Enforce** flags records that break a rule, with what is wrong, the cost it touches, the
  action and the owner. It never changes the cloud; one command per rule, several per run.

## D-011

**Report** (`xops report`) writes a table, CSV or Excel, one table per record kind. Writing to an
existing file appends rows under the existing header.

## D-012

**Layout.** One folder per binary and stage (`finops/enrich/`), cloud clients in `aws/`, the
shared record code in `record/`; each package keeps its own `testdata/`, reference cases
included. No `cmd/` or `internal/`.

## Q-001

**Closed** by [D-014](#d-014) to [D-018](#d-018). What the first release needs.

1. FOCUS: is D-008's reading right — `aws-focus` reads the export from S3, `focus-cost` turns any
   FOCUS file into `cost` records? Is `focus-cost` shared (`xops`), since FOCUS is the same for
   every cloud?
2. Account mapping rules: as worked out in OpenLZ (an account has one Environment and may serve
   several Apps; OU links inherited, deepest OU wins; account entry › OU › LZ default › global
   default › `UNALLOC`)? Or decided again here?
3. `share`: a shared account's cost splits by the App tag; untagged cost split equally between
   its Apps; a wrong tag to `APP-INVALID` — as in OpenLZ?
4. `impact` levels: still four, doubling from the floor (factor ≥ 2, 4, 8, 16), with the floor-factor
   file overriding floor and factor per node?
5. First release scope: all of Sense, Enrich, Analyze, Enforce and Report for FinOps on AWS, or a
   smaller first slice (for example Sense and Enrich only)?

## D-013

**Developers can run, debug and change any command on its own** (repo owner, 2026-10-27; the
details are the planner's, Q-002 confirms them).

- **Every command runs alone from files.** With `--from-<kind> FILE` and `--mock`, any command
  can be started in a debugger with no pipe and no cloud access.
- **Stage logic is plain functions**: records in, records out, no I/O, no globals. The command
  code only reads flags and files and calls them. A breakpoint in the logic needs no AWS and no
  CLI setup.
- **Ready-made debug setups**: `.vscode/launch.json` has one entry per command, fed by the
  package's `testdata/` fixtures, with the right build tags, so Delve works on Windows, macOS and
  Linux.
- **Logs go to stderr, data to stdout.** `--log-level debug|info|warn|error` (default `warn`,
  Go `log/slog`) on every command; at `debug` each command logs its inputs, its major parameters
  and every cloud or cache call. Errors name the file and line number of the record that failed.
- **One task runner** for the common steps (build, test, run a command on the mock, debug), and
  `docs/develop.md`: how to build, run, debug, add a command, and add a reference case.
- **Plain Go**: no code generation, no reflection-driven frameworks, no hidden init order. Small
  files; a package has at most 5 non-test `.go` files.

## Q-002

**Closed** by [D-019](#d-019) (items 1–3) and [D-020](#d-020) (item 4). Developer setup.

1. Editor: VS Code (with `launch.json`), GoLand, or both?
2. Your OS for development: Windows, macOS or Linux? (Decides how the DuckDB/CGO build is set up.)
3. Task runner: `Taskfile.yml` (Task, as OpenLZ v1 used), a `Makefile`, or plain `go run ./scripts/...`?
4. A dev container (Docker, with Go, DuckDB and the AWS CLI ready), or local installs only?

## D-014

**`focus-cost` is shared** (repo owner, 2026-10-27; Q-001 item 1). `finops sense aws-focus` reads
AWS's FOCUS export from S3; `xops sense focus-cost` turns any FOCUS file, from any cloud, into
`cost` records. FOCUS is the same for every cloud, so the command sits in `xops`.

## D-015

**Account mapping rules** (repo owner, 2026-10-27; Q-001 item 2: keep the rules worked out
before). Stated here in full; nothing refers back to OpenLZ.

- **Landing zone.** An organization's LZ is the taxonomy LZ whose management account matches.
  None → `LZ-UNALLOC`, and all its accounts are `UNALLOC`.
- **OU links.** An OU is named by ID (`ou-…`) or by name path (`/Workloads/Checkout`; a value
  starting with `/` is a path). A link may set an App (only on an OU that has child OUs) and an
  Environment (any OU). Links are inherited down the tree; the deepest OU that says something
  wins. The OU tree is never flattened.
- **App**, first answer wins: the account's own entry › the deepest OU with an App link › the
  LZ default › the global default › `APP-UNALLOC` (Project `PRJ-UNALLOC`).
- **Environment**, resolved separately in the same order; nothing found → `ENV-UNSPEC` if the
  App resolved, `ENV-UNALLOC` if not. An Environment must be defined under the App it is used
  with, else `ENV-UNSPEC`.
- **One Environment per account; an account may serve several Apps** (only through its own
  entry). Such an account is **shared**: its cost is split by `share` (D-016).
- Project = the App's Project; BusinessUnit = the Project's; CostCenter = the App's.
- Every decision records the rule that made it (`account`, `ou:<path>`, `lz-default`,
  `default`, `unalloc`), and every miss raises a finding saying why.

## D-016

**Share rules** (repo owner, 2026-10-27; Q-001 item 3).

- Only a **shared** account's cost is split; a single-App account's cost goes whole to its App,
  whatever its tags.
- The `App` tag value is matched to the account's App codes ignoring case and surrounding
  spaces (`app-chk` = `APP-CHK`).
- **Several Apps in one tag:** a value may list Apps separated by `,` or `__`
  (`APP-A,APP-B`, `APP-A__APP-B`). The row is split equally between the listed values, and each
  value is then judged on its own as below (confirmed by D-021).
- A value matching one of the account's Apps → that App. **Untagged** → split equally between
  all the account's Apps, with finding `shared-untagged`. A value naming a real App the account
  doesn't serve → `APP-INVALID` (finding `shared-tag-wrong-app`); naming no App at all →
  `APP-INVALID` (finding `shared-tag-unknown`).
- In every equal split the Apps are ordered by code and the last takes the remainder, so the
  parts add back up exactly. Total cost in equals total cost out for every period, metric and
  currency, or the command fails.

## D-017

**Impact levels** (repo owner, 2026-10-27; Q-001 item 4). Baseline = average of the 3 periods
before; factor = amount ÷ baseline. A movement counts only when it is at least the **floor**
above the baseline (default 30). Four levels: factor ≥ 2, ≥ 4, ≥ 8, ≥ 16. The floor-factor file
overrides floor and factor per hierarchy node, the most specific node winning.

## D-018

**Release 1 is a smaller first slice** (repo owner, 2026-10-27; Q-001 item 5). *Planner's
proposal*, Q-003 item 2 confirms it:

- **Release 1:** the skeleton and developer setup, `xops sense aws-accounts`,
  `finops sense aws-cost`, `finops enrich map`, `finops enrich share`, `xops report`.
  Result: AWS cost owned by Project › App › Environment, in Excel.
- **Release 2:** FOCUS (`aws-focus`, `focus-cost`), Analyze (`trend`, `changes`, `impact`),
  Enforce, history and `--as-reported`.

## Q-003

**Closed** by [D-021](#d-021). Confirm the split and the slice.

1. A tag `APP-A,APP-B` splits the row equally (half each), each half then judged alone, so
   `APP-A,APP-WRONG` gives half to APP-A and half to `APP-INVALID`. Right?
2. Release 1 = skeleton, `aws-accounts`, `aws-cost`, `map`, `share`, `report`; Release 2 =
   FOCUS, Analyze, Enforce, history. Right?

## D-019

**VS Code on Windows first; Task as the task runner** (repo owner, 2026-10-27; Q-002 items 1–3).

- **Editor:** VS Code. `.vscode/launch.json` (one debug entry per command), `settings.json` and
  `extensions.json` (the Go extension) are part of the repository. Other editors aren't
  configured, but nothing stops them.
- **OS:** Windows first, then Linux and macOS. Every Work Order's `Verify` must pass on Windows;
  CI runs it on Windows and Linux (macOS when a Work Order needs it). The DuckDB/CGO build and
  Delve debugging must work on Windows, with the toolchain written down in `docs/develop.md`.
- **Task runner:** Task (`Taskfile.yml`), with tasks that work in PowerShell and in bash: at
  least `build`, `check` (the Verify command), `test`, `run:<command>` on the mock, and `clean`.

## D-020

**A dev container as an optional extra** (repo owner, 2026-10-27; Q-002 item 4). Native Windows
stays the main path (D-019); `.devcontainer/` gives a ready Linux environment (Go, Task, AWS
CLI, Delve) for VS Code "Reopen in Container" and GitHub Codespaces.

## D-021

**Multi-App tags split equally; Release 1 adds `analyze changes`** (repo owner, 2026-10-27;
Q-003).

- `APP-A,APP-B` splits the row equally, each part judged on its own: `APP-A,APP-WRONG` gives
  half to `APP-A` and half to `APP-INVALID`.
- Release 1 = D-018's slice **plus `finops analyze changes`**: what changed between two `map`
  outputs (accounts added, removed, moved between OUs, remapped to another App or Environment)
  and the cost that moved with it. Trend, impact, Enforce, FOCUS and the history store stay in
  Release 2.

## D-022

**Release 1 links no DuckDB** (planner, 2026-10-27). Nothing in Release 1 needs SQL or Parquet:
the Cost Explorer cache is JSON files under `data/cache/`. DuckDB, its CGO toolchain and the
`no_duckdb_arrow` tag arrive with FOCUS in Release 2, so Release 1 builds with plain `go build`
on Windows.

## D-023

**Free to use for every organization, closed to resale; the repo owner keeps the right to sell**
(repo owner, 2026-10-27).

- Anyone may contribute. Any person or company may use OpenXOps free of charge for their own
  organization's clouds.
- Nobody may sell OpenXOps, or offer it or a product built from it as a paid product or hosted
  service, without the repo owner's permission.
- The repo owner keeps the right to sell it in future (a hosted version, commercial licences),
  so outside contributions will need a contributor licence agreement (CLA) before the first
  one is merged.
- This makes OpenXOps **source-available**, not open source in the OSI sense; it supersedes
  D-001's "open source". The licence is D-024.

## Q-004

**Closed** by [D-024](#d-024). The licence (planner, 2026-10-27). BSL 1.1 parameters: Licensor, Additional Use Grant
wording, Change Date (years until each version becomes open source), Change License (Apache 2.0
or another); and when the CLA is set up.

## D-024

**Licence: Functional Source License 1.1, Apache 2.0 future licence (`FSL-1.1-Apache-2.0`)**
(repo owner, 2026-10-27; answers Q-004).

- Licensor: **Chanin Sainumkheaw**. Copyright line: `Copyright 2026 Chanin Sainumkheaw`.
- Any purpose is permitted except a Competing Use (making OpenXOps, or something substantially
  similar, available commercially as a product or service). Internal use and professional
  services for a client (consultants) are permitted.
- Each released version becomes Apache 2.0 two years after it is made available.
- The repo owner adds `LICENSE` with the published FSL-1.1-Apache-2.0 text, unchanged except for
  the licensor and year. The README says in one line: free for your own use; contact the
  licensor for commercial use.
- A contributor licence agreement is set up before the first outside contribution is merged
  (D-023).

## D-025

**Linux first, developed with Cursor; Windows and the dev container are not required for now**
(repo owner, 2026-10-27). Supersedes D-019's "Windows first" and D-020.

- Development and debugging happen on Linux, with Cursor. A Work Order's `Verify` must pass on
  Linux; CI keeps its Windows job as a free early warning, but nothing waits on a person
  checking Windows.
- No Done-when asks for a Windows walkthrough or a dev container run. Existing `.devcontainer/`
  files may stay as an unverified extra.
- VS Code debug entries, Task and `docs/develop.md` stay (D-013, D-019); `docs/develop.md`
  covers Linux first.

## D-026

**The Windows CI job is optional** (repo owner, 2026-10-27; makes D-025 explicit in CI).

- CI runs `scripts/check` on `ubuntu-latest` (required) and on `windows-latest` (optional): the
  Windows job is marked `continue-on-error: true` and named `check (windows-latest, optional)`,
  so a Windows failure shows as a warning and never blocks a PR.
- If branch protection lists required checks, only the Linux job is required.

## D-027

**One state protocol for every agent** (repo owner, 2026-10-27). `AGENTS.md` §7 is the rule. The
repo owner's text was adjusted to fit the existing rules:

- **Covers Cursor too.** The owner's text named Claude Code and Antigravity (`agy-argon`); Cursor
  is an implementer (D-002), so it gets `agent` `cursor`.
- **Doesn't change roles.** Claude logs its planning and review steps but still writes no
  implementation code, and the issue claim still decides who builds a Work Order.
- **Stale locks.** A `RUNNING` record from a crashed run would lock its files for good. The repo
  owner releases it, or the same agent does with the owner's go-ahead, by appending `FAILED`
  with `stale lock`.
- **Local only.** `.state/tasks.jsonl` is never committed: a log shared through git would
  conflict on every branch. So it locks within one checkout. Across machines, the issue claim
  and the one-Work-Order-per-package rule remain the lock, and the issue, branch and PR remain
  the record across sessions.
- **Fixed field names**: `ts`, `task_id`, `agent`, `status`, `files`, `exit_code`, `hash`,
  `error`.

This replaces the first draft of D-027 (an implementer-only protocol), which was never merged.

## D-028

**A Cursor rule that holds Composer 2.5 to execution** (repo owner, 2026-10-27).
`.cursor/rules/implementor.mdc` is applied to every Cursor request. It points to `AGENTS.md` §4
and §7, which win, and narrows execution: no repository-wide search, minimal diffs, logged
state, stop after each task. Claude writes `.cursor/rules/`; implementers don't edit it. The
repo owner's text was adjusted to fit the existing rules:

- **Agent names stay as in §7.** The owner's text used `cursor-composer-2.5` and
  `agy-gemini-3.8-flash`. The rule uses `agent` `cursor` with `model` `composer-2.5` (§7 now
  allows a `model` field). Every active lock counts, including another Cursor run's.
- **The spec is always read.** Besides the task's files: `AGENTS.md`, the Work Order, its
  Contracts and its reference cases.
- **The issue claim comes first.** Logging `RUNNING` doesn't replace claiming the Work Order on
  its GitHub issue (§4.2). If no `PENDING` task is logged yet, the implementer logs the Work
  Order's steps first.
- **`Verify` still runs in full.** The 50-line cap is for targeted tests; the full `Verify`
  runs uncapped before the PR, and its output goes in the PR.

## D-029

**`AGENTS.md` in three parts; Antigravity can stand in as planner and reviewer** (repo owner,
2026-10-27).

- **One file, three parts**: A for everyone (§1–7), B for the planner and reviewer (§8), C for
  implementers (§9). `CLAUDE.md`, `GEMINI.md` and `CURSOR.md` point to the
  parts their agent needs. A real split into separate files waits until Part B or C passes about
  100 lines. `CURSOR.md` holds the summary of §9 (which wins), and
  `.cursor/rules/implementor.mdc`, the file Cursor applies to every request, only points to it
  (amends D-028).
- **Stand-in** (amends D-002): Antigravity (`agy-argon`) takes the planner and reviewer role
  when the repo owner says so, for a named scope. While standing in, it follows Part B, writes
  what Claude writes and writes no implementation code. It never reviews a PR it implemented,
  and a Work Order it specified or dispatched is implemented by Cursor unless the owner says
  otherwise. Its reviews, decisions and state records say it is standing in. Claude resumes from
  `STATUS.md`, `DECISIONS.md` and the open issues and PRs, without re-reviewing unless asked.
- **Confirmed by the repo owner (2026-10-27):** the stand-in never reviews its own PR.
  *Planner's defaults, still to confirm:* a stand-in starts on the owner's word in any channel,
  and Cursor implements what the stand-in specified.

## D-030

**When a command reads stdin** (repo owner, 2026-10-27; amends D-006 and WI-0101's *Inputs*). A
command reads stdin only when it is given no `--from-<kind>` flag, or when a flag's `FILE` is
`-`. A command whose inputs all come from files never touches stdin, so an open, empty stdin
(cron, CI) can't make it hang. Mixing a pipe with files is explicit:
`… | finops enrich map --from-accounts a.jsonl --from-cost -`. Found while testing WO-0104-A.
Built by WO-0101-B.

## D-031

**One follow-up Work Order for the review notes** (repo owner, 2026-10-27). WO-0101-B carries
D-030 and the non-blocking notes from the WO-0102-A, WO-0104-A and WO-0106-A reviews. It is
dispatched after WO-0103-A, WO-0105-A and WO-0107-A merge, because it touches `record/`, which
they all use.

## Q-005

**Release 2 scope** (planner, 2026-10-27). **Closed** by [D-032](#d-032) to [D-036](#d-036).

1. Enforce: which rules come first, and what is each rule's action and owner?
2. History store: record files per run, DuckDB tables, or both? How long is history kept?
3. FOCUS: where does the export live? Does `aws-cost` stay alongside FOCUS, or does FOCUS
   replace it for detailed views?
4. Trend: monthly only, or daily too? Which baseline when fewer than 3 periods exist?
5. Release 2 slice: all of it in one release, or split?

## D-032

**Release 2 is FOCUS, history, trend and impact; Release 3 is Enforce** (repo owner, 2026-10-27;
Q-005 item 5; amends D-018 and D-021). Enforce moves to Release 3 because it grades better once
impact exists.

## D-033

**The first Enforce rules, for Release 3** (repo owner, 2026-10-27; Q-005 item 1). Each rule is
its own command and flags only (D-010):

| Rule | Flags | Action | Owner |
|---|---|---|---|
| untagged shared cost | a shared account's cost with no App tag | tag the resources | the account's Apps |
| unallocated cost | `UNALLOC` cost above a threshold | map the account in the taxonomy | FinOps |
| Project tag mismatch | a `Project` tag that disagrees with the taxonomy | fix the tag or the taxonomy | the Project |

Thresholds and the record format are decided when Release 3 is grilled.

## D-034

**History is record files; DuckDB reads them** (repo owner, 2026-10-27; Q-005 item 2; amends
D-007). Each saved run is a JSON Lines file, `data/history/<command>/<run timestamp>.jsonl`,
written once and never edited. DuckDB (Release 2, D-022) reads them on demand; no table is kept
in step. History is kept forever.

## D-035

**FOCUS is opt-in; trend is monthly on average daily cost** (repo owner, 2026-10-27; Q-005
items 3 and 4).

- **FOCUS:** `finops sense aws-cost` stays the default. `aws-focus` and `focus-cost` are opt-in,
  for region and `Feature` detail. Each LZ in the taxonomy gets an optional `focus` entry
  naming the export's bucket and prefix.
- **Trend is monthly only** in Release 2, and compares **average daily cost**: a month's amount
  divided by its number of days, so months of different lengths compare fairly. The open
  (current) month divides by the days that have cost so far. The baseline is the average of the
  3 months before, as daily averages (D-017); with fewer, the months there are; with none,
  finding `trend-no-baseline` and no grade. `impact` grades the same daily averages.

## D-036

**Release 2 scope, in sum** (planner, 2026-10-27; from D-032 to D-035). `finops sense
aws-focus`, `xops sense focus-cost`, DuckDB with `no_duckdb_arrow`, the history files and
`--as-reported`, `finops analyze trend` (monthly, average daily cost) and `finops analyze impact`.
Not in Release 2: Enforce (Release 3), daily trend, other clouds.

## D-037

**Every input record is checked against its kind's schema** (repo owner, 2026-10-27).

- One schema per record kind, a plain Go table in `record/` (field, type, required). Each Work
  Item's Contracts table stays the readable source; a test asserts every kind there has a schema.
- A missing required field or a value of the wrong type stops the command, naming the file (or
  `stdin`), the line and the field. **Extra fields are allowed**, because records gain fields
  down the pipeline.
- Only the kinds a command reads are checked; other kinds pass through unchecked.
- The check is the same for the pipe and for `--from-<kind>` files.
- Built in WO-0101-B with the D-030 pipe rule, after WO-0103-A, WO-0105-A and WO-0107-A merge.

## D-038

**FOCUS files are cached locally** (repo owner, 2026-10-27; Release 2). `aws-focus` downloads
each closed month's FOCUS Parquet files once into `data/cache/aws/focus/`; DuckDB reads them
from there. The open month is fetched again on each run, as for the Cost Explorer cache (D-007).

## D-039

**History is saved with `--save`** (repo owner, 2026-10-27; Release 2; with D-034). `map`,
`share`, `trend` and `impact` take `--save`, which writes the run's output to
`data/history/<command>/<run timestamp>.jsonl` as well as stdout. Reading history stays
`--from-history FILE`.

## D-040

**`--as-reported` is in Release 2** (repo owner, 2026-10-27; D-007). It uses the taxonomy and
organization saved for each period instead of today's.

## D-041

**How deep trend goes** (repo owner, 2026-10-27; Release 2; D-010). Cost hierarchy: OU ›
account › region › service, with region only when FOCUS is the input. Project hierarchy:
Project › App › Environment from `aws-cost`, and › Feature only when FOCUS is the input.

## Q-006

**Planner defaults in DD-02, to confirm** (planner, 2026-10-27). **Closed** by [D-042](#d-042).

1. **Impact floor unit:** the floor (default 30) applies to the monthly equivalent of the rise:
   (`daily_avg` − baseline) × the calendar days of the month (WI-0204).
2. **Trend baseline:** a node's baseline uses only earlier periods in which the node has cost; a
   node whose latest period has none gets `trend-no-baseline` (WI-0203).
3. **FOCUS hand-off:** `aws-focus` emits `focus_file` records (local cached paths), not FOCUS rows;
   `focus-cost` reads the files with DuckDB; the mock FOCUS data is CSV (WI-0202).
4. **Change explains impact:** a change touches a cost node through its account, and a project node
   when its `old` or `new` value equals the node's value for the changed field (WI-0204).
5. **As reported:** a period uses the newest snapshot saved before its end; `allocation` records are
   written per snapshot used, and `share` matches on `snapshot` (WI-0205).

## D-042

**The DD-02 planner defaults stand** (repo owner, 2026-10-27; closes Q-006). All five items of
[Q-006](#q-006) are decided as written there:

1. The impact floor applies to the monthly equivalent of the rise: (`daily_avg` − baseline) ×
   the calendar days of the month (WI-0204).
2. A node's trend baseline uses only earlier periods in which the node has cost (WI-0203).
3. `aws-focus` emits `focus_file` records; `focus-cost` reads the files with DuckDB; the mock
   FOCUS data is CSV (WI-0202).
4. A change touches a cost node through its account, and a project node when its `old` or `new`
   equals the node's value for the changed field (WI-0204).
5. As reported, a period uses the newest snapshot saved before its end; `allocation` records are
   written per snapshot, and `share` matches on `snapshot` (WI-0205).

DD-02 Work Orders are dispatched once Release 1 and WO-0101-B have merged, on the repo owner's
go-ahead.

## Q-007

**Drop the stdin sort by kind?** (raised by Cursor, kept by the repo owner, 2026-10-27; blocks the
Release 1 tag). WI-0101 *Inputs* says "Records from stdin are sorted by `kind`". As a result, a
piped run passes records through in kind order and a file run in file order: the same records,
in a different order. Dropping the sort would make piped and file runs byte-identical. Open.

## Q-008

**`scripts/check` leaves a mock cache in `data/mock/`** (raised by Cursor, kept by the repo owner,
2026-10-27). The cache is gitignored and harmless, but a check run is not side-effect free.
Options: point the check's mock runs at a temporary cache, or accept it. Open.
