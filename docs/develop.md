# Developer setup

Linux is the main path (D-025). Windows and macOS follow the same commands once the tools are on
`PATH`. Windows CI is optional (D-026). Release 1 builds with plain `go build` (no CGO, no DuckDB).

## Install on Windows

1. **Go 1.22 or newer.** Download the MSI from https://go.dev/dl/ and run it. Open a **new**
   PowerShell window so `go version` works.
2. **Git.** https://git-scm.com/download/win — the installer is enough. Use “Git from the command
   line and also from 3rd-party software”. The repo’s `.gitattributes` keeps text files as LF,
   including on Windows, so `gofmt` and Git do not fight over line endings.
3. **Task.** In the new PowerShell window:

   ```powershell
   winget install Task.Task
   ```

   Or `go install github.com/go-task/task/v3/cmd/task@v3.40.1` (a version that builds with Go
   1.22) and add `%USERPROFILE%\go\bin` to your user `PATH`. Confirm with `task --version` in a
   new window.
4. **VS Code.** https://code.visualstudio.com/download
5. **Go extension.** Open VS Code → Extensions → install **Go** (`golang.go`). When it offers to
   install `gopls` and Delve (`dlv`), accept. That is the only debugger setup; nothing else is
   required on Windows for Release 1.

Clone and Verify:

```powershell
git clone https://github.com/PopGreenTh/OpenXOps.git
cd OpenXOps
task check
```

`task check` is `go run ./scripts/check`: build, `go vet`, `gofmt -l`, `go test ./...`, and a
`docs` step that runs `scripts/doccheck`.

## Linux and macOS

- **Go:** https://go.dev/dl/ or your package manager (`brew install go`).
- **Git:** already present on macOS; `sudo apt install git` on Debian/Ubuntu.
- **Task:** https://taskfile.dev/installation/ — `brew install go-task` on macOS, or the
  install script. `go install github.com/go-task/task/v3/cmd/task@v3.40.1` also works (pin; newer
  Task releases may need a newer Go than 1.22).
- **VS Code + Go extension + Delve:** same as Windows.

Then `git clone` and `task check` as above.

## Build

```powershell
task build
.\bin\xops.exe version
.\bin\finops.exe version
```

On Linux and macOS the binaries have no `.exe` suffix (`./bin/xops version`).

`go build -o bin/xops ./xops` and `go build -o bin/finops ./finops` do the same thing. Version
output comes from the Go build’s VCS information (commit and build date). `task clean` removes
`bin/`.

## Run a command alone from files

Every command can start from `--from-<kind> FILE` and/or `--mock` with no pipe and no cloud:

```bash
go run ./xops version
go run ./finops version --log-level debug
go run ./xops sense aws-accounts --tenant testtnt --mock
go run ./finops enrich map --tenant testtnt --taxonomy finops/enrich/testdata/demo-taxonomy.yaml \
  --from-accounts finops/enrich/testdata/demo-accounts.jsonl
task run:xops:sense-aws-accounts
```

**Pipe rule (D-030).** A command reads stdin only when it is given **no** `--from-<kind>` flag,
or when a flag's `FILE` is `-`. File-only flags never read stdin, even if stdin is an open pipe
(that would hang under cron). Stdin records keep their input order, as file records do (D-043).
The same kind from stdin and
from a file is an error that names both flags.

Mix pipe and files explicitly, for example:

```powershell
go run ./finops sense aws-cost --tenant testtnt --mock | go run ./finops enrich map --tenant testtnt --taxonomy finops/enrich/testdata/demo-taxonomy.yaml --from-accounts finops/enrich/testdata/demo-accounts.jsonl --from-cost -
```

`--from-<kind> -` reads stdin; `--from-accounts` reads only `ou` and `account` from its file.
Logs go to stderr; records go to stdout.

## Debug in VS Code

Open the repository folder (`File → Open Folder`). The repo’s `.vscode/launch.json` already has
one entry per command, plus a test entry.

### Breakpoint in a `record/` test

1. Open `record/read.go` and click in the gutter on the `CheckFlat` call inside `parseRecord`
   (or open `record/record_test.go` and break in `TestNestedValueRejected`).
2. Run and Debug (`Ctrl+Shift+D`) → **record: tests** → green play (`F5`).
3. Delve stops on the breakpoint. Step with `F10` / `F11`. The test uses
   `record/testdata/nested.jsonl`; no AWS credentials and no extra Windows setup.

You can also open `record/record_test.go` and use the Go extension’s **debug test** CodeLens
above a single test function.

### Debug a command

Run and Debug → pick any command entry (for example **finops enrich map**). Each launch uses
`--mock` or that package’s `testdata/`.

