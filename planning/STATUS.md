# Status

Written by Claude only. Status values: see [README.md](README.md#status-values).

## Open questions

| Question | Blocks | State |
|---|---|---|
| [Q-001](DECISIONS.md#q-001): FOCUS, mapping rules, share rules, impact levels, first release scope | — | closed by D-014 to D-018 |
| [Q-004](DECISIONS.md#q-004): licence | — | closed by D-024 |
| [Q-003](DECISIONS.md#q-003): multi-App tag split, Release 1 slice | — | closed by D-021 |
| [Q-002](DECISIONS.md#q-002): editor, OS, task runner, dev container | — | closed by D-019, D-020 |

## DD-01 — Release 1: AWS cost owned by Project › App › Environment

| Work Order | Assignee | Depends on | Status | Issue | PR | Branch | Revisions |
|---|---|---|---|---|---|---|---|
| [WO-0101-A](DD-01-r1-finops-aws/WI-0101-Skeleton.md#wo-0101-a--binaries-record-check-ci-and-developer-setup) | Cursor (claimed in PR) | — | merged | [#1](https://github.com/PopGreenTh/OpenXOps/issues/1) | [#3](https://github.com/PopGreenTh/OpenXOps/pull/3), landed through [#2](https://github.com/PopGreenTh/OpenXOps/pull/2) | `cursor/wo-0101-a-skeleton-7589` | 1 |
| [WO-0102-A](DD-01-r1-finops-aws/WI-0102-Aws-Accounts.md#wo-0102-a--organizations-discovery-and-mock) | open | WO-0101-A | to do | [#6](https://github.com/PopGreenTh/OpenXOps/issues/6) | — | — | 0 |
| [WO-0103-A](DD-01-r1-finops-aws/WI-0103-Aws-Cost.md#wo-0103-a--two-level-cost-explorer-cache-mock-cost) | open | WO-0102-A | not started | — | — | — | 0 |
| [WO-0104-A](DD-01-r1-finops-aws/WI-0104-Enrich-Map.md#wo-0104-a--taxonomy-validate-and-map) | open | WO-0101-A | to do | [#7](https://github.com/PopGreenTh/OpenXOps/issues/7) | — | — | 0 |
| [WO-0105-A](DD-01-r1-finops-aws/WI-0105-Enrich-Share.md#wo-0105-a--share-by-app-tag) | open | WO-0104-A | not started | — | — | — | 0 |
| [WO-0106-A](DD-01-r1-finops-aws/WI-0106-Report.md#wo-0106-a--table-csv-excel-with-appending) | open | WO-0101-A | to do | [#8](https://github.com/PopGreenTh/OpenXOps/issues/8) | — | — | 0 |
| [WO-0107-A](DD-01-r1-finops-aws/WI-0107-Analyze-Changes.md#wo-0107-a--changes-between-two-map-outputs) | open | WO-0104-A | not started | — | — | — | 0 |

WO-0101-A dispatched 2026-10-27 (#1); PR #3 reviewed 2026-10-27: changes requested, then approved once D-025 dropped the Windows and dev container checks; one CI change asked for D-026 (Windows job optional), done in `104d92c` and approved. `main` created from the plan branch on 2026-10-27. WO-0101-A merged 2026-10-27 through #2 (`c27018d`), whose branch carried #3; #3 closed. WO-0102-A (#6), WO-0104-A (#7) and WO-0106-A (#8) dispatched 2026-10-27 on the repo owner's go-ahead, to run in parallel. Before dispatch, `go.mod`/`go.sum` were added to WO-0102-A's and WO-0104-A's packages, WO-0104-A's *Done when* now reads the demo organization from its own fixture instead of waiting on WO-0102-A, and WO-0106-A's *Done when* follows D-026.
