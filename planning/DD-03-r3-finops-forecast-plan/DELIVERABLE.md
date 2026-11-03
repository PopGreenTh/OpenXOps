# DD-03 — Release 3: monthly forecast and plan vs actual

Scope: [D-062](../DECISIONS.md#d-062) to [D-065](../DECISIONS.md#d-065), on top of Release 2
(DD-02, refreshed by D-066 to D-068). The planner defaults listed in
[Q-010](../DECISIONS.md#q-010) wait for the repo owner.

## Goal

For each month and node, a FinOps analyst sees actual cost beside a run-rate forecast, a manual
forecast and a plan, with the gap graded, by BusinessUnit, CostCenter, Project, App,
Environment, account, service category and service. A new Project can be planned from its
go-live month before it has any cost.

```bash
xops   sense aws-accounts --tenant T                                                 > accounts.jsonl
finops sense aws-cost     --tenant T --taxonomy taxonomy.yaml                         > cost.jsonl
finops enrich map   --tenant T --taxonomy taxonomy.yaml --from-accounts accounts.jsonl --from-cost cost.jsonl > map.jsonl
finops enrich share --tenant T --taxonomy taxonomy.yaml --from-cost map.jsonl         > shared.jsonl
finops sense plan-lines   --tenant T --taxonomy taxonomy.yaml --save                  > lines.jsonl
finops analyze forecast   --tenant T --from-cost shared.jsonl --from-plan lines.jsonl > forecast.jsonl
finops analyze variance   --tenant T --from-cost shared.jsonl --from-forecast forecast.jsonl --from-plan lines.jsonl > variance.jsonl
xops   report excel       --tenant T --out finops.xlsx < variance.jsonl
xops   report urn-lookup  --tenant T --taxonomy taxonomy.yaml --from-node tree.jsonl --out lookup.csv
```

## Work Items

| Work Item | Capability | Work Orders |
|---|---|---|
| [WI-0301](WI-0301-Service-Category.md) | `service_categories` in the taxonomy; `geo` and `sct` in the cost URNs (D-064) | A |
| [WI-0302](WI-0302-Plan-Lines.md) | `finops sense plan-lines`: plan and manual forecast from Directus (D-062, D-063, D-065) | A |
| [WI-0303](WI-0303-Forecast.md) | `finops analyze forecast`: the run-rate forecast (D-062, D-063), reference cases | A |
| [WI-0304](WI-0304-Variance.md) | `finops analyze variance` and the plan-vs-actual sheet (D-062, D-065), reference cases | A, B |
| [WI-0305](WI-0305-Urn-Lookup.md) | `xops report urn-lookup`: the valid URNs for Directus (D-062) | A |

## Order

```
Release 2 tagged ─► WO-0301-A ─┬─► WO-0302-A ─► WO-0303-A ─► WO-0304-A ─► WO-0304-B
                               └─► WO-0305-A
```

WO-0302-A and WO-0305-A can run at the same time.

## Shared Contracts

### Kinds (D-063)

`cost` stays actuals only. Two new kinds, each with `tenant` required, one record per month and
node:

| Field | `plan` | `forecast` |
|---|---|---|
| `kind` | `plan` | `forecast` |
| `tenant` | the tenant | the tenant |
| `source` | `directus` | `manual` (Directus) or `runrate` (computed) |
| `plan_type` | `budget` or `change` ([Q-010](../DECISIONS.md#q-010) item 2) | — |
| `basis` | — | `runrate`: `run_rate` or `planned_change`; `manual`: `""` |
| `line_id` | the Directus row id | the Directus row id; `""` for `runrate` |
| `month_start`, `period` | `2026-11-01`, `2026-11` | same |
| `business_unit_urn`, `cost_center_urn`, `project_urn`, `app_urn`, `environment_urn`, `account_urn`, `service_category_urn`, `service_urn` | the node and the columns above it; those below `""` (D-065) | same |
| `node_urn`, `node_type` | the deepest URN given and its type (`app`, `acc`, `svc`…) | same |
| `service_category`, `service` | codes, or `""` | same |
| `currency`, `metric` | `USD`, `UnblendedCost` | same |
| `amount` | the month | the month |
| `days_in_month`, `daily_cost` | calendar days; `amount` ÷ `days_in_month` | same |

Both schemas go in `record/schema/` (D-037) in the Work Order that first writes the kind
(WO-0302-A).

### A node and its actuals (D-065)

A line names one node: its deepest URN column. Its **actual** for a month is the sum of the
`cost` records of that month, metric and currency whose URN field of the node's type equals the
node's URN (`app_urn` for an App node, `service_urn` for a service node). The columns above the
node are checked against the taxonomy when the line is read (WI-0302); they never change which
`cost` records count.

## Release criteria

1. CI's Linux job is green; every new command has a Task and a VS Code debug entry.
2. Offline, from a clean clone, with `--tenant testtnt`: `aws-cost --mock | map | share`,
   `plan-lines --mock`, `forecast`, `variance` and `report excel` write a workbook with a
   plan-vs-actual sheet.
3. Every reference case under `finops/analyze/forecast/testdata/cases/` and
   `finops/analyze/variance/testdata/cases/` passes, unchanged.
4. On the mock, every `cost` record's `service_urn` has the shape
   `<account_urn>:geo:global:sct:<category>:svc:<service>`, and every `map` and `share`
   reference case still passes.
5. A plan line naming a URN the taxonomy doesn't hold, or contradicting it, never reaches the
   output; it gives a finding.
6. A month's variance uses the plan saved before the month began, even after the plan was edited
   in Directus (shown with two saved runs).
7. Nothing is written to Directus or to any cloud: the Directus client only sends `GET`.

**Release 4:** Enforce ([D-033](../DECISIONS.md#d-033)).
