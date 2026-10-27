# Decisions

Newest last. A decision is changed only by a later decision that says so. D-001 to D-012 were
agreed by the repo owner on 2026-10-27, in the session that closed OpenLZ.

## D-001

**OpenXOps is a fresh start.**

- A standalone, single-tenant tool for cloud operations, open source.
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
  turns FOCUS rows into the same `cost` records that `map` and later stages read. *Planner's
  reading* of "aws-focus and focus-cost"; Q-001 item 1 confirms it.
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

**Open: what the first release needs** (planner, 2026-10-27).

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
