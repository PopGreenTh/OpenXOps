# OpenXOps

Standalone multi-tenant command-line tools for cloud operations. Commands chain with `|`. Every
command runs for one tenant, named with `--tenant`, and every record says which tenant it belongs
to.

**Licence:** FSL-1.1-Apache-2.0 — free for your own use; contact the licensor for commercial use.

Two binaries:

- **`xops`** — shared commands (`xops sense aws-accounts`, which lists OUs and accounts with
  each account's join time, and `xops report excel`)
- **`finops`** — cost (first); `secops` and `cloudops` later

Stages: **Sense › Enrich › Analyze › Enforce › Report**. Records are flat JSON Lines; only Report
writes other formats.

## Quick start (`--mock`)

Needs [Go 1.22+](https://go.dev/dl/). From the repo root:

```bash
go run ./scripts/check
go run ./xops sense aws-accounts --tenant testtnt --mock > accounts.jsonl
go run ./finops sense aws-cost --tenant testtnt --mock > cost.jsonl
go run ./finops enrich map --tenant testtnt --taxonomy finops/enrich/testdata/demo-taxonomy.yaml \
  --from-accounts accounts.jsonl --from-cost cost.jsonl > map.jsonl
go run ./finops enrich share --tenant testtnt --taxonomy finops/enrich/testdata/demo-taxonomy.yaml \
  --from-cost map.jsonl > shared.jsonl
go run ./xops report excel --tenant testtnt --from-cost shared.jsonl --out finops.xlsx
```

Optionally, see the cloud hierarchy as URNs: `xops enrich lz-tree` turns `accounts.jsonl` into
`node` records and `xops report tree` writes them as nested YAML:

```bash
go run ./xops enrich lz-tree --tenant testtnt --taxonomy xops/enrich/testdata/demo-taxonomy.yaml \
  --from-accounts accounts.jsonl > tree.jsonl
go run ./xops report tree --tenant testtnt --from-node tree.jsonl --out tree.yaml
```

## Documentation

| Audience | Start here |
|---|---|
| Users | [docs/user/getting-started.md](docs/user/getting-started.md) — full walkthrough, AWS setup, taxonomy, commands, records, findings |
| Developers | [docs/develop.md](docs/develop.md) — build, debug, add commands (Linux-first) |

Run `go run ./scripts/check` (or `task check`) before opening a pull request.
