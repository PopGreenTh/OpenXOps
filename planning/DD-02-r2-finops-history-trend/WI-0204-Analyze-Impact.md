# WI-0204 — `finops analyze impact`

## Outcome

Which movements matter: each node's rise is graded in four levels (D-017) on average daily cost
(D-035), marked new, ongoing or resolved against the last saved run, and tied to the mapping
changes that may explain it (D-010).

## Contracts

### `finops analyze impact [--from-trend FILE] [--from-change FILE] [--from-history FILE] [--floor-factor FILE] [--period P] [--save]`

Reads `trend` records (pipe or `--from-trend`) and, optionally, `change` records (`finops analyze
changes`) and the previous run's output (`--from-history`, an `impact` file). Writes `impact`
records, then the input records (pass-through).

### Grading (D-017, D-035)

For each `trend` record of the graded period (`--period`, default the latest period in the input)
with a baseline:

- `delta_monthly` = (`daily_avg` − `baseline_daily_avg`) × the number of days in the period's
  month (the calendar length, also for an open month). The **floor** (default 30, in the record's
  currency) applies to it: below the floor, the node is not graded.
- With **base factor** `f` (default 2), the levels are: `factor` ≥ `f` → 1, ≥ 2`f` → 2, ≥ 4`f` → 3,
  ≥ 8`f` → 4. Below `f`, not graded.
- Only rises are graded; a node whose cost fell is not graded.

### Floor-factor file (D-010, D-017)

```yaml
cost:
  - { ou_path: /Workloads, floor: 100 }
  - { account_id: "200000000001", service: AmazonEC2, factor: 3 }
project:
  - { project: PRJ-SHOP, floor: 500 }
  - { project: PRJ-SHOP, app: APP-CHK, factor: 3 }
```

- An entry applies to a node when every field it names equals the node's field.
- **Floor and factor are chosen separately:** each comes from the most specific applying entry that
  sets it (the one naming the most fields), else the default. Two equally specific applying entries
  that both set the same value are an error naming both.
- Unknown keys are an error (strict YAML).

### Impact records

One `impact` record per graded node, plus one per node `resolved` (below):

| Field | Value |
|---|---|
| `hierarchy`, `level`, `node`, the level fields, `period`, `metric`, `currency` | from the trend record |
| `daily_avg`, `baseline_daily_avg`, `factor`, `delta_monthly` | as above |
| `floor`, `base_factor` | the values used |
| `impact_level` | 1 to 4; 0 for `resolved` |
| `status` | `new`: not graded in the history; `ongoing`: graded in the history too; `resolved`: graded in the history, not now |
| `explained_by_change` | true when a `change` record touches the node (below) |

- History is matched on `hierarchy`, `node`, `metric` and `currency`. Without `--from-history`,
  every graded node is `new` and nothing is `resolved`.
- **A change touches a node** when its account (the `change` `subject` up to `/`) is the node's
  `account_id` (cost hierarchy), or its `old` or `new` value equals the node's value for the
  changed `field` (project hierarchy: `project`, `app`, `environment`).
- Sorted by `impact_level` (highest first), `hierarchy`, `node`, `metric`, `currency`.

### Reference cases

`finops/analyze/testdata/cases/impact/*.yaml`, written by Claude: `args` (`period`, optional
`floor_factor` inline YAML), input `records` (`trend`, `change`), optional `history` (`impact`
records), and `expect.impacts`: the **full** set of impact records, matched on `hierarchy`,
`node`, `metric`, `currency` and compared on the fields listed (numbers within `1e-6`).

## WO-0204-A — Impact

**Assignee:** open
**Depends on:** WO-0203-A, WO-0107-A
**Packages:** `finops/analyze/` (the package then has at most 5 non-test files), `record/schema/`
(the `impact` schema),
`finops/analyze/testdata/` (new fixtures only, not `cases/`), `Taskfile.yml`,
`.vscode/launch.json`, `README.md`

**Goal.** `impact` as specified.

**In scope.**
- The command, grading, the floor-factor file, history status, `explained_by_change`, `--save`.
- Debug logging; deterministic output; pass-through.
- Tests: every impact reference case; a floor-factor file with two equally specific entries
  setting the same value is an error; unknown keys are an error.
- Task and launch entry on a fixture of `trend` output.
- README: impact, the levels and the floor-factor file.

**Out of scope.** Enforce (Release 3).

**Done when.**
- Every `cases/impact/` case passes, unchanged.
- Two runs on the fixture, the second with `--from-history` the first's saved file and one node's
  cost settled, show that node `resolved` and the others `ongoing` (shown in the PR; release
  criterion 6).

**Verify.** `go run -tags=no_duckdb_arrow ./scripts/check`
