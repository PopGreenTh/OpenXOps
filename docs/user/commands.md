# Commands

Every command is `<binary> <stage> <thing>`. Stages: **Sense › Enrich › Analyze › Enforce › Report**.
All commands accept `--log-level` on stderr; records go to stdout (except `report table` without
`--out`).

**Tenant (D-045).** Every command except `version` takes a required `--tenant ID`: lowercase
letters and digits, joined by single dashes (`^[a-z0-9]+(-[a-z0-9]+)*$`, for example `testtnt` or
`acme-retail`). A missing or malformed id stops the command before it reads anything.

- Every record a command writes has `tenant` set to `--tenant`. Records passed through are
  unchanged.
- Every record a command reads that has a `tenant` field must equal `--tenant`, whatever its
  kind; otherwise the command stops with an error naming the file (or `stdin`), the line and both
  tenants.
- `validate`, `map`, `share` and `aws-cost --taxonomy` stop when the taxonomy's `tenant` differs
  from `--tenant` ([taxonomy.md](taxonomy.md)).
- The Cost Explorer cache stays keyed by management account, not by tenant.

## `xops version`

Print version, commit and build date.

**Example:** `go run ./xops version`

## `finops version`

Same for the FinOps binary.

**Example:** `go run ./finops version`

## `xops sense aws-accounts`

List an AWS Organization as `ou` and `account` records (OUs first, then accounts, each sorted by
`id`).

**Flags:** `--tenant ID` (required), `--mock`, `--profile`, `--region` (default `us-east-1`), `--taxonomy` with an optional `--lz`. With `--taxonomy`, the records carry the taxonomy's one LZ as `lz` (D-052); a `--lz` that names another code stops the command with both codes in the error, and `--lz` without `--taxonomy` is an error.

**Reads:** nothing (live AWS or mock).

**Writes:** `ou`, `account`; may emit `finding` (`ou-path-ambiguous`).

**Example:** `go run ./xops sense aws-accounts --tenant testtnt --mock > accounts.jsonl`

## `finops sense aws-cost`

Two-level Cost Explorer read: `project_tag_cost` then `cost` records.

**Flags:** `--tenant ID` (required), `--mock`, `--profile`, `--region`, `--granularity monthly|daily`, `--metric`,
`--from` / `--to`, `--app-tag`, `--project-tag`, `--taxonomy` (its `tenant` must be `--tenant`)
with an optional `--lz` that defaults to its one LZ, `--refresh`.

**Reads:** nothing (live AWS or mock).

**Writes:** `project_tag_cost`, `cost`; may emit `finding` (`cost-history-short`).

**Example:** `go run ./finops sense aws-cost --tenant testtnt --mock > cost.jsonl`

## `finops enrich validate`

Check a taxonomy YAML file and print `finding` records.

**Flags:** `--tenant ID` (required), `--taxonomy FILE` (required).

**Reads:** taxonomy file.

**Writes:** `finding` (taxonomy codes); exits 1 on severity `error`.

**Example:** `go run ./finops enrich validate --tenant testtnt --taxonomy finops/enrich/testdata/demo-taxonomy.yaml`

## `finops enrich map`

Apply taxonomy to accounts and optional cost; pass through inventory; write `allocation` rows.

**Flags:** `--tenant ID` (required), `--taxonomy FILE` (required), `--from-accounts`, `--from-cost`, `--from-finding`, …

**Reads:** `ou`, `account`, optional `cost`, optional upstream `finding`.

