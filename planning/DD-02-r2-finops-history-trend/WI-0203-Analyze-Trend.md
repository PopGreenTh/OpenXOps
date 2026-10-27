# WI-0203 — `finops analyze trend`

## Outcome

How each node of a hierarchy moved this month against its own recent past, compared on **average
daily cost** so months of different lengths compare fairly (D-035).

## Contracts

### `finops analyze trend --hierarchy cost|project [--as-of DATE] [--from-cost FILE] [--save]`

Reads `cost` records (after `map` or `share`) and, for the cost hierarchy, `account` records
(their `ou_path`), from the pipe or `--from-cost`. Writes `trend` records, then findings, then the
input records (pass-through).

### Hierarchies and nodes (D-010, D-041)

| `--hierarchy` | Levels, top to bottom | Fields |
|---|---|---|
| `cost` | `ou` › `account` › `region` › `service` | `ou_path` (the account's, from its `account` record), `account_id`, `region`, `service` |
| `project` | `project` › `app` › `environment` › `feature` | `project`, `app`, `environment`, `feature_tag` |

- `region` and `feature` are levels only when at least one input `cost` record has a non-empty
  `region` / `feature_tag` (that is, FOCUS input); otherwise they are skipped.
- A **node** is one combination of the fields from the top level down to its level. Its `node`
  field joins those values with ` > ` (`PRJ-SHOP > APP-CHK`). A `cost` record counts towards one
  node at every level.
- Nodes are kept apart by `metric` and `currency`, never summed across them.

### Trend records

One `trend` record per node, level, metric, currency and **period in which the node has cost**:

| Field | Value |
|---|---|
| `hierarchy`, `level`, `node` | as above |
| the level fields | the node's values down to its level; lower ones `""` |
| `period`, `metric`, `currency` | |
| `amount` | the node's cost in the period |
| `days` | the days counted: the month's length for a closed month; for the open month (the month of `--as-of`), the days before `--as-of` (`--as-of` 2026-10-11 → 10). An open month with 0 days gets no record |
| `open` | true for the open month |
| `daily_avg` | `amount` ÷ `days` |
| `baseline_periods` | how many of the node's previous periods form the baseline: the up to **3** most recent earlier periods **in which it has cost** (0 to 3) |
| `baseline_daily_avg` | the mean of those periods' `daily_avg`; `null` when `baseline_periods` is 0 |
| `factor` | `daily_avg` ÷ `baseline_daily_avg`; `null` when there is no baseline or it is 0 |

- `--as-of` (`YYYY-MM-DD`) defaults to today (UTC), read from a clock tests can set.
- Months later than `--as-of`'s month are an error naming the record.
- Sorted by `hierarchy`, `metric`, `currency`, level (top first), `node`, `period`.
- **Finding** `trend-no-baseline` (info, subject `<hierarchy>:<node>`) for each node whose latest
  period has no baseline (D-035); its trend records still appear, with `factor` `null`.

### Reference cases

`finops/analyze/testdata/cases/trend/*.yaml`, written by Claude: `args` (`hierarchy`, `as_of`),
input `records`, and `expect.trends`: the trend records to find, matched on `level`, `node`,
`metric`, `currency` and `period` and compared on the fields listed (numbers within `1e-6`, `null`
as written); trend records not listed are not checked. `expect.findings` (`code`, `subject`) is
the full set of findings. The test reads every case in place.

## WO-0203-A — Trend

**Assignee:** open
**Depends on:** WO-0201-A (for `--save`), WO-0105-A (share output as input)
**Packages:** `finops/analyze/` (the package then has at most 5 non-test files), `record/schema/`
(the `trend` schema),
`finops/analyze/testdata/` (new fixtures only, not `cases/`), `Taskfile.yml`,
`.vscode/launch.json`, `README.md`

**Goal.** `trend` as specified.

**In scope.**
- The command, both hierarchies, `--as-of`, `--save` (with `record/history`).
- Debug logging (`record.LogStart`); deterministic output; pass-through.
- Tests: every trend reference case; months later than `--as-of` are an error; two currencies stay
  apart.
- Task and launch entry on a fixture of `share` output.
- README: trend, with an example of why a short month doesn't read as a drop.

**Out of scope.** Daily trend (D-035). Grading (WI-0204).

**Done when.**
- Every `cases/trend/` case passes, unchanged.
- `finops analyze trend --hierarchy project` on the fixture prints trend records whose
  `daily_avg` for a 28-day February and a 31-day March of equal daily spend are equal (shown in the
  PR).

**Verify.** `go run -tags=no_duckdb_arrow ./scripts/check`
