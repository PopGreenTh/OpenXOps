# DD-02 — Release 2: FOCUS, history, trend and impact

Scope: D-032 to D-042, with D-007, D-010, D-014, D-017 and D-022. The planner defaults listed
in [Q-006](../DECISIONS.md#q-006) were confirmed by [D-042](../DECISIONS.md#d-042).

**Refreshed 2026-11-03** ([D-066](../DECISIONS.md#d-066)) for tenants and URNs (D-045 to D-049),
the one-LZ and code rules (D-052, D-053, D-056), the nested taxonomy (D-057 to D-061), URN
nodes in `trend` and `impact` ([D-067](../DECISIONS.md#d-067)) and `first_seen` / `last_changed`
([D-068](../DECISIONS.md#d-068)). Every command takes `--tenant`; every record it writes has
`tenant`; the taxonomy is the nested layout read by `record/taxonomy`.

## Goal

A FinOps analyst sees how cost moved month on month, fairly across months of different lengths,
graded by impact, with the history to tell a new movement from an ongoing one, and with region and
`Feature` detail from FOCUS when wanted.

```bash
xops   sense aws-accounts --tenant T --save --from-history <last aws-accounts run>  > accounts.jsonl
finops sense aws-focus --tenant T --taxonomy taxonomy.yaml                         > files.jsonl
xops   sense focus-cost --tenant T --from-focus files.jsonl                        > cost.jsonl
finops enrich map   --tenant T --taxonomy taxonomy.yaml --from-accounts accounts.jsonl --from-cost cost.jsonl --save > map.jsonl
finops enrich share --tenant T --taxonomy taxonomy.yaml --from-cost map.jsonl --save          > shared.jsonl
finops analyze trend  --tenant T --hierarchy project --from-cost shared.jsonl --save          > trend.jsonl
finops analyze impact --tenant T --from-trend trend.jsonl --from-history data/history/T/impact/<last>.jsonl --save > impact.jsonl
xops   report excel --tenant T --out finops.xlsx < impact.jsonl
```

`aws-cost` stays the default source of cost (D-035); FOCUS is opt-in.

## Work Items

| Work Item | Capability | Work Orders |
|---|---|---|
| [WI-0201](WI-0201-History.md) | The history store and `--save` (D-034, D-039); `first_seen` and `last_changed` on `ou` and `account` (D-051, D-068) | A, B |
| [WI-0202](WI-0202-Focus.md) | `xops sense focus-cost` with DuckDB; `finops sense aws-focus` (D-014, D-022, D-035, D-038) | A, B |
| [WI-0203](WI-0203-Analyze-Trend.md) | `finops analyze trend`, monthly on average daily cost (D-035, D-041), reference cases | A |
| [WI-0204](WI-0204-Analyze-Impact.md) | `finops analyze impact` (D-010, D-017, D-035), reference cases | A |
| [WI-0205](WI-0205-As-Reported.md) | Taxonomy snapshots and `--as-reported` (D-007, D-040) | A |

## Order

```
Release 1 done, WO-0101-B ─┬─► WO-0201-A ─┬─► WO-0203-A ─► WO-0204-A
                           │              ├─► WO-0205-A
                           │              └─► WO-0201-B
                           └─► WO-0202-A ─► WO-0202-B
```

WO-0201-A and WO-0202-A can run at the same time; so can WO-0203-A, WO-0205-A and WO-0201-B.

A Work Order that adds a record kind (`focus_file`, `trend`, `impact`) adds its schema in
`record/schema/` (D-037), with `tenant` required (D-045). The new fields on `cost` (`region`,
`feature_tag`, `source`, `snapshot`) and on `ou` and `account` (`first_seen`, `last_changed`) are
optional in their schemas.

## Release criteria

1. CI's Linux job is green with CGO and the `no_duckdb_arrow` tag (D-022); every command has a Task
   and a VS Code debug entry.
2. Offline, from a clean clone, with `--tenant testtnt`: `aws-focus --mock | focus-cost | map |
   share | trend | impact | report excel` writes a workbook, and every record in it has `tenant`.
3. Every reference case in `finops/analyze/testdata/cases/` passes, unchanged.
4. On the mock, for every closed month, `focus-cost` totals equal `aws-cost` totals for the same
   metric and currency.
5. A second `aws-focus` run over closed months downloads nothing.
6. Two `impact` runs, the second with `--from-history` the first's saved output, mark a
   still-moving node `ongoing` and a settled one `resolved`.
7. `map --as-reported` maps an account's old periods with the taxonomy saved for them.
8. Two `aws-accounts --mock --save` runs, the second with `--from-history` the first's file and
   one account renamed in the mock, keep every `first_seen` and move only that account's
   `last_changed`.

**Release 3:** monthly forecast and plan vs actual ([D-062](../DECISIONS.md#d-062)). Enforce is
Release 4.
