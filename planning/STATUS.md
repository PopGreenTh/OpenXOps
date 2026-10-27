# Status

Written by Claude only. Status values: see [README.md](README.md#status-values).

## Open questions

| Question | Blocks | State |
|---|---|---|
| [Q-001](DECISIONS.md#q-001): FOCUS, mapping rules, share rules, impact levels, first release scope | — | closed by D-014 to D-018 |
| [Q-004](DECISIONS.md#q-004): licence | — | closed by D-024 |
| [Q-003](DECISIONS.md#q-003): multi-App tag split, Release 1 slice | — | closed by D-021 |
| [Q-002](DECISIONS.md#q-002): editor, OS, task runner, dev container | — | closed by D-019, D-020 |
| [Q-005](DECISIONS.md#q-005): Release 2 scope | DD-02 | open (grilling started 2026-10-27) |

## DD-01 — Release 1: AWS cost owned by Project › App › Environment

| Work Order | Assignee | Depends on | Status | Issue | PR | Branch | Revisions |
|---|---|---|---|---|---|---|---|
| [WO-0101-A](DD-01-r1-finops-aws/WI-0101-Skeleton.md#wo-0101-a--binaries-record-check-ci-and-developer-setup) | Cursor (claimed in PR) | — | merged | [#1](https://github.com/PopGreenTh/OpenXOps/issues/1) | [#3](https://github.com/PopGreenTh/OpenXOps/pull/3), landed through [#2](https://github.com/PopGreenTh/OpenXOps/pull/2) | `cursor/wo-0101-a-skeleton-7589` | 1 |
| [WO-0101-B](DD-01-r1-finops-aws/WI-0101-Skeleton.md#wo-0101-b--pipe-rule-and-review-follow-ups) | open | WO-0103-A, WO-0105-A, WO-0107-A | not started | — | — | — | 0 |
| [WO-0102-A](DD-01-r1-finops-aws/WI-0102-Aws-Accounts.md#wo-0102-a--organizations-discovery-and-mock) | Cursor (claimed in PR) | WO-0101-A | merged | [#6](https://github.com/PopGreenTh/OpenXOps/issues/6) | [#10](https://github.com/PopGreenTh/OpenXOps/pull/10) | `cursor/wo-0102-a-aws-accounts-7589` | 3 |
| [WO-0103-A](DD-01-r1-finops-aws/WI-0103-Aws-Cost.md#wo-0103-a--two-level-cost-explorer-cache-mock-cost) | open | WO-0102-A | to do | [#12](https://github.com/PopGreenTh/OpenXOps/issues/12) | — | — | 0 |
| [WO-0104-A](DD-01-r1-finops-aws/WI-0104-Enrich-Map.md#wo-0104-a--taxonomy-validate-and-map) | Cursor (claimed in PR) | WO-0101-A | merged | [#7](https://github.com/PopGreenTh/OpenXOps/issues/7) | [#15](https://github.com/PopGreenTh/OpenXOps/pull/15) | `cursor/wo-0104-enrich-7589` | 2 |
| [WO-0105-A](DD-01-r1-finops-aws/WI-0105-Enrich-Share.md#wo-0105-a--share-by-app-tag) | open | WO-0104-A | to do | [#18](https://github.com/PopGreenTh/OpenXOps/issues/18) | — | — | 0 |
| [WO-0106-A](DD-01-r1-finops-aws/WI-0106-Report.md#wo-0106-a--table-csv-excel-with-appending) | Cursor (claimed in PR) | WO-0101-A | merged | [#8](https://github.com/PopGreenTh/OpenXOps/issues/8) | [#14](https://github.com/PopGreenTh/OpenXOps/pull/14) | `cursor/wo-0106-report-7589` | 2 |
| [WO-0107-A](DD-01-r1-finops-aws/WI-0107-Analyze-Changes.md#wo-0107-a--changes-between-two-map-outputs) | open | WO-0104-A | to do | [#19](https://github.com/PopGreenTh/OpenXOps/issues/19) | — | — | 0 |

WO-0101-A dispatched 2026-10-27 (#1); PR #3 reviewed 2026-10-27: changes requested, then approved once D-025 dropped the Windows and dev container checks; one CI change asked for D-026 (Windows job optional), done in `104d92c` and approved. `main` created from the plan branch on 2026-10-27. WO-0101-A merged 2026-10-27 through #2 (`c27018d`), whose branch carried #3; #3 closed. WO-0102-A (#6), WO-0104-A (#7) and WO-0106-A (#8) dispatched 2026-10-27 on the repo owner's go-ahead, to run in parallel. Before dispatch, `go.mod`/`go.sum` were added to WO-0102-A's and WO-0104-A's packages, WO-0104-A's *Done when* now reads the demo organization from its own fixture instead of waiting on WO-0102-A, and WO-0106-A's *Done when* follows D-026. PR #10 (WO-0102-A, Cursor) reviewed 2026-10-27: changes requested (live errors swallowed by a shadowed `err`, paging test missing, `aws/` over the 5-file limit, breakpoint not confirmed).

WO-0102-A merged 2026-10-27 (#10, `44e25ff`) after three review rounds. WO-0103-A dispatched 2026-10-27 (#12). Before dispatch, its spec moved Cost Explorer to a new `aws/ce/` package (`aws/` is at the 5-file limit), added `go.mod`/`go.sum` and `aws/retry.go` for `LimitExceededException`, required a settable clock, and spelled out debug logging.

PR #14 (WO-0106-A, Cursor) reviewed 2026-10-27: changes requested (CSV append writes earlier kinds before failing on a later one; `table --out` appends rows under the wrong block). WI-0106 Contracts clarified: on a table append each kind's rows follow a fresh `--- <kind> ---` title line, and all headers are checked before the first write. The Excel *Done when* was verified by the reviewer in LibreOffice.

PR #15 (WO-0104-A, Cursor) reviewed 2026-10-27: changes requested (unknown taxonomy keys caught only at the top level; `map` drops record kinds it doesn't handle; validation-error table covers 4 rules out of 7). All 11 map reference cases pass unchanged.

WO-0106-A merged 2026-10-27 (#14, `d94b7c4`) after two review rounds. Its branch was first updated from `main`: `main` requires `check (ubuntu-latest)` on a branch that is up to date.

PR #15 (WO-0104-A) full re-read 2026-10-27, after its approval on `94c0047`: changes requested. The `environment-unresolved` finding is never emitted, and a shared account with no Environment gets `ENV-UNALLOC` instead of `ENV-UNSPEC`. Neither was covered by a reference case, so map case 12 was added and WI-0104 now names the finding's subject.

WO-0104-A merged 2026-10-27 (#15, `f07c7c5`) after two review rounds and a full re-read, with all 12 map reference cases passing. WO-0105-A (#18) and WO-0107-A (#19) dispatched 2026-10-27. Before dispatch, WO-0105-A's split moved to a new `finops/enrich/share/` package (`finops/enrich/` is at the 5-file limit; the command stays in `commands.go` to avoid an import cycle), and both Work Orders' *Done when* now use fixtures instead of the mock cost that WO-0103-A has not built yet.

WO-0101-B (D-030 pipe rule and the review follow-ups) specified 2026-10-27, to be dispatched after WO-0103-A, WO-0105-A and WO-0107-A merge (D-031). Release 2 grilling started: Q-005.
