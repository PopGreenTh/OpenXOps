# WI-0106 — `xops report`

## Outcome

`xops report` turns any records into a table on screen, CSV files or an Excel workbook (D-011).
Writing to an existing file appends, so repeated runs build one growing table.

## Contracts

`xops report table|csv|excel [--out PATH] [--kinds K1,K2] [--from-<kind> FILE]...`

- Reads any records, from the pipe and from any `--from-<kind>` flags (one per kind it is given).
- **One table per `kind`**, in first-seen order; `--kinds` limits which. Columns are the fields
  in first-seen order, `kind` left out.
- **`table`**: aligned text to stdout, one titled block per kind; numbers with 2 decimals. With
  `--out FILE` the same text is appended to that file instead.
- **`csv`**: `--out DIR` (required); `DIR/<kind>.csv`, UTF-8, RFC 4180.
- **`excel`**: `--out FILE.xlsx` (required); one sheet per kind, named by it. Pure Go library,
  no CGO.
- **Appending.** A new file or sheet gets a header row, then rows. An existing one gets only
  rows, at the end. A record field missing from the existing header stops the command with an
  error naming the fields, and nothing is written; a header column the records lack is empty.
- Report never sums or groups (that is Analyze).

## WO-0106-A — Table, CSV, Excel with appending

**Assignee:** open
**Depends on:** WO-0101-A (may run in parallel with any other Work Order)
**Packages:** `xops/report/`, `xops/main.go`, `Taskfile.yml`, `.vscode/launch.json`,
`README.md`, `go.mod`, `go.sum`

**Goal.** The three formats with appending.

**In scope.**
- The command and its formats; a task and launch entry per format on a fixture file.
- Golden-file tests per format on `xops/report/testdata/mixed.jsonl` (three kinds); appending
  twice gives one header and the rows twice; a new field stops the append and writes nothing.

**Out of scope.** Markdown, Power BI, REST. Any summing.

**Done when.**
- The tests above pass on Linux (CI's Linux job; Windows is optional, D-026).
- `xops report excel --out x.xlsx --from-cost …` run twice leaves one workbook whose sheets hold
  both runs' rows under one header (opened in Excel or LibreOffice; said in the PR).

**Verify.** `go run ./scripts/check`
