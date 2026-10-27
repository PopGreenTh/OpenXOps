# Commands

Every command is `<binary> <stage> <thing>`. Stages: **Sense › Enrich › Analyze › Enforce › Report**.
All commands accept `--log-level` on stderr; records go to stdout (except `report table` without
`--out`).

## `xops version`

Print version, commit and build date.

**Example:** `go run ./xops version`

## `finops version`

Same for the FinOps binary.

**Example:** `go run ./finops version`

## `xops sense aws-accounts`

List an AWS Organization as `ou` and `account` records (OUs first, then accounts, each sorted by
`id`).

**Flags:** `--mock`, `--profile`, `--region` (default `us-east-1`), `--lz` with `--taxonomy`.

**Reads:** nothing (live AWS or mock).

**Writes:** `ou`, `account`; may emit `finding` (`ou-path-ambiguous`).

**Example:** `go run ./xops sense aws-accounts --mock > accounts.jsonl`

## `finops sense aws-cost`

Two-level Cost Explorer read: `project_tag_cost` then `cost` records.

**Flags:** `--mock`, `--profile`, `--region`, `--granularity monthly|daily`, `--metric`,
`--from` / `--to`, `--app-tag`, `--project-tag`, `--lz` with `--taxonomy`, `--refresh`.

**Reads:** nothing (live AWS or mock).

**Writes:** `project_tag_cost`, `cost`; may emit `finding` (`cost-history-short`).

**Example:** `go run ./finops sense aws-cost --mock > cost.jsonl`

## `finops enrich validate`

Check a taxonomy YAML file and print `finding` records.

**Flags:** `--taxonomy FILE` (required).

**Reads:** taxonomy file.

**Writes:** `finding` (taxonomy codes); exits 1 on severity `error`.

**Example:** `go run ./finops enrich validate --taxonomy finops/enrich/testdata/demo-taxonomy.yaml`

## `finops enrich map`

Apply taxonomy to accounts and optional cost; pass through inventory; write `allocation` rows.

**Flags:** `--taxonomy FILE` (required), `--from-accounts`, `--from-cost`, `--from-finding`, …

**Reads:** `ou`, `account`, optional `cost`, optional upstream `finding`.

**Writes:** pass-through inventory, `allocation`, enriched optional `cost`, mapping `finding`
records.

**Example:**

```bash
go run ./finops enrich map --taxonomy finops/enrich/testdata/demo-taxonomy.yaml \
  --from-accounts finops/enrich/testdata/demo-accounts.jsonl \
  --from-cost finops/enrich/testdata/demo-cost.jsonl
```

## `finops enrich share`

Split shared-account `cost` by App tag using `allocation` rows from `map`.

**Flags:** `--taxonomy FILE` (required), `--from-cost`, `--from-allocation`, …

**Reads:** `cost`, `allocation`, pass-through kinds from `map`.

**Writes:** updated `cost` rows with `app` / `project` / `environment`; share `finding` records.

**Example:**

```bash
go run ./finops enrich share --taxonomy finops/enrich/testdata/demo-taxonomy.yaml \
  --from-cost map.jsonl
```

## `finops analyze changes`

Compare a previous map output to the current one; emit `change` records, then pass through input.

**Flags:** `--before FILE` (required), `--from-map FILE` or stdin.

**Reads:** prior and current map (or share) output.

**Writes:** `change`, then pass-through records from current input.

**Example:** `go run ./finops analyze changes --before before.jsonl --from-map now.jsonl`

## `xops report table`

Print aligned tables per record kind (stdout or append with `--out`).

**Flags:** `--from-cost`, `--from-change`, `--from-allocation`, `--from-finding`,
`--from-accounts`, `--kinds`, `--out`.

**Reads:** declared record kinds from files or stdin (pipe rule applies).

**Writes:** text tables.

**Example:** `go run ./xops report table --from-cost xops/report/testdata/mixed.jsonl`

## `xops report csv`

Write one CSV per kind under a directory.

**Flags:** `--out DIR` (required), `--from-*`, `--kinds`.

**Reads:** same as `report table`.

**Writes:** `DIR/<kind>.csv`.

**Example:** `go run ./xops report csv --out /tmp/report --from-cost xops/report/testdata/mixed.jsonl`

## `xops report excel`

Write one sheet per kind to an `.xlsx` file.

**Flags:** `--out FILE.xlsx` (required), `--from-*`, `--kinds`.

**Reads:** same as `report table`.

**Writes:** Excel workbook.

**Example:** `go run ./xops report excel --out finops.xlsx --from-cost shared.jsonl`
