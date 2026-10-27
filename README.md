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

### `xops sense aws-accounts`

Lists an AWS Organization as `ou` and `account` JSON Lines records (OUs first, then accounts,
each sorted by `id`).

```bash
go run ./xops sense aws-accounts --mock
task run:xops:sense-aws-accounts
```

Flags: `--mock` (embedded demo org), `--profile`, `--region` (default `us-east-1`), `--lz` with
`--taxonomy` to stamp `lz` and verify the management account.

Record kinds:

| `kind` | Fields |
|---|---|
| `ou` | `id`, `name`, `parent_id`, `path`, `id_path`, `has_children`, plus `csp`, `org_id`, `management_account`, optional `lz` |
| `account` | `id`, `name`, `email`, `status`, `parent_id`, `ou_path`, `ou_id_path`, plus shared org fields |

Read-only IAM (management or delegated admin): `organizations:DescribeOrganization`,
`organizations:ListRoots`, `organizations:ListOrganizationalUnitsForParent`,
`organizations:ListAccountsForParent`.

### `finops enrich validate` and `finops enrich map`

The taxonomy is a YAML file (see `finops/enrich/testdata/demo-taxonomy.yaml`) with business units,
projects, apps, landing zones (`lzs`), optional `ou_links` and per-account overrides.

```bash
go run ./finops enrich validate --taxonomy finops/enrich/testdata/demo-taxonomy.yaml
go run ./finops enrich map --taxonomy finops/enrich/testdata/demo-taxonomy.yaml \
  --from-accounts finops/enrich/testdata/demo-accounts.jsonl
task run:finops:enrich-validate
task run:finops:enrich-map
```

`validate` prints `finding` records and exits 1 on severity `error`. `map` passes through `ou`
and `account` records, adds `allocation` rows, enriches optional `cost` records, and emits
mapping `finding` records. Pipe inventory from `xops sense aws-accounts --mock` instead of
`--from-accounts` when the org fixture is available.

### `finops analyze changes`

Compares two `map` (or `share`) outputs and emits `change` records for accounts and
allocations that were added, removed, moved between OUs, or remapped, then passes through
the current input unchanged.

```bash
go run ./finops enrich map --taxonomy finops/enrich/testdata/demo-taxonomy.yaml \
  --from-accounts finops/enrich/testdata/demo-accounts.jsonl \
  --from-cost finops/analyze/testdata/cost.jsonl > /tmp/map-before.jsonl
go run ./finops enrich map --taxonomy finops/analyze/testdata/demo-taxonomy-env-changed.yaml \
  --from-accounts finops/enrich/testdata/demo-accounts.jsonl \
  --from-cost finops/analyze/testdata/cost.jsonl > /tmp/map-now.jsonl
go run ./finops analyze changes --before /tmp/map-before.jsonl --from-map /tmp/map-now.jsonl
task run:finops:analyze-changes
```

Flags: `--before FILE` (required), `--from-map FILE` or pipe for the current map output.

### `xops report table|csv|excel`

Turns JSON Lines records into human-readable tables. One table per `kind`, in first-seen order;
`--kinds` limits which kinds are written. Columns follow field first-seen order (`kind` is
omitted). Inputs come from the pipe and/or `--from-*` flags (`--from-cost`, `--from-change`,
`--from-allocation`, `--from-finding`, `--from-accounts`).

```bash
go run ./xops report table --from-cost xops/report/testdata/mixed.jsonl
go run ./xops report csv --out /tmp/report --from-cost xops/report/testdata/mixed.jsonl
go run ./xops report excel --out /tmp/report.xlsx --from-cost xops/report/testdata/mixed.jsonl
task run:xops:report-table
task run:xops:report-csv
task run:xops:report-excel
```

| Subcommand | Output |
|---|---|
| `table` | Aligned text on stdout; numbers show two decimals. `--out FILE` appends the same text. |
| `csv` | `--out DIR` (required): `DIR/<kind>.csv`, UTF-8 RFC 4180. |
| `excel` | `--out FILE.xlsx` (required): one sheet per kind. |

Appending to an existing CSV, sheet or table file adds rows only (no second header). A record
field that is not already in the header stops the command with an error and leaves the file
unchanged; missing columns are left empty.

Developer setup (Linux first; see [docs/develop.md](docs/develop.md)), debugging and how to add a command.
