# DD-02 — Release 2: FOCUS, history, trend and impact

Scope: D-032 to D-041, with D-007, D-010, D-014, D-017 and D-022. Planner defaults still to be
confirmed are listed in [Q-006](../DECISIONS.md#q-006); no Work Order is dispatched until it closes.

## Goal

A FinOps analyst sees how cost moved month on month, fairly across months of different lengths,
graded by impact, with the history to tell a new movement from an ongoing one, and with region and
`Feature` detail from FOCUS when wanted.

```bash
xops   sense aws-accounts                                                          > accounts.jsonl
finops sense aws-focus --taxonomy taxonomy.yaml --lz LZ-MAIN                        > files.jsonl
xops   sense focus-cost --from-focus files.jsonl                                    > cost.jsonl
finops enrich map   --taxonomy taxonomy.yaml --from-accounts accounts.jsonl --from-cost cost.jsonl --save > map.jsonl
finops enrich share --taxonomy taxonomy.yaml --from-cost map.jsonl --save          > shared.jsonl
finops analyze trend  --hierarchy project --from-cost shared.jsonl --save          > trend.jsonl
finops analyze impact --from-trend trend.jsonl --from-history data/history/impact/<last>.jsonl --save > impact.jsonl
xops   report excel --out finops.xlsx < impact.jsonl
```

`aws-cost` stays the default source of cost (D-035); FOCUS is opt-in.

## Work Items

| Work Item | Capability | Work Orders |
|---|---|---|
| [WI-0201](WI-0201-History.md) | The history store and `--save` (D-034, D-039) | A |
| [WI-0202](WI-0202-Focus.md) | `xops sense focus-cost` with DuckDB; `finops sense aws-focus` (D-014, D-022, D-035, D-038) | A, B |
| [WI-0203](WI-0203-Analyze-Trend.md) | `finops analyze trend`, monthly on average daily cost (D-035, D-041), reference cases | A |
| [WI-0204](WI-0204-Analyze-Impact.md) | `finops analyze impact` (D-010, D-017, D-035), reference cases | A |
| [WI-0205](WI-0205-As-Reported.md) | Taxonomy snapshots and `--as-reported` (D-007, D-040) | A |

## Order

```
Release 1 done, WO-0101-B ─┬─► WO-0201-A ─┬─► WO-0203-A ─► WO-0204-A
                           │              └─► WO-0205-A
                           └─► WO-0202-A ─► WO-0202-B
```

WO-0201-A and WO-0202-A can run at the same time; so can WO-0203-A and WO-0205-A.

A Work Order that adds a record kind (`focus_file`, `trend`, `impact`) adds its schema in
`record/schema/` (D-037), and the new fields on `cost` (`region`, `feature_tag`, `source`,
`snapshot`) are optional in its schema.

## Release criteria

1. CI's Linux job is green with CGO and the `no_duckdb_arrow` tag (D-022); every command has a Task
   and a VS Code debug entry.
2. Offline, from a clean clone: `aws-focus --mock | focus-cost | map | share | trend | impact |
   report excel` writes a workbook.
3. Every reference case in `finops/analyze/testdata/cases/` passes, unchanged.
4. On the mock, for every closed month, `focus-cost` totals equal `aws-cost` totals for the same
   metric and currency.
5. A second `aws-focus` run over closed months downloads nothing.
6. Two `impact` runs, the second with `--from-history` the first's saved output, mark a
   still-moving node `ongoing` and a settled one `resolved`.
7. `map --as-reported` maps an account's old periods with the taxonomy saved for them.

**Release 3:** Enforce (D-032, D-033).
