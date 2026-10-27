# WI-0201 — History store and `--save`

## Outcome

Runs can be kept. `--save` writes a command's output to the history store as well as stdout, so
later runs (`impact`, `--as-reported`) can compare against what was reported before (D-034, D-039).

## Contracts

### Store (D-034)

```
data/history/<command>/<run timestamp>.jsonl     e.g. data/history/map/20261027T101500Z.jsonl
```

- `<command>` is the command's last word (`map`, `share`, `trend`, `impact`). The timestamp is the
  run's start, UTC, `YYYYMMDDTHHMMSSZ`.
- Files are written once and never edited or deleted (kept forever). A name already taken is an
  error, never an overwrite.
- The file holds exactly what the command wrote to stdout.

### `--save`

- On `map`, `share`, `trend` and `impact` (D-039). Other commands don't take it.
- With `--save`, the command writes stdout as usual and, only after it has succeeded, the same
  records to its history file. A failed run saves nothing.
- stderr names the file written (any log level).
- `--history-dir DIR` (default `data/history`) moves the store; tests always set it.

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
- `map … --save` run twice leaves two files under `data/history/map/`, each equal to its run's
  stdout (shown in the PR).

**Verify.** `go run -tags=no_duckdb_arrow ./scripts/check`
