# WI-0305 — `xops report urn-lookup`

## Outcome

Directus offers only valid URNs ([D-062](../DECISIONS.md#d-062)). This command writes them as one
list, so the plan lines people enter match the URNs `map` puts on actuals. The business and
project trees come from the taxonomy, the cloud tree from the `node` records `lz-tree` writes.

## Contracts

### `xops report urn-lookup --tenant ID --taxonomy FILE [--from-node FILE] [--format csv|json] --out FILE` (planner default, [Q-010](../DECISIONS.md#q-010) item 5)

- **Rows**, one per node, in this order:
  - the business tree: `bzu`, `cct`;
  - the project tree: `prj`, `app`, `env`;
  - the service categories: `sct`, one per category in `service_categories`, plus `other`;
  - the cloud tree from `--from-node`: `lzc`, `org`, `acc`.

  Without `--from-node`, only the LZ from the taxonomy is listed for the cloud tree.
- **Columns:** `tenant`, `urn`, `type`, `code`, `name`, `parent_urn`, `ref_bu_urn` (a Project's
  BusinessUnit), `ref_cct_urn` (an App's CostCenter), `ref_id` (a cloud node's AWS id), `alias`.
  A column that doesn't apply is empty.
- An `sct` row has no account, so its `urn` is `""` and its `code` is the category. A line
  names it in its `service_category` field, and `plan-lines` builds the URN from the line's
  account (WI-0302).
- `--format` is `csv` (default) or `json` (an array of objects with the same keys). Report
  writes a file, so this lives in `xops report` (D-006).
- `node` records of another tenant stop the command, as WI-0109 says.

## WO-0305-A — URN lookup

**Assignee:** open
**Depends on:** WO-0301-A (`service_categories`)
**Packages:** `xops/report/` (the command in `commands.go`; its writer in a new subpackage
`xops/report/lookup/`), `xops/report/testdata/`, `docs/user/` (`commands.md`, `directus.md`),
`Taskfile.yml`, `.vscode/launch.json`

**Goal.** The command as specified.

**In scope.** Both formats; with and without `--from-node`; golden files from the demo taxonomy
and the demo tree.

**Out of scope.** Pushing the list to Directus (people import the file).

**Done when.**
- The golden files match, and every `urn` in them passes the URN rule (WI-0109).
- Every URN in the lookup from the mock tree is accepted by `plan-lines` when used in a line
  (shown in the PR).
- `go run -tags=no_duckdb_arrow ./scripts/check` all PASS, with `docs PASS`.

**Verify.** `go run -tags=no_duckdb_arrow ./scripts/check`
