# WI-0304 — `finops analyze variance` and the plan-vs-actual sheet

## Outcome

For each month and planned node, one record sets actual, run-rate forecast, manual forecast and
plan side by side, with the gaps to plan in money and percent, graded as `impact` grades a rise
and tied to the mapping changes that may explain it ([D-062](../DECISIONS.md#d-062)). A closed
month is judged against the plan as it stood before the month began
([D-065](../DECISIONS.md#d-065)). The workbook shows it as one pivot.

## Contracts

### `finops analyze variance --tenant ID [--from-cost FILE] [--from-forecast FILE] [--from-plan FILE] [--from-change FILE] [--floor-factor FILE] [--as-of DATE] [--history-dir DIR] [--save]`

Reads `cost`, `forecast` (both sources), `plan` and, optionally, `change` records. Writes
`variance` records, then findings, then the input records (pass-through).

### Nodes and values

- **Nodes:** every `node_urn` named by a `plan` or `forecast` record, per month, `metric` and
  `currency`.
- **Rolling up** (D-065): a record counts towards a node when its URN field of the node's type
  equals the node's URN (`app_urn` for an App node, `service_urn` for a service node; see *A node
  and its actuals* in DELIVERABLE.md). The same rule sums all four values:

  | Field | Sum of |
  |---|---|
  | `actual` | `cost` `amount`; 0 when there is none; for the open month, the cost so far |
  | `forecast_runrate` | `forecast` with `source` `runrate`, both `basis` values; `null` when there is none |
  | `forecast_manual` | `forecast` with `source` `manual`; `null` when there is none |
  | `plan` | `plan`, both `plan_type` values; `null` when there is none |

- **Which plan** (D-065): for a month that has begun by `--as-of` (closed or open), the plan is
  the `plan` records of the last `plan-lines` run saved **before the month's first instant**
  (UTC), read from the history store. Without such a run, the month uses `--from-plan`'s
  records, with finding `variance-plan-not-frozen` (warning, subject the period) once per month.
  Later months always use `--from-plan`.
- `month_state`: `closed`, `open` (the month of `--as-of`) or `future`.

### Gaps and grading (planner default, [Q-010](../DECISIONS.md#q-010) item 4)

- `gap_actual` = `actual` − `plan`, for a closed month only (else `null`).
- `gap_runrate` = `forecast_runrate` − `plan`; `gap_manual` = `forecast_manual` − `plan`.
- Each gap also has a `_pct`: the gap ÷ `plan`, or `null` when `plan` is `null` or 0.
- `leading_gap` is `gap_actual` for a closed month and `gap_runrate` otherwise. It is graded with
  `impact`'s rules and floor-factor file (WI-0204, D-067), using `factor` = (`plan` +
  `leading_gap`) ÷ `plan`:
  - below the floor, or with a gap under plan, the level is 0;
  - otherwise levels 1 to 4, at `factor` ≥ `f`, 2`f`, 4`f` and 8`f`.
- `explained_by_change`: true when a `change` record touches the node, as in WI-0204.

### `variance` records

`tenant`, `period`, `month_start`, `month_state`, `node_urn`, `node_type`, the URN columns, the
four values, the three gaps and their `_pct`, `leading_gap`, `floor`, `base_factor`,
`impact_level`, `explained_by_change`, `metric`, `currency`. Sorted by `period`, `impact_level`
(highest first), `node_urn`.

### The plan-vs-actual sheet

When `xops report excel` gets `variance` records, it also writes a sheet named `plan-vs-actual`:
- one row per `node_urn`, `metric` and `currency`, with `node_type`;
- for each period in the input, five columns: actual, run rate, manual, plan and leading gap
  (`2026-11 actual`, …), periods in order.

The `variance` sheet itself is written as for every kind (WI-0106).

### Reference cases

`finops/analyze/variance/testdata/cases/*.yaml`, written by Claude: `args` (`as_of`, optional
`floor_factor`), input `records` (`cost`, `forecast`, `plan`, `change`; run with `--tenant
testtnt`), optional `history` (the `plan` records of saved `plan-lines` runs, each with its run
`at`), and `expect.variances`: the **full** set of `variance` records, matched on `period`,
`node_urn`, `metric` and `currency` and compared on the fields listed (numbers within `1e-6`,
`null` as written). `expect.findings` is the full set of findings.

## WO-0304-A — Variance

**Assignee:** open
**Depends on:** WO-0303-A, WO-0204-A (grading), WO-0201-A (history)
**Packages:** `finops/analyze/variance/` (new), the command wired in
`finops/analyze/commands.go`, `record/schema/` (the `variance` kind),
`finops/analyze/variance/testdata/` (new fixtures only, not `cases/`), `docs/user/`
(`commands.md`, `records.md`, `findings.md`), `README.md`, `Taskfile.yml`, `.vscode/launch.json`

**Goal.** The command as specified.

**In scope.**
- Nodes, roll-up, the frozen plan from history, gaps, grading, `explained_by_change`, `--save`.
- Debug logging, deterministic output, pass-through.
- Tests: every variance reference case; two currencies stay apart; a plan edited after its
  month began doesn't change that month's `plan` (release criterion 6).

**Out of scope.** The workbook sheet (WO-0304-B). Variance of `cost` with no plan or forecast.

**Done when.**
- Every `cases/` case passes, unchanged.
- Two saved `plan-lines --mock` runs, the second with an edited amount for the open month, give
  that month the first run's `plan` (shown in the PR).
- `go run -tags=no_duckdb_arrow ./scripts/check` all PASS, with `docs PASS`.

**Verify.** `go run -tags=no_duckdb_arrow ./scripts/check`

## WO-0304-B — The plan-vs-actual sheet

**Assignee:** open
**Depends on:** WO-0304-A
**Packages:** `xops/report/` (a new file, or a subpackage if the package is at 5 files),
`xops/report/testdata/`, `docs/user/commands.md`

**Goal.** The sheet as specified.

**In scope.** The sheet, its column order, and a golden-file test on a small `variance` input.

**Out of scope.** Charts. The sheet in `table` and `csv`.

**Done when.**
- The golden test passes.
- The mock pipeline's workbook has a `plan-vs-actual` sheet with one row per planned node
  (shown in the PR; release criterion 2).
- `go run -tags=no_duckdb_arrow ./scripts/check` all PASS.

**Verify.** `go run -tags=no_duckdb_arrow ./scripts/check`
