# WI-0201 — History store and `--save`

## Outcome

Runs can be kept. `--save` writes a command's output to the history store as well as stdout, so
later runs (`impact`, `--as-reported`) can compare against what was reported before (D-034, D-039).

The inventory gets a memory too: `aws-accounts` saves its runs, and reads the last one back to
tell when each OU and account was first seen and last changed ([D-051](../DECISIONS.md#d-051),
[D-068](../DECISIONS.md#d-068)).

## Contracts

### Store (D-034)

```
data/history/<tenant>/<command>/<run timestamp>.jsonl     e.g. data/history/testtnt/map/20261027T101500Z.jsonl  (D-045)
```

- `<command>` is the command's last word (`map`, `share`, `trend`, `impact`). The timestamp is the
  run's start, UTC, `YYYYMMDDTHHMMSSZ`.
- Files are written once and never edited or deleted (kept forever). A name already taken is an
  error, never an overwrite.
- The file holds exactly what the command wrote to stdout.

### `--save`

- On `map`, `share`, `trend` and `impact` (D-039), and on `xops sense aws-accounts` (D-068). Other
  commands don't take it.
- With `--save`, the command writes stdout as usual and, only after it has succeeded, the same
  records to its history file. A failed run saves nothing.
- stderr names the file written (any log level).
- `--history-dir DIR` (default `data/history`) moves the store; tests always set it.

### `first_seen` and `last_changed` (D-068)

`xops sense aws-accounts … [--save] [--from-history FILE]`

- `--from-history` names an earlier saved `aws-accounts` file (any `ou` and `account` records
  of the same tenant; other kinds are ignored).
- Every `ou` and `account` record gets `first_seen` and `last_changed`, RFC 3339 UTC to the
  second (`2026-11-03T08:00:00Z`):

  | The node in the history file (same `id`) | `first_seen` | `last_changed` |
  |---|---|---|
  | absent | this run's start | this run's start |
  | present, same `name` and `parent_id` | the history's | the history's |
  | present, `name` or `parent_id` differs | the history's | this run's start |

  A history record with an empty `first_seen` counts as absent for it.
- Status, email and every other field don't count as a change.
- Without `--from-history`, both fields are `""`. They are optional strings in `record/schema`,
  so older files stay valid.
- A node in the history but not in this run is not written; nothing reports it in Release 2.
- The run's start comes from a clock tests can set.

### Package

`record/` already has the 5 non-test files `AGENTS.md` §5 allows, so the store is a new package,
`record/history/`: `Save(dir, command string, start time.Time, recs []record.Record) (path string, err error)`,
`List(dir, command string) ([]string, error)` (oldest first) and `Latest(dir, command string)`.
Plain functions; the clock is passed in.

## WO-0201-A — History store and `--save`

**Assignee:** open
**Depends on:** Release 1's Work Orders and WO-0101-B
**Packages:** `record/history/` (new), `finops/enrich/commands.go` (`--save` on `map` and `share`),
`docs/develop.md`, `README.md`

**Goal.** The store and `--save` on `map` and `share`. `trend` and `impact` add their own
`--save` with this package (WI-0203, WI-0204).

**In scope.**
- `record/history/` as specified, with tests: save and list in order; a name collision is an error
  and keeps the first file; `Latest` with no history returns no file and no error.
- `--save` and `--history-dir` on `map` and `share`; a failed run writes no history file (test).
- README and `docs/develop.md`: the store, `--save`, and that `data/` is local and gitignored.

**Out of scope.** Snapshots of the taxonomy (WI-0205). DuckDB (WI-0202).

**Done when.**
- The tests above pass.
- `map … --save` run twice leaves two files under `data/history/<tenant>/map/`, each equal to its run's
  stdout (shown in the PR).

**Verify.** `go run -tags=no_duckdb_arrow ./scripts/check`

## WO-0201-B — `first_seen` and `last_changed` on the inventory

**Assignee:** open
**Depends on:** WO-0201-A
**Packages:** `xops/sense/` (`aws-accounts`), `record/schema/` (the two optional fields on `ou` and
`account`), `xops/sense/testdata/` (new fixtures only), `docs/user/` (`commands.md`,
`records.md`), `README.md`, `Taskfile.yml`, `.vscode/launch.json`

**Goal.** `--save` and `--from-history` on `aws-accounts`, as *`first_seen` and `last_changed`*
says.

**In scope.**
- The flags, the table above, and the schema fields.
- Tests with a fixed clock and `--history-dir` in `t.TempDir()`:
  - a node absent from the history;
  - an unchanged node;
  - a renamed account;
  - an account moved to another OU;
  - a status change that doesn't count;
  - no `--from-history` giving `""`.
- `aws-accounts --mock --save` twice, the second with `--from-history` the first's file.

**Out of scope.** Reporting nodes that disappeared. `lz-tree` showing the two fields.

**Done when.**
- The tests above pass.
- Two mock runs, the second over a fixture where one account is renamed, keep every
  `first_seen` and move only that account's `last_changed` (shown in the PR; release criterion
  8).
- `go run -tags=no_duckdb_arrow ./scripts/check` all PASS.

**Verify.** `go run -tags=no_duckdb_arrow ./scripts/check`
