# Getting started

This walkthrough runs the Release 1 FinOps pipeline on mock data, then describes the same
steps against your AWS Organization. You need [Go 1.22+](https://go.dev/dl/) on your path.

## Build

From a clean clone:

```bash
go run ./scripts/check
go build -o bin/xops ./xops
go build -o bin/finops ./finops
```

Or with [Task](https://taskfile.dev/): `task check` and `task build`.

## Mock pipeline

These commands use embedded fixtures only (no AWS credentials). They write intermediate files in
the current directory and finish with `finops.xlsx`.

```bash
go run ./xops sense aws-accounts --mock > accounts.jsonl
go run ./finops sense aws-cost --mock > cost.jsonl
go run ./finops enrich map \
  --taxonomy finops/enrich/testdata/demo-taxonomy.yaml \
  --from-accounts accounts.jsonl \
  --from-cost cost.jsonl > map.jsonl
go run ./finops enrich share \
  --taxonomy finops/enrich/testdata/demo-taxonomy.yaml \
  --from-cost map.jsonl > shared.jsonl
go run ./xops report excel --from-cost shared.jsonl --out finops.xlsx
```

Open `finops.xlsx`: each sheet is one record kind from the shared cost file.

Optional: compare two map runs with `finops analyze changes` (see [commands.md](commands.md)).

## Real Organization

1. Configure AWS credentials with read-only access described in [aws-setup.md](aws-setup.md).
2. Prepare a taxonomy YAML for your organization (see [taxonomy.md](taxonomy.md)).
3. Replace `--mock` with live calls and add `--lz` / `--taxonomy` where your landing zone
   requires it:

```bash
go run ./xops sense aws-accounts > accounts.jsonl
go run ./finops sense aws-cost --lz LZ-MAIN --taxonomy taxonomy.yaml > cost.jsonl
go run ./finops enrich map --taxonomy taxonomy.yaml \
  --from-accounts accounts.jsonl --from-cost cost.jsonl > map.jsonl
go run ./finops enrich share --taxonomy taxonomy.yaml --from-cost map.jsonl > shared.jsonl
go run ./xops report excel --from-cost shared.jsonl --out finops.xlsx
```

The second `aws-cost` run over the same closed months should make no new Cost Explorer calls
(release criterion 5); stderr reports call count and cost.

## More detail

- [commands.md](commands.md) — every flag and example
- [records.md](records.md) and [findings.md](findings.md) — data shapes and codes
- [develop.md](../develop.md) — contributor setup (Linux-first)
