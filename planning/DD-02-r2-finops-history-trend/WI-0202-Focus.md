# WI-0202 — FOCUS: `xops sense focus-cost` and `finops sense aws-focus`

## Outcome

Cost with region and `Feature` detail, read from the cloud's FOCUS export (D-014). `aws-focus`
fetches AWS's export into a local cache (D-038); `focus-cost`, shared by every cloud, turns FOCUS
files into the same `cost` and `project_tag_cost` records `aws-cost` writes, so every later stage
works unchanged. FOCUS is opt-in; `aws-cost` stays the default (D-035).

## Contracts

### `xops sense focus-cost [--from-focus FILE] [--file PATH]... [--metric M] [--app-tag KEY] [--project-tag KEY] [--feature-tag KEY]`

- **Input:** `focus_file` records (pipe or `--from-focus`), or FOCUS files named with `--file`.
  A file ending `.parquet` or `.csv` is read with DuckDB (`read_parquet`, `read_csv`); anything
  else is an error naming the file.
- **FOCUS 1.0 columns used:** `BillingAccountId`, `SubAccountId`, `ProviderName`,
  `ChargePeriodStart`, `ServiceName`, `RegionId`, `BillingCurrency`, `BilledCost`,
  `EffectiveCost`, `Tags` (a JSON object). A missing column is an error naming the file and the
  column.
- **Metric:** `UnblendedCost` (default) reads `BilledCost`; `AmortizedCost` reads `EffectiveCost`.
  Any other metric is an error: FOCUS doesn't provide it.
- **Monthly only** in Release 2: rows are grouped by the month of `ChargePeriodStart`.
- **Output:** the WI-0103 records with the same fields and sort order, plus three fields on
  `cost` (`region`, `feature_tag`, `source` `focus`) and `source` `focus` on `project_tag_cost`.
  `cost` is one record per account, service, region, App tag value, Feature tag value and period;
  `project_tag_cost` per account, Project tag value and period. `management_account` is
  `BillingAccountId`; `csp` is `ProviderName` lower-cased (`aws`). Tag values are trimmed; a
  missing tag is `""`.
- **Tag keys:** `--app-tag` `App`, `--project-tag` `Project`, `--feature-tag` `Feature` by default.
- Every input `focus_file` record passes through after the output (pass-through rule).

### `finops sense aws-focus --taxonomy FILE --lz CODE [--from P] [--to P] [--refresh] [--profile P] [--mock]`

- **Where the export lives:** a new optional `focus` entry on the LZ in the taxonomy (D-035):
  `lzs: [{ code: LZ-MAIN, …, focus: { bucket: my-bucket, prefix: exports/focus/data } }]`. The
  prefix is the export's `data/` folder; months are its `BILLING_PERIOD=YYYY-MM/` subfolders. No
  `focus` entry on that LZ is an error naming the LZ.
- **Cache (D-038):** `data/cache/aws/focus/<management_account>/<YYYY-MM>/<file name>`. A closed
  month (before the current one) is downloaded once and never again; the open month is downloaded
  again on every run into a fresh `<YYYY-MM>@<UTC timestamp>/` folder, the newest of which is
  used. `--refresh` downloads every month in range into fresh `@` folders; a closed month's plain
  folder is never overwritten. Every S3 call goes through `aws/`'s retry wrapper.
- **Range:** `--from`/`--to` `YYYY-MM`, inclusive; default the 24 complete months before the
  current one plus the current month (as `aws-cost`).
- **Output:** one `focus_file` record per file in the cache used for each month in range:
  `csp` `aws`, `lz`, `management_account`, `period`, `path` (the local file), `bytes`, `closed`
  (bool). Sorted by `period`, then `path`.
- stderr shows the number of files downloaded and their total size (any log level).
- **IAM:** `s3:ListBucket` and `s3:GetObject` on the export's bucket and prefix.
- **Mock:** `--mock` emits `focus_file` records for the mock FOCUS files (WO-0202-A), copied into
  `data/mock/cache/aws/focus/…`; no AWS call.