## Add a command

1. Put the command in `<binary>/<stage>/` (for example `finops/enrich/`). Export
   `Commands() []*cobra.Command`. Keep at most five non-test `.go` files in the package.
2. Stage logic is a plain function: records in, records out, no I/O. The command file only
   reads flags and files and calls that function.
3. In the binary’s `main.go`, add the stage’s commands to the root (`record.Root` already
   provides `--log-level` and `version`).
4. Add `run:<binary>:<command>` to `Taskfile.yml` and a matching entry in `.vscode/launch.json`,
   fed by the package’s `testdata/` or `--mock`.
5. Document flags on the cobra command (they become `--help`). Update
   [docs/user/commands.md](user/commands.md) (and the README quick start if the change is
   user-visible).

## How the pieces fit

OpenXOps splits **binaries** (`xops/`, `finops/`), **stages** (Sense / Enrich / Analyze / Report
under each binary), **records** (`record/`, flat JSON Lines with a `kind` field), and **cloud
clients** (`aws/`, later other providers). Commands wire flags and files to plain functions; only
the command layer does I/O. The full picture is in
[planning/ARCHITECTURE.md](../planning/ARCHITECTURE.md).

## Add a record kind

1. Add the kind’s field table in `record/schema/schema.go` (`ByKind` and a `*Schema` variable).
   Keep fields flat (scalars only) per [D-006](../planning/DECISIONS.md#d-006).
2. Add the kind to the `want` slice in `TestEveryContractKindHasSchema` in
   `record/schema/schema_test.go`.
3. Add tests in `record/schema/schema_test.go` or `record/record_test.go` if validation behaviour
   is new.
4. Document the kind in [docs/user/records.md](user/records.md): one `## \`kind\`` section with
   columns field, type, required, meaning — every schema field must appear.
5. If a command emits a new finding code (a string literal passed to `record.Finding` or to a
   helper that forwards to it), add a row to [docs/user/findings.md](user/findings.md).
6. Run `go run ./scripts/check` (must print `docs PASS`).

## Run Release 1 checks (criteria 1–5)

Criteria 6–7 need AWS access and user docs; for day-to-day development, run criteria **1–5** from
[planning/DD-01-r1-finops-aws/DELIVERABLE.md](../planning/DD-01-r1-finops-aws/DELIVERABLE.md#release-criteria):

1. `go run ./scripts/check` — Linux CI equivalent (build, vet, gofmt, tests, docs).
2. Mock pipeline from [docs/user/getting-started.md](user/getting-started.md) (or the README
   quick start) through `finops.xlsx`.
3. Reference cases: `go test ./finops/enrich/...` (map and share cases under `testdata/cases/`).
4. Cost conservation after `share` — `go test ./finops/enrich/...` (share cases and
   `CheckConservation` in tests).
5. Second `aws-cost --mock` run reports zero new Cost Explorer calls on stderr.

## Add a reference case

Reference cases live in the `testdata/cases/` of the **package that reads them** (not at the
repo root). They are YAML: a name, why the case exists, inputs (`taxonomy`, `records`) and
`expect`. Tests read them in place and must not edit them.

1. Copy a neighbour under `testdata/cases/<command>/`.
2. Change only the new file’s inputs and expected output.
3. Run the package tests. If a case looks wrong, stop and report it; do not “fix” the case
   to make a test pass.

Examples already in the tree: `finops/enrich/testdata/cases/map/` and `…/share/`.

## Log levels

`--log-level debug|info|warn|error` on every command, default `warn`. Go `log/slog` text to
**stderr**. At `debug` a command logs every input file, its flags, and every cloud or cache
call. stdout is records only (except `version`). Exit 0 on success, including findings; 1 on
error. Errors name the file (or `stdin`) and the line number.

## Dev container (optional)

Native Windows stays the main path. `.devcontainer/` is a ready Linux environment (Go 1.22,
Task v3.40.1, AWS CLI v2, Delve v1.25.2) for VS Code **Reopen in Container** and GitHub
Codespaces. Delve is pinned: newer releases need a newer Go than 1.22.

1. Install Docker Desktop and the VS Code Dev Containers extension.
2. Open the folder → Command Palette → **Dev Containers: Reopen in Container**.
3. In the container terminal: `task check`.

## Verify

```powershell
task check
go run ./scripts/check
```

CI runs the same command on `ubuntu-latest` for every push and pull request
(`.github/workflows/check.yml`). A `windows-latest` job also runs, marked optional
(`continue-on-error`): a Windows failure is a warning and does not block the PR (D-026).
