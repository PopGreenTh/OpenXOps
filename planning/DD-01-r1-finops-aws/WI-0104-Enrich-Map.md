# WI-0104 — `finops enrich map` and the taxonomy

## Outcome

The taxonomy is one YAML file, edited through PRs. `finops enrich validate` checks it with no
cloud access. `finops enrich map` resolves every account to its LZ, one Environment and one or
more Apps (D-015), adds that to the account's cost, and says why for everything it cannot map.

## Contracts

### Taxonomy file

```yaml
business_units: [{ code: BU-RETAIL, name: Retail }]
cost_centers:   [{ code: CC-ECOM, name: E-commerce, business_unit: BU-RETAIL }]
projects:       [{ code: PRJ-SHOP, name: Online shop, business_unit: BU-RETAIL }]
apps:
  - code: APP-CHK
    name: Checkout
    project: PRJ-SHOP
    cost_center: CC-ECOM
    environments: [{ code: ENV-PROD, name: Production }, { code: ENV-DEV, name: Development }]
lzs:
  - { code: LZ-MAIN, name: Main, csp: aws, currency: USD, management_account: "111111111111",
      default: { app: APP-CHK, environment: ENV-DEV } }        # default is optional
ou_links:                                                      # app and/or environment
  - { lz: LZ-MAIN, ou: /Workloads/Checkout, app: APP-CHK }
  - { lz: LZ-MAIN, ou: /Workloads/Checkout/Prod, environment: ENV-PROD }
accounts:                                                      # apps and/or environment
  - { account: "222222222222", environment: ENV-PROD, apps: [APP-CHK, APP-PAY] }
defaults: { app: APP-CHK, environment: ENV-DEV }               # optional
tags: { app: App, project: Project }                           # optional; these are the defaults
```

Codes are unique within their list. `-UNALLOC`, `-UNSPEC` and `-INVALID` endings are reserved
for system nodes and rejected in the file.

### `finops enrich validate --taxonomy FILE`

Prints `finding` records; exits 1 if any has severity `error`. Errors: unknown keys; missing
required fields; duplicate codes; a reference to an unknown code; a reserved ending; an account
entry or LZ/global default whose Environment is not defined under (each of) its App(s); two
`ou_links` for the same `lz` and `ou` setting the same field. Warnings: an unused BusinessUnit,
CostCenter or Project.

### Resolution (D-015)

For each account, using its organization's `ou` and `account` records:

0. **LZ**: the taxonomy LZ whose `management_account` equals the record's. None → LZ
   `LZ-UNALLOC`, App `APP-UNALLOC`, Environment `ENV-UNALLOC`, rules `unalloc`, finding
   `lz-unknown` (subject: org ID), steps 1–4 skipped.
1. **OU links** of that LZ: an `ou` starting with `/` is a path, else an OU ID. One that matches
   no OU → ignored, finding `ou-not-found` (subject: the value as written). An `app` on an OU with
   no child OUs → ignored, finding `app-on-leaf-ou`; its `environment` still applies.
2. **App**, first answer wins: the account entry's `apps` (`app_rule` `account`) › the deepest OU
   on the account's chain with an `app` link (`ou:<path>`) › the LZ `default.app`
   (`lz-default`) › the global `defaults.app` (`default`) › `APP-UNALLOC` (`unalloc`, finding
   `account-unmapped`, subject: account ID).
3. **Environment**, separately, same order and rule names. Nothing found → `ENV-UNSPEC`
   (`unspec`, finding `environment-unresolved`) if the App resolved, `ENV-UNALLOC` (`unalloc`)
   if not.
4. **Environment under App**: a resolved Environment not defined under the App →
   `ENV-UNSPEC`, `environment_rule` `unspec`, finding `environment-not-in-app` (subject
   `<account>/<app>`). Under `APP-UNALLOC` the Environment is kept as resolved.
5. **Derived**: Project = the App's; BusinessUnit = the Project's; CostCenter = the App's. Under
   `APP-UNALLOC`: `PRJ-UNALLOC`, `BU-UNALLOC`, `CC-UNALLOC`.
6. An account entry for an account not in the input → finding `account-not-discovered`.

### `finops enrich map --taxonomy FILE [--from-accounts FILE] [--from-cost FILE]`

Reads `ou` and `account` (pipe or `--from-accounts`, required), `cost` and `project_tag_cost`
(pipe or `--from-cost`, optional). Writes, in order:

1. the `ou` and `account` records, unchanged;
2. one `allocation` record per account and App: `account_id`, `account_name`, `lz`, `ou_path`,
   `project`, `app`, `environment`, `business_unit`, `cost_center`, `app_rule`,
   `environment_rule`, `shared` (true when the account serves two or more Apps);
3. each `cost` record with `lz`, `environment`, `environment_rule` and `shared` added; when not
   shared, also `project`, `app`, `business_unit`, `cost_center`, `app_rule`. A shared account's
   cost keeps those five fields `""` for `share` to fill (WI-0105). Cost of an account not in the
   input: `lz` from its management account (step 0), every other level `UNALLOC`, rules
   `unalloc`, `shared` false, finding `account-not-in-inventory` (subject: account ID);
4. findings. **Project tag check (D-009):** a `project_tag_cost` record with a non-empty
   `project_tag` that is not the account's Project (or one of them, for a shared account) →
   finding `project-tag-mismatch`, warning, subject `<account>/<tag value>`, message naming
   both. `project_tag_cost` records are not passed on.

Cost amounts are never changed by `map`.

### Reference cases

`finops/enrich/testdata/cases/map/*.yaml`, written by Claude. Each has `taxonomy`, input
`records`, and `expect` with `allocations` (compared on the fields listed, sorted by
`account_id` then `app`), `costs` (same) and `findings` (`code` and `subject`, as a set). The
test reads every case in place.

### Demo taxonomy

`finops/enrich/testdata/demo-taxonomy.yaml`, for the mock organization (WI-0102): two
BusinessUnits, two Projects, `APP-CHK` and `APP-PAY` with Prod and NonProd Environments linked
by OU, the shared account serving both, the Security accounts left unmapped. It passes
`validate` with no error.

## WO-0104-A — Taxonomy, `validate` and `map`

**Assignee:** open
**Depends on:** WO-0101-A (may run in parallel with WO-0102-A and WO-0103-A)
**Packages:** `finops/enrich/`, `finops/main.go`, `Taskfile.yml`, `.vscode/launch.json`,
`README.md`

**Goal.** `validate` and `map` as specified.

**In scope.**
- Taxonomy model and YAML loader (strict: unknown keys are errors), validation, resolution,
  the command, the demo taxonomy.
- Tasks and launch entries for `validate` and `map` on the demo taxonomy and fixture files.
- Tests: every map reference case; a table of validation errors, one per rule.
- README: the taxonomy format and both commands.

**Out of scope.** Splitting shared cost (WI-0105).

**Done when.**
- Every `cases/map/` case passes, unchanged.
- `finops enrich validate --taxonomy finops/enrich/testdata/demo-taxonomy.yaml` exits 0.
- `xops sense aws-accounts --mock | finops enrich map --taxonomy …/demo-taxonomy.yaml` prints an
  `allocation` for every mock account, and the Security accounts get `account-unmapped`.

**Verify.** `go run ./scripts/check`
