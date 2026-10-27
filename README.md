# OpenXOps

Standalone single-tenant command-line tools for cloud operations. Commands chain with `|`.

**Licence:** FSL-1.1-Apache-2.0 — free for your own use; contact the licensor for commercial use.

Two binaries:

- **`xops`** — shared commands (`xops sense aws-accounts`, `xops report excel`)
- **`finops`** — cost (first); `secops` and `cloudops` later

Stages: **Sense › Enrich › Analyze › Enforce › Report**. Records are flat JSON Lines; only Report
writes other formats.

## Quick start (`--mock`)

Needs [Go 1.22+](https://go.dev/dl/). From the repo root:

```bash
go run ./scripts/check
go run ./xops sense aws-accounts --mock > accounts.jsonl
go run ./finops sense aws-cost --mock > cost.jsonl
go run ./finops enrich map --taxonomy finops/enrich/testdata/demo-taxonomy.yaml \
  --from-accounts accounts.jsonl --from-cost cost.jsonl > map.jsonl
go run ./finops enrich share --taxonomy finops/enrich/testdata/demo-taxonomy.yaml \
  --from-cost map.jsonl > shared.jsonl
go run ./xops report excel --from-cost shared.jsonl --out finops.xlsx
```

## Documentation

| Audience | Start here |
|---|---|
| Users | [docs/user/getting-started.md](docs/user/getting-started.md) — full walkthrough, AWS setup, taxonomy, commands, records, findings |
| Developers | [docs/develop.md](docs/develop.md) — build, debug, add commands (Linux-first) |

Run `go run ./scripts/check` (or `task check`) before opening a pull request.
