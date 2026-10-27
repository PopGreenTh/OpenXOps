# WI-0101 — Skeleton, records and developer setup

## Outcome

Two binaries, `xops` and `finops`, that build on Windows, Linux and macOS with plain `go build`;
the shared record code every command uses; and a developer setup in which any command can be
run, debugged and tested on its own (D-013, D-019, D-020). No cloud command yet.

## Contracts

### Layout (D-012)

```
xops/            package main: the xops binary; xops/<stage>/ for its commands
finops/          package main: the finops binary; finops/<stage>/ for its commands
record/          JSON Lines reader and writer, input assembly, findings, common flags
aws/             AWS config, retry wrapper, clients, mock data (from WI-0102)
scripts/check/   the Verify command
docs/develop.md  developer guide
.vscode/  .devcontainer/  .github/workflows/  Taskfile.yml  README.md  .gitignore
```

A package has at most 5 non-test `.go` files and keeps its fixtures in its own `testdata/`.
Each stage package exports one `Commands() []*cobra.Command` (or the equivalent for the CLI
library chosen); the binary's `main.go` only adds them. Command code reads flags and files and
calls plain functions that take and return records (D-013).

### Records (D-006)

- UTF-8 JSON Lines: one JSON object per line, `\n` line ends, no blank lines.
- **Flat**: values are strings, numbers, booleans or `null`; never objects or arrays.
- Every record has `kind`. Field names are `snake_case`; amounts are JSON numbers; periods are
  `YYYY-MM` or `YYYY-MM-DD`; timestamps RFC 3339 UTC.
- A command passes on every record whose `kind` it doesn't handle, unchanged and in input
  order, after its own output.

`finding` records report problems without failing the command:

`{"kind":"finding","severity":"warning","code":"lz-unknown","subject":"o-ab12cd34","message":"…","command":"finops enrich map"}`

`severity` is `info`, `warning` or `error`. Reference cases compare `code` and `subject` only.

### Inputs (D-006)

- `--from-<kind> FILE` reads a record file; `FILE` may be `-` for stdin. Each command lists the
  kinds it reads, and has one such flag per kind (`--from-accounts` reads `ou` and `account`,
  `--from-cost` reads `cost`; each Work Item names its own).
- **The pipe** (D-030, replaces "when stdin is not a terminal"). A command reads stdin only when
  it is given **no** `--from-<kind>` flag, or when a flag's `FILE` is `-`. Records from stdin
  are sorted by `kind`. A command whose inputs all come from files never touches stdin, so it
  can't hang under cron or CI. To mix, say so: `aws-cost | map --from-accounts a.jsonl --from-cost -`.
  A command that reads no records (Sense) never reads stdin.
- The same kind from both the pipe and a flag is an error that names both.
- `--from-history FILE` is the command's own earlier output, for commands that use it.
- A bad line stops the command with an error naming the file (or `stdin`) and the line number.

### Common flags and behaviour

- `--log-level debug|info|warn|error`, default `warn`, Go `log/slog` text to **stderr**. At
  `debug` a command logs every input file, its flags and every cloud or cache call.
- stdout carries records only. Exit code 0 on success (findings included), 1 on error.
- `xops version` and `finops version` print the version, commit and build date.

### Developer setup (D-013, D-019, D-020)

- **Taskfile.yml** (Task), working in PowerShell and bash: `build` (both binaries into `bin/`),
  `check` (= Verify), `test`, `run:<binary>:<command>` for every command on its fixtures or
  `--mock` (later Work Items add theirs), `clean`.
- **`.vscode/`**: `launch.json` with one debug entry per command (later Work Items add theirs),
  `settings.json`, `extensions.json` recommending the Go extension. Debugging with Delve works on
  Windows with no extra setup beyond `docs/develop.md`.
- **`.devcontainer/`**: Go, Task, AWS CLI v2, Delve; `task check` passes inside it.
- **`docs/develop.md`**: install on Windows (first), Linux, macOS; build; run a command alone
  from files; debug in VS Code; add a command; add a reference case; the log levels.
