# WI-0302 — `finops sense plan-lines`

## Outcome

Plan and manual forecast lines, edited in Directus, become `plan` and `forecast` records that
OpenXOps can trust: only the tenant's rows, only URNs the taxonomy holds, one line per month,
node and type ([D-062](../DECISIONS.md#d-062), [D-063](../DECISIONS.md#d-063)). Each read can be
saved, so a month is compared with the plan as it stood before the month began
([D-065](../DECISIONS.md#d-065)).

## Contracts

### Directus collection ([D-069](../DECISIONS.md#d-069))

One collection, `finops_lines` by default, one row per month and node:

| Field | Type | |
|---|---|---|
| `id` | Directus primary key | becomes `line_id` |
| `tenant` | string | required |
| `line_type` | `budget`, `change` or `forecast` | `budget` and `change` become `plan` records with that `plan_type`; `forecast` becomes a `forecast` record with `source` `manual` |
| `month_start` | date, the first of a month | |
| `business_unit_urn`, `cost_center_urn`, `project_urn`, `app_urn`, `environment_urn`, `account_urn` | string, may be empty | chosen from the lookup (WI-0305) |
| `service_category` | string, may be empty | a category code from the lookup; needs `account_urn` |
| `service` | string, may be empty | the service name as the cost source spells it; needs `service_category` |
| `currency` | string | |
| `metric` | string | default `UnblendedCost` |
| `amount` | number | the month's amount |
| `note` | string | optional, passed through |

`docs/user/directus.md` describes the collection, and a read-only role whose permission filters
`tenant` on the server.

### `finops sense plan-lines --tenant ID --taxonomy FILE [--from-node FILE] [--directus-url URL] [--collection NAME] [--from-file FILE] [--mock] [--from-month YYYY-MM] [--months N] [--save] [--history-dir DIR]`

- **Request:** `GET <url>/items/<collection>?filter[tenant][_eq]=<tenant>&limit=500&page=N`, paged
  until a page comes back short. Only `GET` is ever sent (release criterion 7). Every request
  goes through the retry wrapper of the new `directus/` client package (like `aws/`).
- **URL and token:** `--directus-url` or `OPENXOPS_DIRECTUS_URL`; the token from
  `OPENXOPS_DIRECTUS_TOKEN`, sent as `Authorization: Bearer`, never logged. A missing URL or
  token is an error, except with `--from-file` or `--mock`.
- **Tenant check:** a returned row whose `tenant` is not `--tenant`, or is empty, stops the
  command, naming the row `id`. The server filter is a convenience, not the guard.
- **Offline:** `--from-file` reads a Directus export, `.json` (an array, or `{ "data": [...] }`) or
  `.csv` with the field names above. `--mock` reads `finops/sense/planlines/testdata/mock-lines.json`.
- **Horizon:** `--from-month` (default the current month, from a clock tests can set) and
  `--months` (default 12). A row outside it is dropped with `plan-line-out-of-horizon` (info).
- **`--save`** keeps the output in the history store as `plan-lines` (WI-0201; amends D-039).

### Service URNs

`plan-lines` builds the two service URNs from the row, as `map` does for `cost` (WI-0301):
`service_category_urn` = `<account_urn>:geo:global:sct:<service_category>`, and `service_urn` =
`<service_category_urn>:svc:<slug of service>`. A row with `service_category` and no
`account_urn`, or `service` and no `service_category`, is `plan-line-hierarchy`. So is a
`service` the taxonomy files under another category.

### Checks, each a finding; the row is dropped

| Code | Severity | When | Subject |
|---|---|---|---|
| `plan-line-no-node` | error | every URN column, `service_category` and `service` are empty | the row id |
| `plan-line-urn-unknown` | error | a URN names no node: the taxonomy for business and project nodes, the `--from-node` file for `lzc`, `org` and `acc` nodes (without it, cloud URNs are only checked for form); a `service_category` the taxonomy lacks | the row id and the URN or code |
| `plan-line-hierarchy` | error | the columns disagree with the taxonomy or with each other: an App not under the Project given, a CostCenter that isn't the App's, a BusinessUnit that isn't the Project's, a deeper URN not under a shallower one | the row id |
| `plan-line-duplicate` | error | two rows with the same `line_type`, month, node, metric and currency; both are dropped | both row ids |
| `plan-line-bad-value` | error | an unknown `line_type`, a `month_start` that isn't the first of a month, a non-numeric `amount` | the row id |
| `plan-line-out-of-horizon` | info | the month is outside `--from-month` + `--months` | the row id |

Dropping rows doesn't fail the command: it exits 0, and the findings say which rows were dropped
and why.

### Output

`plan` and `forecast` records as *Kinds* in DELIVERABLE.md says, sorted by `period`, `node_urn`,
`line_type`, then findings. The columns above the node are filled from the taxonomy when the row
left them empty, so every record has its full chain.

## WO-0302-A — `plan-lines`

**Assignee:** open
**Depends on:** WO-0301-A, WO-0201-A (`--save`)
**Packages:**
- `finops/sense/planlines/` (new), with the command wired in `finops/sense/commands.go`;
- `directus/` (new: the HTTP client behind an interface, `GET` only, with a retry wrapper);
- `record/schema/` (the `plan` and `forecast` kinds);
- `docs/user/` (`commands.md`, `records.md`, `findings.md`, new `directus.md`), `README.md`,
  `Taskfile.yml`, `.vscode/launch.json`

**Goal.** The command as specified.

**In scope.**
- The client, paging, the tenant check, `--from-file` (JSON and CSV), `--mock`, the horizon and
  `--save`.
- Every check in the table, each with a test.
- Tests against a local `httptest` server, never a real Directus:
  - paging over two pages;
  - a row of another tenant stopping the command;
  - a missing token;
  - only `GET` sent.
- The mock lines: a budget per Project for 12 months, a `change` line for a new Project from
  M+1 with no account, a manual forecast for one App, and one bad row of each check.

**Out of scope.** Writing to Directus. Creating the collection (the doc says how).

**Done when.**
- The tests above pass.
- `plan-lines --mock --tenant testtnt` prints `plan` and `forecast` records and one finding per
  bad mock row, and no bad row reaches the output (shown in the PR; release criterion 5).
- `go run -tags=no_duckdb_arrow ./scripts/check` all PASS, with `docs PASS`.

**Verify.** `go run -tags=no_duckdb_arrow ./scripts/check`
