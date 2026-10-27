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
- **The pipe.** When stdin is not a terminal, its records are read too and sorted by `kind`.
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
- **CI** (`.github/workflows/check.yml`): `go run ./scripts/check` on `windows-latest` and
  `ubuntu-latest`, for every push and PR.
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
- `task check` and `go run ./scripts/check` pass on Windows and Linux (CI green on both).
- `task build` produces `xops` and `finops`; each prints its version.
- The `record/` tests above pass.
- `docs/develop.md` lets a new developer go from a clean Windows machine to a breakpoint in a
  `record/` test in VS Code, and the PR description says it was followed on Windows.
- The dev container opens in VS Code and `task check` passes inside it (shown in the PR).

**Verify.** `go run ./scripts/check`
