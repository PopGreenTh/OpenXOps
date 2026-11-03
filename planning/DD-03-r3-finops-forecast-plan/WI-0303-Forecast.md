# WI-0303 — `finops analyze forecast`

## Outcome

FinOps computes its own forecast from actuals: the run rate, plus the planned changes on top
([D-062](../DECISIONS.md#d-062)). The open month uses the days so far, so a forecast made on the
10th already accounts for what was spent; a new Project shows up from its go-live month, before
it has any cost.

## Contracts

### `finops analyze forecast --tenant ID [--from-cost FILE] [--from-plan FILE] [--as-of DATE] [--months N] [--save]`

Reads `cost` records after `map` or `share`, and optionally `plan` records (`plan-lines`'s
output; only `plan_type` `change` is used). Writes `forecast` records with `source` `runrate`,
then findings, then the input records (pass-through).

### Run rate (planner default, [Q-010](../DECISIONS.md#q-010) item 3)

- **The open month** is the month of `--as-of` (default today, UTC, from a clock tests can set).
  Its days elapsed are the days before `--as-of`, as in `trend` (WI-0203).
- **Nodes:** one per `service_urn` (the finest actual node), keeping all the URN columns of its
  `cost` records. Nodes are kept apart by `metric` and `currency`.
- **Base daily cost** of a node:
  - the open month's cost so far ÷ its days elapsed, when at least one day has elapsed;
  - else, on the first of a month, the last closed month's cost ÷ its days.
  - A node with no cost in that month has no base and no run-rate line.
- **Horizon:** the open month and the `--months` − 1 months after it (`--months` default 12).
- For each node and month in the horizon: one `forecast` record, `basis` `run_rate`, `amount`
  = base daily cost × `days_in_month`.
- For each `plan` record with `plan_type` `change` whose month is in the horizon: one `forecast`
  record, `basis` `planned_change`, with the plan's node, URN columns and `amount`. It isn't
  merged into a run-rate line: a change usually names a Project or App with no account yet.
- `daily_cost` = `amount` ÷ `days_in_month`; `line_id` is `""` for `run_rate` and the plan's
  `line_id` for `planned_change`.
- Records are sorted by `period`, `node_urn`, `basis`.
- **Finding** `forecast-no-base` (info, subject the node URN) for a node with cost in the last
  three closed months but none in the base month.

### Reference cases

`finops/analyze/forecast/testdata/cases/*.yaml`, written by Claude: `args` (`as_of`, `months`),
input `records` (`cost`, `plan`; run with `--tenant testtnt`), and `expect.forecasts`: the
**full** set of `forecast` records, matched on `period`, `node_urn`, `basis`, `metric` and
`currency` and compared on the fields listed (numbers within `1e-6`). `expect.findings` is the
full set of findings.

## WO-0303-A — Forecast

**Assignee:** open
**Depends on:** WO-0302-A (the `forecast` schema and `plan` records)
**Packages:** `finops/analyze/forecast/` (new), the command wired in
`finops/analyze/commands.go`, `finops/analyze/forecast/testdata/` (new fixtures only, not
`cases/`), `docs/user/commands.md`, `README.md`, `Taskfile.yml`, `.vscode/launch.json`

**Goal.** The command as specified.

**In scope.**
- Run rate, planned changes, the horizon, the finding, `--save`.
- Debug logging, deterministic output, pass-through.
- Tests: every forecast reference case; two currencies stay apart; a `budget` plan line is
  ignored.

**Out of scope.** Seasonality, trend-adjusted forecasts, daily forecasts.

**Done when.**
- Every `cases/` case passes, unchanged.
- On the mock pipeline with `--as-of` the 10th of the open month, a node's open-month forecast
  equals its cost so far ÷ 9 × the month's days, and the mock's new Project has
  `planned_change` lines from M+1 (shown in the PR).
- `go run -tags=no_duckdb_arrow ./scripts/check` all PASS, with `docs PASS`.

**Verify.** `go run -tags=no_duckdb_arrow ./scripts/check`
