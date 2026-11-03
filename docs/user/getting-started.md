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

These commands use embedded fixtures only (no AWS credentials needed). Every command runs for one
tenant, named with `--tenant`; the demo taxonomy belongs to tenant `testtnt`, and every record
written carries `"tenant":"testtnt"`.

### 1. Step-by-step FinOps pipeline (with intermediate files)

Writing intermediate `.jsonl` files lets you inspect records between stages:

```bash
# Discover accounts and validate demo taxonomy
go run ./xops sense aws-accounts --tenant testtnt --mock > accounts.jsonl
go run ./finops enrich validate --tenant testtnt --taxonomy finops/enrich/testdata/demo-taxonomy.yaml

# Pull mock Cost Explorer data
go run ./finops sense aws-cost --tenant testtnt --mock > cost.jsonl

# Map account ownership & cost URNs
go run ./finops enrich map --tenant testtnt \
  --taxonomy finops/enrich/testdata/demo-taxonomy.yaml \
  --from-accounts accounts.jsonl \
  --from-cost cost.jsonl > map.jsonl

# Split shared account costs by app tags
go run ./finops enrich share --tenant testtnt \
  --taxonomy finops/enrich/testdata/demo-taxonomy.yaml \
  --from-cost map.jsonl > shared.jsonl

# Generate multi-sheet Excel report
go run ./xops report excel --tenant testtnt --from-cost shared.jsonl --out finops.xlsx
```

Open `finops.xlsx`: each sheet is one record kind from the shared cost file (`cost`, `allocation`, `finding`).

To inspect records directly in the terminal without opening Excel:
```bash
go run ./xops report table --tenant testtnt --from-cost shared.jsonl
```

Optional: compare two map runs over time with `finops analyze changes` (see [commands.md](commands.md)).

### 2. Cloud hierarchy & scaffolding pipeline

See your cloud hierarchy as URNs before cost is mapped (D-054). `lz-tree` turns accounts into `node`
records per LZ, OU, and account. You can chain commands directly using pipes (`|`):

```bash
# View human-readable nested YAML tree
go run ./xops sense aws-accounts --tenant testtnt --mock | \
  go run ./xops enrich lz-tree --tenant testtnt --taxonomy xops/enrich/testdata/demo-taxonomy.yaml | \
  go run ./xops report tree --tenant testtnt --out tree.yaml

# Or scaffold a taxonomy-compatible landing zone connection block:
go run ./xops sense aws-accounts --tenant testtnt --mock | \
  go run ./xops enrich lz-tree --tenant testtnt --taxonomy xops/enrich/testdata/demo-taxonomy.yaml | \
  go run ./xops report tree --tenant testtnt --format taxonomy --out tree-taxonomy.yaml
```

## Real Organization

### 1. Configure AWS credentials
Configure AWS credentials with read-only access described in [aws-setup.md](aws-setup.md).

### 2. Scaffold your initial taxonomy (`lz_conns`)
To bootstrap a nested taxonomy matching your AWS organization layout:
1. Discover your organization's accounts and OUs:
   ```bash
   go run ./xops sense aws-accounts --tenant <tenant> > accounts.jsonl
   ```
2. Create a minimal bootstrap taxonomy containing your tenant ID and one landing zone:
   ```yaml
   tenant: { code: <tenant>, name: <tenant name> }
   lzs:
     - code: <lz-code>
       name: <lz-name>
       csp: aws
       currency: USD
       management_account: "<management-account-id>"
   ```
3. Enrich the discovered inventory into cloud hierarchy nodes:
   ```bash
   go run ./xops enrich lz-tree --tenant <tenant> --taxonomy bootstrap.yaml \
     --from-accounts accounts.jsonl > tree.jsonl
   ```
4. Scaffold the `lz_conns` block with leaf OUs and accounts:
   ```bash
   go run ./xops report tree --tenant <tenant> --from-node tree.jsonl --format taxonomy --out lz-conns.yaml
   ```
5. Merge `lz-conns.yaml` with your Business Value (`business_units`) and Project (`projects`) trees
   into your final `taxonomy.yaml` (see [taxonomy.md](taxonomy.md)), then add `ref_urn_apps` and
   `ref_urn_env` to the OUs and accounts that should map to an App or an Environment. The scaffold
   is a taxonomy the loader reads, so `finops enrich validate` checks it as it stands.

### 3. Validate your taxonomy
Before running cost mapping, validate your taxonomy structure and rules:
```bash
go run ./finops enrich validate --tenant <tenant> --taxonomy taxonomy.yaml
```
Fix any `error` findings before proceeding.

### 4. Run the FinOps pipeline
Run the live pipeline using your verified taxonomy:

```bash
# 1. Discover accounts
go run ./xops sense aws-accounts --tenant <tenant> > accounts.jsonl

# 2. Sense live AWS Cost Explorer
go run ./finops sense aws-cost --tenant <tenant> --taxonomy taxonomy.yaml > cost.jsonl

# 3. Enrich ownership and URNs
go run ./finops enrich map --tenant <tenant> --taxonomy taxonomy.yaml \
  --from-accounts accounts.jsonl --from-cost cost.jsonl > map.jsonl

# 4. Split shared account costs
go run ./finops enrich share --tenant <tenant> --taxonomy taxonomy.yaml --from-cost map.jsonl > shared.jsonl

# 5. Export multi-sheet Excel report
go run ./xops report excel --tenant <tenant> --from-cost shared.jsonl --out finops.xlsx

# Optional: preview formatted tables directly in the terminal
go run ./xops report table --tenant <tenant> --from-cost shared.jsonl
```

The second `aws-cost` run over the same closed months makes no new Cost Explorer calls
(release criterion 5); stderr reports call count and cost.

> [!TIP]
> After building binaries with `go build -o bin/xops ./xops` and `go build -o bin/finops ./finops` (or `task build`),
> you can invoke `./bin/xops` and `./bin/finops` directly (`.\bin\xops.exe` on Windows) to skip Go compilation overhead.

## More detail

- [commands.md](commands.md) — every flag and example
- [records.md](records.md) and [findings.md](findings.md) — data shapes and codes
- [develop.md](../develop.md) — contributor setup (Linux-first)