**Writes:** pass-through inventory, `allocation`, enriched optional `cost`, mapping `finding`
records. `allocation` and `cost` carry the node ids and the `*_urn` fields ([taxonomy.md](taxonomy.md#urns-d-046),
[records.md](records.md)); a shared account's `cost` leaves the App-level URNs `""` for `share`.

**Example:**

```bash
go run ./finops enrich map --tenant testtnt --taxonomy finops/enrich/testdata/demo-taxonomy.yaml \
  --from-accounts finops/enrich/testdata/demo-accounts.jsonl \
  --from-cost finops/enrich/testdata/demo-cost.jsonl
```

## `finops enrich share`

Split shared-account `cost` by App tag using `allocation` rows from `map`.

**Flags:** `--tenant ID` (required), `--taxonomy FILE` (required), `--from-cost`, `--from-allocation`, …

**Reads:** `cost`, `allocation`, pass-through kinds from `map`.

**Writes:** updated `cost` rows with `app` / `project` / `environment` and every `*_urn` field
filled from the App each row lands on; share `finding` records.

**Example:**

```bash
go run ./finops enrich share --tenant testtnt --taxonomy finops/enrich/testdata/demo-taxonomy.yaml \
  --from-cost map.jsonl
```

## `finops analyze changes`

Compare a previous map output to the current one; emit `change` records, then pass through input.

**Flags:** `--tenant ID` (required), `--before FILE` (required), `--from-map FILE` or stdin.

**Reads:** prior and current map (or share) output.

**Writes:** `change`, then pass-through records from current input.

**Example:** `go run ./finops analyze changes --tenant testtnt --before before.jsonl --from-map now.jsonl`

## `xops enrich lz-tree`

Turn `ou` and `account` records into one flat `node` record per node of the tenant's cloud
hierarchy: the landing zone, every OU and every account (D-054). The LZ is the taxonomy's one LZ
(D-052). URNs are built as `finops enrich map` builds `lz_urn`, `ou_urn` and `account_urn`, so the
two agree.

**Flags:** `--tenant ID` (required), `--taxonomy FILE` (required; its `tenant` must be `--tenant`
and it must have exactly one LZ), `--from-accounts FILE` or stdin.

**Reads:** `ou` and `account` records. Every one's `management_account` must be the LZ's, or the
command stops with the file (or `stdin`), the line and both accounts.

**Writes:** only `node` records: the LZ node, then the OU nodes sorted by `urn`, then the account
nodes sorted by `urn`. Input records are not passed through. The same input gives the same output.

**Example:**

```bash
go run ./xops sense aws-accounts --tenant testtnt --mock > accounts.jsonl
go run ./xops enrich lz-tree --tenant testtnt --taxonomy xops/enrich/testdata/demo-taxonomy.yaml \
  --from-accounts accounts.jsonl > tree.jsonl
```

## `xops report tree`

Write the `node` records as one nested YAML file people can read: the LZ at the top and every
node's `children` under it, OUs before accounts, each group in `id` order. A node without children
has no `children` key.

**Flags:** `--tenant ID` (required), `--from-node FILE` or stdin, `--out FILE` (the YAML goes to
stdout when it is not given). An existing `--out` file is replaced: a tree is a snapshot, not a log.

**Reads:** `node` records.

**Writes:** one YAML document. The command fails, writing nothing, when there isn't exactly one
`lzc` node, when two nodes have the same `urn`, or when a node's `parent_urn` names no node; the
error names the file (or `stdin`), the line and the URN.

**Example:** `go run ./xops report tree --tenant testtnt --from-node tree.jsonl --out tree.yaml`

## `xops report table`

Print aligned tables per record kind (stdout or append with `--out`).

**Flags:** `--tenant ID` (required), `--from-cost`, `--from-change`, `--from-allocation`, `--from-finding`,
`--from-accounts`, `--kinds`, `--out`.

**Reads:** declared record kinds from files or stdin (pipe rule applies).

**Writes:** text tables.

**Example:** `go run ./xops report table --tenant testtnt --from-cost xops/report/testdata/mixed.jsonl`

## `xops report csv`

Write one CSV per kind under a directory.

**Flags:** `--tenant ID` (required), `--out DIR` (required), `--from-*`, `--kinds`.

**Reads:** same as `report table`.

**Writes:** `DIR/<kind>.csv`.

**Example:** `go run ./xops report csv --tenant testtnt --out /tmp/report --from-cost xops/report/testdata/mixed.jsonl`

## `xops report excel`

Write one sheet per kind to an `.xlsx` file.

**Flags:** `--tenant ID` (required), `--out FILE.xlsx` (required), `--from-*`, `--kinds`.

**Reads:** same as `report table`.

**Writes:** Excel workbook.

**Example:** `go run ./xops report excel --tenant testtnt --out finops.xlsx --from-cost shared.jsonl`