- **CI** (`.github/workflows/check.yml`): `go run ./scripts/check` on `ubuntu-latest`
  (required) and on `windows-latest` (optional: `continue-on-error: true`, job name ending
  `optional`), for every push and PR (D-026).
- **`scripts/check`**: build, `go vet`, `gofmt -l` (fails on any file listed), `go test ./...`;
  one PASS/FAIL line per step; non-zero exit on any failure.

## WO-0101-A — Binaries, `record/`, check, CI and developer setup

**Assignee:** open
**Depends on:** —
**Packages:** the whole repository except `planning/`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`

**Goal.** Everything above, with `version` as the only command of each binary.

**In scope.**
- `go.mod` (module `github.com/PopGreenTh/OpenXOps`), both binaries with `version`.
- `record/`: reader (file and line tracking), flat writer, flatness check, input assembly (pipe
  plus `--from-<kind>` flags, the conflict error), finding helper, `--log-level` setup.
- `scripts/check`, CI, Taskfile, `.vscode/`, `.devcontainer/`, `docs/develop.md`, README (what
  OpenXOps is, the five stages, build and run, and a licence line: "FSL-1.1-Apache-2.0: free
  for your own use; contact the licensor for commercial use", D-024), `.gitignore` (`bin/`,
  `data/`).
- Unit tests for `record/`: a nested value rejected; a bad line reported with file and line;
  unknown kinds pass through in order; pipe plus flag of the same kind is an error; records
  sorted by kind from a mixed pipe.

**Out of scope.** Any cloud or stage command (WI-0102 onward). DuckDB (D-022). The licence file
(the repo owner adds it).

**Done when.**
- `task check` and `go run ./scripts/check` pass on Linux (CI's Linux job green); the Windows
  job is marked optional (D-026).
- `task build` produces `xops` and `finops`; each prints its version.
- The `record/` tests above pass.
- `docs/develop.md` lets a new developer go from a clean Linux machine to a breakpoint in a
  `record/` test in VS Code or Cursor (D-025).

**Verify.** `go run ./scripts/check`

## WO-0101-B — Pipe rule and review follow-ups

**Assignee:** open
**Depends on:** WO-0103-A, WO-0105-A, WO-0107-A (dispatched after they merge, D-031)
**Packages:** `record/`, `aws/orgapi.go`, `xops/sense/`, `xops/report/`, `finops/enrich/resolve.go`,
`docs/develop.md`, `README.md`, and any command package whose tests rely on the old pipe rule

**Goal.** The D-030 pipe rule in `record/`, plus the non-blocking notes left from the WO-0102-A,
WO-0104-A and WO-0106-A reviews.

**In scope.**
- `record/`: input assembly follows D-030 (*Inputs* above). Tests: no flag reads stdin; a
  `--from-<kind>` file and no `-` never reads stdin, even when stdin is an open pipe that never
  closes; `--from-<kind> -` reads stdin; the same kind from stdin and a file is still an error.
- `xops/sense`: `aws-accounts` writes through `cmd.OutOrStdout()`; the discovery-error test
  checks it gets the injected error (`errors.Is`). `aws/orgapi.go`: drop the unused
  `maxConcurrent` parameter (or use it, bounded at 5).
- `xops/report`: no trailing spaces on table rows; Excel append finds the next row from the
  sheet's real last row, not `len(GetRows)`, with a test whose last appended record is all
  empty.
- `finops/enrich/resolve.go`: one `lz-unknown` finding per organization; the
  `project-tag-mismatch` message names the tag and the account's Project or Projects.
- `docs/develop.md` and `README.md`: the pipe rule, with the mixing example.

**Out of scope.** Any new command or record field.

**Done when.**
- With stdin an open pipe that never closes (`sleep 60 | finops enrich map --taxonomy … --from-accounts …`),
  the command finishes as fast as with stdin closed; a `record/` test proves it.
- Every bullet above has a test, and every reference case still passes unchanged.

**Verify.** `go run ./scripts/check`

