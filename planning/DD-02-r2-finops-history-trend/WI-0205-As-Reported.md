# WI-0205 — Taxonomy snapshots and `--as-reported`

## Outcome

Cost can be shown as it was reported at the time, not only restated with today's taxonomy
(D-007, D-040). Restated stays the default.

## Contracts

### Snapshots (D-007)

- `map --save` also saves what it mapped with: the taxonomy file, byte for byte, as
  `data/history/taxonomy/<run timestamp>.yaml`, and its `ou` and `account` input as
  `data/history/inventory/<run timestamp>.jsonl`. Same timestamp as the run's `map` history file
  (WI-0201).

### `finops enrich map … --as-reported [--history-dir DIR]`

- For each `cost` record, the **snapshot of its period** is the newest snapshot saved before the
  end of that period (the first instant of the next month, UTC). That snapshot's taxonomy and
  inventory map the record, instead of today's.
- A period with no snapshot before its end is mapped with today's taxonomy and inventory, with
  finding `as-reported-missing` (warning, subject: the period), once per period.
- Every `cost` record gains `snapshot`: the snapshot timestamp used, or `""` for today's.
- `allocation` records are written for today's mapping (`snapshot` `""`) **and** for every
  snapshot used, each with its `snapshot`. `share` (WI-0105) splits a cost record using the
  allocations with the same `snapshot`.
- Without `--as-reported`, nothing changes: no `snapshot` field is added, and `share` behaves as
  before.

## WO-0205-A — Snapshots and `--as-reported`

**Assignee:** open
**Depends on:** WO-0201-A, WO-0105-A
**Packages:** `finops/enrich/commands.go`, `finops/enrich/map.go`, `finops/enrich/share/`,
`record/history/`, `finops/enrich/testdata/` (new fixtures only, not `cases/`), `README.md`

**Goal.** Snapshots on `map --save`, and `--as-reported` on `map` and `share`.

**In scope.**
- Saving the taxonomy and inventory snapshots; choosing a period's snapshot; the finding; the
  `snapshot` field; allocations per snapshot; `share` matching on `snapshot`.
- Tests (with `--history-dir` in `t.TempDir()` and a fixed clock): two snapshots where an account's
  App changed between them, and cost over periods before and after: each period is mapped with
  its own snapshot; a period before the first snapshot gets `as-reported-missing`; without
  `--as-reported` the output is unchanged from today's; `share` splits a shared account by the
  right snapshot's Apps.
- README: restated vs as reported, with an example.

**Out of scope.** As-reported trend and impact: they read `map`/`share` output and need no change.

**Done when.**
- The tests above pass, and every existing `map` and `share` reference case passes unchanged.
- `map --as-reported` on the fixtures maps the changed account's old periods to its old App
  (shown in the PR; release criterion 7).

**Verify.** `go run -tags=no_duckdb_arrow ./scripts/check`