### Mock FOCUS data

CSV files, one per month, in `xops/sense/testdata/focus/`, generated for the same accounts,
services, App tag values and Project tag values as WI-0103's mock, with every amount equal to the
`aws-cost` mock's, plus regions (`us-east-1`, `eu-west-1`) and `Feature` tag values on some rows.
So for every closed month the totals of `focus-cost` and `aws-cost --mock` agree (release
criterion 4).

## WO-0202-A — `focus-cost` and DuckDB

**Assignee:** open
**Depends on:** Release 1's Work Orders and WO-0101-B
**Packages:** `xops/sense/` (one new file for `focus-cost`; the package then has 5 non-test files),
`record/schema/` (the `focus_file` schema),
`xops/sense/testdata/focus/` (new), `scripts/check/`, `.github/workflows/check.yml`, `Taskfile.yml`,
`.vscode/launch.json`, `docs/develop.md`, `README.md`, `go.mod`, `go.sum`

**Goal.** `focus-cost` as specified, and DuckDB in the build (D-022).

**In scope.**
- DuckDB through `github.com/marcboeker/go-duckdb` (MIT), pinned to a version that builds with the
  module's Go version; CGO on. Build tag `no_duckdb_arrow` on every build and test: in
  `scripts/check`, the CI workflow (with CGO and a C compiler on `ubuntu-latest`; the Windows job
  stays optional), the Taskfile and every launch entry.
- `docs/develop.md`: the C toolchain needed on Linux.
- The command; the mock FOCUS CSV files; a Task and a launch entry on them.
- Tests: grouping and sums on a small CSV; `BilledCost` vs `EffectiveCost`; an unsupported metric;
  a missing column; tag parsing from `Tags` (present, absent, padded); `.parquet` read (a test
  writes one with DuckDB `COPY … TO … (FORMAT parquet)` into `t.TempDir()`); for every closed
  month of the mock, totals equal `aws-cost --mock`'s (release criterion 4).

**Out of scope.** Fetching from S3 (WO-0202-B). Daily granularity.

**Done when.**
- The tests above pass on Linux CI with CGO and `no_duckdb_arrow`.
- `xops sense focus-cost --file xops/sense/testdata/focus/<a month>.csv | finops enrich map …`
  runs end to end (shown in the PR).

**Verify.** `go run -tags=no_duckdb_arrow ./scripts/check`

## WO-0202-B — `aws-focus`

**Assignee:** open
**Depends on:** WO-0202-A
**Packages:** `finops/sense/` (one new file), `aws/focus/` (new: the S3 client behind an interface,
listing and download through `aws/`'s retry wrapper), `finops/enrich/taxonomy.go` (the optional
`focus` entry on an LZ, and its validation), `Taskfile.yml`, `.vscode/launch.json`, `README.md`,
`go.mod`, `go.sum`

**Goal.** `aws-focus` as specified.

**In scope.**
- The LZ `focus` entry in the taxonomy: `bucket` and `prefix`, both required when `focus` is set;
  `validate` reports a missing one as an error. WI-0104's taxonomy contract gains the entry.
- The command, the cache, `--refresh`, `--mock`.
- Tests with a counting fake S3 client (never a real endpoint): a second run over closed months
  downloads nothing; the open month is downloaded again each run into a new `@` folder and the
  newest is used; `--refresh` never overwrites a closed month's folder; paging through more than
  one list page; no `focus` entry is an error.
- README: the command, the export layout, the IAM actions.

**Out of scope.** Creating the FOCUS export in AWS (the README says how).

**Done when.**
- The tests above pass.
- `finops sense aws-focus --mock --taxonomy … --lz LZ-MAIN | xops sense focus-cost` prints `cost`
  records for 25 months (shown in the PR).

**Verify.** `go run -tags=no_duckdb_arrow ./scripts/check`
