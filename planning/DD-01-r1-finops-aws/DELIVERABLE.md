# DD-01 — Release 1: AWS cost owned by Project › App › Environment

Scope: D-018 and D-021.

## Goal

A FinOps analyst chains `xops` and `finops` commands from AWS to an Excel report: accounts and
cost in, every dollar owned by Project › App › Environment (and BusinessUnit › CostCenter),
shared accounts split by the App tag, anything unmapped shown as `UNALLOC` with the reason.

```bash
xops   sense aws-accounts                                          > accounts.jsonl
finops sense aws-cost --lz LZ-MAIN                                 > cost.jsonl
finops enrich map   --from-cost cost.jsonl --from-accounts accounts.jsonl --taxonomy taxonomy.yaml > map.jsonl
finops enrich share --from-cost map.jsonl                          > shared.jsonl
finops analyze changes --from-map shared.jsonl --before last-month.jsonl > changes.jsonl
xops   report excel --from-cost shared.jsonl --from-change changes.jsonl --out finops.xlsx
```

## Work Items

| Work Item | Capability | Work Order |
|---|---|---|
| [WI-0101](WI-0101-Skeleton.md) | `xops` and `finops` binaries, `record/`, check, CI; developer setup (D-013, D-019, D-020) | A |
| [WI-0102](WI-0102-Aws-Accounts.md) | `xops sense aws-accounts`, demo organization | A |
| [WI-0103](WI-0103-Aws-Cost.md) | `finops sense aws-cost`: two-level Cost Explorer, cache, mock cost (D-008) | A |
| [WI-0104](WI-0104-Enrich-Map.md) | Taxonomy, `finops enrich validate` and `map` (D-015), 11 reference cases | A |
| [WI-0105](WI-0105-Enrich-Share.md) | `finops enrich share` (D-016, D-021), 6 reference cases | A |
| [WI-0106](WI-0106-Report.md) | `xops report table\|csv\|excel`, appending (D-011) | A |
| [WI-0107](WI-0107-Analyze-Changes.md) | `finops analyze changes` (D-021) | A |
| [WI-0108](WI-0108-Docs.md) | User and developer docs, doc check (D-044) | A, B |

## Order

```
WO-0101-A ─┬─► WO-0102-A ─► WO-0103-A
           ├─► WO-0104-A ─┬─► WO-0105-A
           │              └─► WO-0107-A
           └─► WO-0106-A
```

After WO-0101-A, three can run at once: WO-0102-A, WO-0104-A and WO-0106-A touch different
packages.

## Release criteria

1. CI's Linux job is green (the Windows job is optional, D-026), and every command has a Task
   and a VS Code debug entry.
2. The pipeline above runs on `--mock` from a clean clone with no AWS credentials and writes an
   Excel workbook.
3. Every reference case in `finops/enrich/testdata/cases/` passes, unchanged.
4. After `share`, for every period, metric and currency, total cost equals the Cost Explorer
   total, and no `cost` record is left without an App.
5. A second `aws-cost` run over closed months makes zero Cost Explorer calls.
6. Against the repo owner's real AWS Organization, the pipeline completes with the read-only
   permissions listed in `docs/user/aws-setup.md`.
7. The docs (D-044): `scripts/check` prints `docs PASS`, and `docs/user/getting-started.md`'s
   `--mock` walkthrough runs as written on a clean clone.

**Checked 2026-10-27** on a clean clone of `main` at `9da24f3` with no AWS credentials:

| # | Criterion | Result |
|---|---|---|
| 1 | CI Linux green; a Task and a debug entry per command | ✅ |
| 2 | The pipeline runs on `--mock` and writes an Excel workbook | ✅ `finops.xlsx` written |
| 3 | Every reference case passes unchanged | ✅ 12 map and 6 share cases |
| 4 | After `share`, totals equal Cost Explorer's; no `cost` without an App | ✅ largest difference 5e-13; 0 without an App |
| 5 | A second `aws-cost` run over closed months makes zero calls | ✅ 10 calls, then 0 |
| 6 | The pipeline completes against the real Organization, read-only | ⏳ for the repo owner to run |
| 7 | Docs: `docs PASS`; the getting-started walkthrough runs on a clean clone | ⏳ after WI-0108 (added 2026-10-27, D-044) |

**Release 2:** FOCUS (`aws-focus`, `focus-cost`), `analyze trend` and `impact`, Enforce, the
history store and `--as-reported`.
