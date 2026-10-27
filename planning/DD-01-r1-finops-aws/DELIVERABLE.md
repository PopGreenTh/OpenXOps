# DD-01 — Release 1: FinOps on AWS (outline)

> Outline. Scope and Work Items are fixed once [Q-001](../DECISIONS.md#q-001) is answered.

## Goal

A FinOps analyst chains `xops` and `finops` commands from AWS to an Excel report: accounts and
cost in, cost owned by Project › App › Environment, its trend and impact graded, problems flagged
with the action to take.

```bash
xops   sense aws-accounts                                         > accounts.jsonl
finops sense aws-cost --lz LZ-MAIN                                > cost.jsonl
finops enrich map   --from-cost cost.jsonl --from-accounts accounts.jsonl --taxonomy taxonomy.yaml > map.jsonl
finops enrich share --from-cost map.jsonl                         > shared.jsonl
finops analyze trend   --from-cost shared.jsonl --hierarchy project > trend.jsonl
finops analyze changes --from-map map.jsonl --from-history map-prev.jsonl > changes.jsonl
finops analyze impact  --from-trend trend.jsonl --from-changes changes.jsonl \
                       --from-history impact-prev.jsonl --floor-factor floors.yaml > impact.jsonl
finops enforce unmapped untagged impact --from-cost shared.jsonl --from-impact impact.jsonl \
  | xops report excel --out finops.xlsx
```

## Candidate Work Items

| Work Item | Capability |
|---|---|
| WI-0101 | Skeleton: `xops` and `finops` binaries, `record/`, `scripts/check`, CI, mock data |
| WI-0102 | `xops sense aws-accounts` |
| WI-0103 | `finops sense aws-cost` (two-level Cost Explorer, cache) |
| WI-0104 | `finops sense aws-focus` and `focus-cost` |
| WI-0105 | `finops enrich map` and `share`, taxonomy YAML |
| WI-0106 | `finops analyze trend`, `changes`, `impact` |
| WI-0107 | `finops enforce` rules |
| WI-0108 | `xops report` table, CSV, Excel |
