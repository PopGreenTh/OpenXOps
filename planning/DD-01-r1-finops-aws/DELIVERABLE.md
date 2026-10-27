# DD-01 — Release 1: AWS cost owned by Project › App › Environment

> Scope from D-018 (Q-003 item 2 confirms it). Work Items are written once Q-002 and Q-003 are
> answered.

## Goal

A FinOps analyst chains `xops` and `finops` commands from AWS to an Excel report: accounts and
cost in, every dollar owned by Project › App › Environment (and BusinessUnit › CostCenter),
shared accounts split by the App tag, anything unmapped shown as `UNALLOC` with the reason.

```bash
xops   sense aws-accounts                                          > accounts.jsonl
finops sense aws-cost --lz LZ-MAIN                                 > cost.jsonl
finops enrich map   --from-cost cost.jsonl --from-accounts accounts.jsonl --taxonomy taxonomy.yaml > map.jsonl
finops enrich share --from-cost map.jsonl                          > shared.jsonl
xops   report excel --from-cost shared.jsonl --out finops.xlsx
```

## Work Items

| Work Item | Capability | Blocked by |
|---|---|---|
| WI-0101 | Skeleton: `xops` and `finops` binaries, `record/`, `scripts/check`, CI; developer setup (D-013) | Q-002 |
| WI-0102 | `xops sense aws-accounts`, with the mock organization | — |
| WI-0103 | `finops sense aws-cost`: two-level Cost Explorer, cache, mock cost (D-008) | — |
| WI-0104 | `finops enrich map`: taxonomy YAML, validation, account mapping (D-015) | — |
| WI-0105 | `finops enrich share`: App-tag split (D-016) | Q-003 item 1 |
| WI-0106 | `xops report table|csv|excel`, appending (D-011) | — |

**Release 2** (not in this release): FOCUS, Analyze, Enforce, history and `--as-reported`.
