# WI-0103 — `finops sense aws-cost`

## Outcome

`finops sense aws-cost` reads Cost Explorer in two levels (D-008) and writes `cost` records per
account, service and App tag value, plus the Project tag totals that `map` checks against the
taxonomy. A closed period is fetched from Cost Explorer only once.

## Contracts

`finops sense aws-cost [--mock] [--profile P] [--lz CODE --taxonomy FILE] [--granularity monthly|daily] [--metric M] [--from P] [--to P] [--app-tag KEY] [--project-tag KEY] [--refresh]`

### Calls (`GetCostAndUsage`, through `aws/`'s retry wrapper, all pages)

| Level | Calls | Group by | Gives |
|---|---|---|---|
| 1 | one per range of uncached periods | `LINKED_ACCOUNT` × tag `--project-tag` | `project_tag_cost` records |
| 2 | one per account (filter `LINKED_ACCOUNT`), per range of uncached periods | `SERVICE` × tag `--app-tag` | `cost` records |

- Level 2's accounts are the accounts level 1 returned. At most 5 calls run at once.
- **Metric** (D-008): `monthly` → `UnblendedCost`, `daily` → `AmortizedCost` by default;
  `--metric` takes any of `UnblendedCost`, `AmortizedCost`, `BlendedCost`, `NetUnblendedCost`,
  `NetAmortizedCost`.
- **Range**: `--from`/`--to` inclusive, `YYYY-MM` (monthly) or `YYYY-MM-DD` (daily). Default:
  monthly, the 24 complete months before the current one plus the current month; daily, the
  last 30 days. Fewer periods available than asked → finding `cost-history-short` (subject: the
  first period returned).
- **Tag keys**: `--app-tag` default `App`, `--project-tag` default `Project`; with `--taxonomy`
  they come from its `tags` section unless the flags are given.
- `--lz` (with `--taxonomy`): the management account reached must be that LZ's (checked with
  `organizations:DescribeOrganization`); records get `lz`.
- IAM: `ce:GetCostAndUsage`, `organizations:DescribeOrganization`.

### Records

Common fields: `csp` `aws`, `management_account`, `granularity`, `period`, `metric`, `amount`,
`currency` (Cost Explorer's `Unit`), `estimated`; `lz` with `--lz`.

| `kind` | Own fields | One record per |
|---|---|---|
| `cost` | `account_id`, `service`, `app_tag` | account, service, App tag value, period |
| `project_tag_cost` | `account_id`, `project_tag` | account, Project tag value, period |

Tag values are trimmed; `""` means untagged (Cost Explorer's `<key>$`). Sorted by `account_id`,
`period`, then `service` and `app_tag` (or `project_tag`); `project_tag_cost` records first.

### Cache (D-007)

```
data/cache/aws/ce/<management_account>/<level>-<granularity>-<metric>-<tag key>[-<account>]/<period>.json
data/cache/aws/ce/<management_account>/<level>-<granularity>-<metric>-<tag key>[-<account>]/<period>@<UTC timestamp>.json
```

- **Closed period** (a month before the current one; a day before yesterday): written once as
  `<period>.json`, never fetched again.
- **Open period** (the current month; yesterday and today): written as `<period>@<timestamp>.json`
  on every fetch; reused when under 12 hours old.
- `--refresh` fetches every period in range and writes new `@` files; a closed `<period>.json`
  is never overwritten. The newest file of a period wins.
- Files are written once and never edited. `--mock` uses `data/mock/cache/…`.
- After each run, stderr shows the number of Cost Explorer calls and their cost at $0.01 each
  (at any log level).

### Mock

Deterministic, embedded: amounts per account, service and period seeded from all three; the
demo organization's accounts (WI-0102) plus one account not in it; on the shared account, App
tag values that are a single App, two Apps (`APP-CHK,APP-PAY`), lower case, a wrong App and
empty; Project tag values that agree and one that disagrees with the demo taxonomy. 26 months of
monthly data, 60 days of daily data.

## WO-0103-A — Two-level Cost Explorer, cache, mock cost

**Assignee:** open
**Depends on:** WO-0102-A (uses `aws/`)
**Packages:** `aws/ce/` (new: the Cost Explorer client and the mock cost), `aws/retry.go` (retry
codes only), `finops/sense/`, `finops/main.go`, `Taskfile.yml`, `.vscode/launch.json`,
`README.md`, `go.mod`, `go.sum`

`aws/` already has the 5 non-test `.go` files `AGENTS.md` §5 allows, so Cost Explorer gets its
own package, `aws/ce/`. It uses `aws/`'s config, retry wrapper and demo organization, and keeps
its fixtures in `aws/ce/testdata/`. The cache lives in `finops/sense/`.

**Goal.** `finops sense aws-cost` as specified.

**In scope.**
- The Cost Explorer client behind an interface, the two levels, the cache, the mock, `--lz`.
- `aws/retry.go`: retry Cost Explorer's throttling error, `LimitExceededException`.
- The current time comes from a clock the tests can set (a function value or parameter, not
  `time.Now()` deep in the logic). Open and closed periods, the default range and the mock's
  periods are all relative to it, so the tests and the golden output don't change with the date.
- Debug logging as WI-0101 requires: `record.LogStart`, and one `slog.Debug` per Cost Explorer
  call and per cache hit or miss.
- Task `run:finops:sense-aws-cost` and a launch entry on `--mock`.
- Tests with a counting fake client: a second run over closed periods makes **zero** calls; an
  open period is reused under 12 hours and fetched after; `--refresh` never overwrites a closed
  file; a different `--metric` or tag key is a different cache entry; level 2 makes one call
  per account; parsing `App$APP-CHK` and `App$`; `cost-history-short`.
- README: the command, records, IAM actions, activating the tag keys as cost allocation tags,
  and turning on multi-year data for 24 months.

**Out of scope.** FOCUS (Release 2). Splitting cost by App (WI-0105).

**Done when.**
- `finops sense aws-cost --mock` prints 25 monthly periods of `cost` and `project_tag_cost`
  records for the mock accounts, sorted as specified; `--granularity daily` prints 30 days of
  `AmortizedCost`.
- The counting-client tests pass.
- Per period, metric and currency, the `cost` total equals the `project_tag_cost` total.

**Verify.** `go run ./scripts/check` and `go run ./finops sense aws-cost --mock`

## Delivered (as built)

Built by WO-0103-A (#22, `ff91c89`) after four review rounds.

- `aws/ce/`: the Cost Explorer client behind an interface, plus the mock (26 months, 60 days)
  and its seed. `aws/retry.go` retries `LimitExceededException`.
- `finops/sense/`: `aws-cost`, the two levels, the cache and the periods, with a clock the tests
  can set.
  - A first mock run makes 10 calls (1 at level 1, plus 1 per account); a rerun makes 0.
  - A closed month fetched again with `--refresh` stays the newest file and never expires.
- `cost-history-short` fires on live data when the first periods of the range have no cost.
  Its subject is the first period returned.
- Adds `github.com/aws/aws-sdk-go-v2/service/costexplorer`, which raised the AWS SDK core to
  v1.30.0.
