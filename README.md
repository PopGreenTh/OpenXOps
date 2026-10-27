# OpenXOps

Standalone single-tenant command-line tools for cloud operations. Commands chain with `|`.

**Licence:** FSL-1.1-Apache-2.0: free for your own use; contact the licensor for commercial use.

Two binaries:

- `xops` — commands shared by every domain (`xops sense aws-accounts`, `xops report excel`)
- `finops` — cost (first); `secops` and `cloudops` later

Every command is `<binary> <stage> <thing>` and belongs to one of five stages:

**Sense › Enrich › Analyze › Enforce › Report**

| Stage | Job |
|---|---|
| Sense | Read a cloud (or `--mock`) and emit raw records |
| Enrich | Add taxonomy context: who owns the cost, how shared cost splits |
| Analyze | Compare and grade over time |
| Enforce | Flag what breaks a rule, with the action to take |
| Report | Turn records into tables people read |

Records are flat JSON Lines. Only Report writes other formats.

## Build and run

Needs [Go 1.22+](https://go.dev/dl/) and, optionally, [Task](https://taskfile.dev/).

```bash
task build          # binaries in bin/
task check          # Verify: build, vet, gofmt, test
go run ./xops version
go run ./finops version
```

Or without Task:

```bash
go run ./scripts/check
go build -o bin/xops ./xops
go build -o bin/finops ./finops
```

On Windows the binaries are `bin\xops.exe` and `bin\finops.exe`.

Developer setup (Windows first, then Linux and macOS), debugging and how to add a command: [docs/develop.md](docs/develop.md).
