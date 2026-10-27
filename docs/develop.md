# Developer setup

Windows is the main path. Linux and macOS follow the same commands once the tools are on `PATH`.
Release 1 builds with plain `go build` (no CGO, no DuckDB).

## Install on Windows

1. **Go 1.22 or newer.** Download the MSI from https://go.dev/dl/ and run it. Open a **new**
   PowerShell window so `go version` works.
2. **Git.** https://git-scm.com/download/win — the installer is enough. Use “Git from the command
   line and also from 3rd-party software”.
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

`task check` is `go run ./scripts/check`: build, `go vet`, `gofmt -l`, `go test ./...`.

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

Every command can start from `--from-<kind> FILE` (and later `--mock`) with no pipe and no cloud.
Today the only command is `version`:

```powershell
go run ./xops version
go run ./finops version --log-level debug
task run:xops:version
task run:finops:version
```

Later commands take files:

```powershell
go run ./finops enrich map --from-cost testdata\cost.jsonl --from-accounts testdata\accounts.jsonl --taxonomy testdata\taxonomy.yaml
```

`--from-<kind> -` reads stdin. When stdin is not a terminal, the command also reads the pipe;
the same kind from both the pipe and a flag is an error. Logs go to stderr; records go to stdout.

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

Run and Debug → **xops version** or **finops version**. Later Work Items add one launch entry
per command, pointed at that package’s `testdata/`.

## Add a command

1. Put the command in `<binary>/<stage>/` (for example `finops/enrich/`). Export
   `Commands() []*cobra.Command`. Keep at most five non-test `.go` files in the package.
2. Stage logic is a plain function: records in, records out, no I/O. The command file only
   reads flags and files and calls that function.
3. In the binary’s `main.go`, add the stage’s commands to the root (`record.Root` already
   provides `--log-level` and `version`).
4. Add `run:<binary>:<command>` to `Taskfile.yml` and a matching entry in `.vscode/launch.json`,
   fed by the package’s `testdata/` or `--mock`.
5. Document flags on the cobra command (they become `--help`). Update the README if the change
   is user-visible.

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

Native Windows stays the main path. `.devcontainer/` is a ready Linux environment (Go, Task,
AWS CLI v2, Delve) for VS Code **Reopen in Container** and GitHub Codespaces.

1. Install Docker Desktop and the VS Code Dev Containers extension.
2. Open the folder → Command Palette → **Dev Containers: Reopen in Container**.
3. In the container terminal: `task check`.

## Verify

```powershell
task check
go run ./scripts/check
```

CI runs the same command on `windows-latest` and `ubuntu-latest` for every push and pull
request (`.github/workflows/check.yml`).
